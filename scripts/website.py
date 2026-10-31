#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Publish reviewed current-main website bytes without mutating product releases."""
import argparse
import base64
import hashlib
import http.client
import importlib.util
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tarfile
import tempfile
import time
from urllib.parse import urlsplit

sys.path.insert(0, str(Path(__file__).resolve().parent))
import qualification_timing as timing


ROOT = Path(__file__).resolve().parents[1]
REPOSITORY = 'shruggietech/insonic'
ENDPOINT = 'repos/' + REPOSITORY


def sibling(name):
    spec = importlib.util.spec_from_file_location('website_' + name, Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module)
    return module


pages, release, public, processes = map(sibling, ('pages', 'release', 'public-verification', 'process_tree'))


def write(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2) + '\n', encoding='utf-8', newline='\n')


def file_entry(path):
    if path.is_symlink() or not path.is_file() or path.stat().st_size > 64 << 20:
        raise ValueError('website archive is missing or excessive')
    data = path.read_bytes()
    return {'name': path.name, 'size_bytes': len(data), 'sha256': hashlib.sha256(data).hexdigest()}


class API:
    """Small bounded API responses; the CLI parent owns the aggregate deadline."""
    def __init__(self, token, writable=False): self.token, self.writable = token, writable

    def call(self, method, endpoint, value=None):
        if method != 'GET' and not (self.writable and method == 'POST' and endpoint == ENDPOINT + '/pages/builds'):
            raise ValueError('website API operation is outside its authority')
        if not endpoint.startswith(ENDPOINT + '/') or '\n' in endpoint or '\r' in endpoint:
            raise ValueError('website API endpoint is invalid')
        headers = {'Accept': 'application/vnd.github+json', 'X-GitHub-Api-Version': '2022-11-28', 'User-Agent': 'insonic-website'}
        if self.token: headers['Authorization'] = 'Bearer ' + self.token
        for attempt in range(3 if method == 'GET' else 1):
            connection = http.client.HTTPSConnection('api.github.com', timeout=timing.REQUEST_SECONDS)
            try:
                connection.request(method, '/' + endpoint, body=json.dumps(value) if value is not None else None, headers=headers)
                response = connection.getresponse(); data = response.read((4 << 20) + 1)
                if len(data) > 4 << 20: raise ValueError('website API response exceeds limit')
                if response.status == 404: return None
                if response.status in (429, 500, 502, 503, 504) and method == 'GET' and attempt < 2:
                    time.sleep(attempt + 1); continue
                if response.status not in (200, 201, 202): raise ValueError('website API failed (HTTP ' + str(response.status) + ')')
                return json.loads(data)
            finally: connection.close()


def green_main(api, revision):
    pages.commit_sha(revision)
    ref = api.call('GET', ENDPOINT + '/git/ref/heads/main')
    if not ref or ref.get('object', {}).get('type') != 'commit' or ref['object'].get('sha') != revision:
        raise ValueError('website source is no longer current main')
    response = api.call('GET', ENDPOINT + '/actions/workflows/ci.yml/runs?branch=main&event=push&head_sha=' + revision + '&per_page=100')
    runs = response.get('workflow_runs', []) if isinstance(response, dict) else []
    if (not runs or len(runs) > 100 or response.get('total_count') != len(runs)
            or any(not isinstance(r, dict) or type(r.get('id')) is not int for r in runs)):
        raise ValueError('website exact-source CI inventory is unavailable or incomplete')
    newest = max(runs, key=lambda r: (r['id'], r.get('run_attempt', 0)))
    if (newest.get('head_sha') != revision or newest.get('head_branch') != 'main' or newest.get('event') != 'push'
            or newest.get('path') != '.github/workflows/ci.yml' or newest.get('status') != 'completed' or newest.get('conclusion') != 'success'):
        raise ValueError('newest exact-main CI run is not green')
    attempt = newest.get('run_attempt')
    if type(attempt) is not int or attempt < 1: raise ValueError('newest CI attempt identity is unavailable')
    inventory = api.call('GET', ENDPOINT + f'/actions/runs/{newest["id"]}/attempts/{attempt}/jobs?per_page=100')
    jobs = inventory.get('jobs', []) if isinstance(inventory, dict) else []
    if (not jobs or len(jobs) > 100 or inventory.get('total_count') != len(jobs)
            or any(not isinstance(job, dict) for job in jobs)
            or any(not any(job.get('name') == name and job.get('status') == 'completed' and job.get('conclusion') == 'success' for job in jobs) for name in ('foundation', 'docs'))
            or any(job.get('status') != 'completed' or job.get('conclusion') not in ('success', 'skipped', 'neutral') for job in jobs)):
        raise ValueError('newest CI attempt lacks required successful qualification jobs')
    return newest['id']


def package_url(url, initial=False):
    p = urlsplit(url)
    if (p.scheme != 'https' or p.username is not None or p.password is not None or p.port not in (None, 443)
            or p.fragment or any(ord(c) < 33 or ord(c) > 126 for c in url) or '\\' in url):
        raise ValueError('official package URL is invalid')
    if initial:
        if p.hostname != 'github.com' or not p.path.startswith('/' + REPOSITORY + '/releases/download/') or p.query:
            raise ValueError('official package URL differs from repository')
    elif p.hostname not in ('release-assets.githubusercontent.com', 'objects.githubusercontent.com'):
        raise ValueError('official package redirect left approved CDN')
    return p


def package_response(url, *, head=False, limit=16 << 20):
    package_url(url, initial=True)
    for index in range(5):
        p = package_url(url, initial=index == 0)
        connection = http.client.HTTPSConnection(p.hostname, timeout=timing.REQUEST_SECONDS)
        try:
            connection.request('HEAD' if head else 'GET', p.path + ('?' + p.query if p.query else ''), headers={'User-Agent': 'insonic-website', 'Accept-Encoding': 'identity'})
            response = connection.getresponse()
            if response.status in (301, 302, 303, 307, 308):
                url = response.getheader('Location', ''); package_url(url); continue
            if response.status != 200: raise ValueError('official package request failed (HTTP ' + str(response.status) + ')')
            size = response.getheader('Content-Length', '')
            if not size.isdecimal(): raise ValueError('official package response size is unavailable')
            if head: return int(size)
            if int(size) > limit: raise ValueError('official metadata exceeds bound')
            data = response.read(limit + 1)
            if len(data) != int(size) or len(data) > limit: raise ValueError('official metadata is incomplete or excessive')
            return data
        finally: connection.close()
    raise ValueError('official package redirect limit exceeded')


def published_product(api, version, fetch=package_response):
    tag = 'v' + version
    value = api.call('GET', ENDPOINT + '/releases/tags/' + tag)
    if (not value or value.get('draft') is not False or value.get('prerelease') is not False or value.get('tag_name') != tag):
        raise ValueError('website version requires an official existing published release')
    revision = pages.commit_sha(value.get('target_commitish'))
    release.verify_tag(api, tag, revision)
    assets = api.call('GET', ENDPOINT + f'/releases/{value["id"]}/assets?per_page=100')
    if (not isinstance(assets, list) or len(assets) != 10 or len({a.get('name') for a in assets}) != 10):
        raise ValueError('published release requires six package assets and four metadata/documentation assets')
    by_name = {a['name']: a for a in assets}
    for asset in assets:
        release.asset_name(asset['name'])
        expected_url = f'https://github.com/{REPOSITORY}/releases/download/{tag}/' + asset['name']
        if (asset.get('state') != 'uploaded' or type(asset.get('id')) is not int or asset['id'] <= 0
                or type(asset.get('size')) is not int or asset['size'] <= 0
                or not re.fullmatch(r'sha256:[a-f0-9]{64}', asset.get('digest', ''))
                or asset.get('browser_download_url') != expected_url):
            raise ValueError('published release metadata is invalid')
    metadata = {}
    for name in ('release-manifest.json', 'SHA256SUMS'):
        if name not in by_name: raise ValueError('published release manifest/checksums missing')
        asset = by_name[name]; data = fetch(asset['browser_download_url'])
        if len(data) != asset['size'] or 'sha256:' + hashlib.sha256(data).hexdigest() != asset['digest']:
            raise ValueError('published metadata bytes differ from API digest')
        metadata[name] = data
    manifest = json.loads(metadata['release-manifest.json'])
    if (manifest.get('kind') != 'release-candidate' or manifest.get('repository') != REPOSITORY
            or manifest.get('version') != version or manifest.get('schema_version') != version
            or manifest.get('tag') != tag or manifest.get('revision') != revision):
        raise ValueError('published product manifest identity differs')
    expected_names = {f'insonic_{version}_{platform}_{variant}{suffix}' for platform, suffix in [('windows_amd64', '.zip'), ('linux_amd64', '.tar.gz')] for variant in ('cli', 'desktop')}
    expected_names.add(f'insonic_{version}_windows_amd64_installer.msi')
    expected_names.add(f'insonic_{version}_windows_amd64_setup.exe')
    entries = manifest.get('assets', [])
    if len(entries) != 8 or {e.get('name') for e in entries} != expected_names | {'documentation.tar.gz', 'offline-help.tar.gz'}:
        raise ValueError('published product download matrix differs')
    for entry in entries:
        asset = by_name.get(entry['name'], {})
        if asset.get('size') != entry.get('size_bytes') or asset.get('digest') != 'sha256:' + entry.get('sha256', ''):
            raise ValueError('published product asset differs from manifest')
    checksum_entries = entries + [{'name': 'release-manifest.json', 'sha256': hashlib.sha256(metadata['release-manifest.json']).hexdigest()}]
    expected_sums = ''.join(e['sha256'] + '  ' + e['name'] + '\n' for e in sorted(checksum_entries, key=lambda e: e['name'])).encode()
    if metadata['SHA256SUMS'] != expected_sums: raise ValueError('published checksums differ from manifest')
    return {'release_revision': revision, 'release_id': value['id'], 'packages': [
        {'name': name, 'url': by_name[name]['browser_download_url'], 'size_bytes': by_name[name]['size'], 'sha256': by_name[name]['digest'][7:]}
        for name in sorted(expected_names)]}


def immutable_schemas(destination, files):
    for name, digest in destination['entries'].items():
        if name.startswith('schemas/v') and (name not in files or pages.git_blob_sha(files[name]) != digest):
            raise ValueError('website would change or remove a published schema: ' + name)


def qualified(archive, receipt, api):
    revision, version = pages.commit_sha(receipt.get('site_revision')), receipt.get('version')
    if (receipt.get('kind') != 'website-publication' or receipt.get('format') != 1
            or version != release.versions() or receipt.get('archive') != file_entry(archive)):
        raise ValueError('website receipt/archive identity differs')
    ci = green_main(api, revision)
    product = published_product(api, version)
    if any(receipt.get(k) != v for k, v in product.items()) or receipt.get('selected_ci_run') != ci:
        raise ValueError('website receipt source/product authority differs')
    files = pages.archive_files(archive)
    requests = public.inventory(files, version, revision, 'https://insonic.io/', [p['url'] for p in product['packages']])
    return files, requests


def prepare(archive, output, revision, api):
    version = release.versions(); release.exact_source(revision)
    receipt = {'kind': 'website-publication', 'format': 1, 'version': version, 'site_revision': revision,
               'selected_ci_run': green_main(api, revision), 'archive': file_entry(archive), **published_product(api, version)}
    qualified(archive, receipt, api)
    write(output, receipt)
    return receipt


def git(args, cwd, env=None):
    selected = dict(os.environ if env is None else env)
    selected.update(GIT_TERMINAL_PROMPT='0', GCM_INTERACTIVE='Never', GIT_CONFIG_NOSYSTEM='1', GIT_CONFIG_GLOBAL=os.devnull)
    with processes.ProcessTree(['git', '-c', 'core.hooksPath=' + os.devnull, '-c', 'commit.gpgsign=false', *args], cwd=cwd, env=selected) as owned:
        try: out, err = owned.process.communicate(timeout=timing.SUBPROCESS_SECONDS)
        except subprocess.TimeoutExpired:
            owned.kill(); owned.process.communicate(timeout=timing.CLEANUP_SECONDS); raise TimeoutError('owned website Git deadline exceeded') from None
        if owned.process.returncode: raise ValueError('owned website Git operation failed')
        if len(out) > 4 << 20 or len(err) > 4 << 20: raise ValueError('owned website Git output exceeds bound')
        return out.decode('utf-8').strip()


def push_tree(api, destination, files, revision, token, runner=git, on_staged=lambda commit: None):
    # Token is never placed in URLs, argv, on disk or diagnostic output.
    env = {k: v for k, v in os.environ.items() if not re.search(r'TOKEN|SECRET|PASSWORD|CREDENTIAL|ASKPASS|^GIT_', k, re.I)}
    env.update(GIT_CONFIG_COUNT='1', GIT_CONFIG_KEY_0='http.https://github.com/.extraheader',
               GIT_CONFIG_VALUE_0='AUTHORIZATION: basic ' + base64.b64encode(('x-access-token:' + token).encode()).decode())
    with tempfile.TemporaryDirectory(prefix='insonic-website-') as temporary:
        checkout = Path(temporary) / 'pages'
        runner(['clone', '--no-checkout', '--depth=1', '--single-branch', '--branch', 'gh-pages', 'https://github.com/' + REPOSITORY + '.git', str(checkout)], ROOT, env)
        parent = runner(['rev-parse', 'HEAD'], checkout, env)
        if parent != destination['head']: raise ValueError('Pages destination changed before staging')
        runner(['read-tree', '--empty'], checkout, env)
        for name, data in files.items():
            target = checkout / pages.member_path(name); target.parent.mkdir(parents=True, exist_ok=True); target.write_bytes(data)
        runner(['add', '--all'], checkout, env)
        tree = runner(['write-tree'], checkout, env)
        if tree == destination['tree']: staged = parent
        else:
            runner(['-c', 'user.name=insonic website publisher', '-c', 'user.email=noreply@shruggie.tech', 'commit', '-m', 'Publish reviewed website ' + revision], checkout, env)
            staged = pages.commit_sha(runner(['rev-parse', 'HEAD'], checkout, env))
            green_main(api, revision)
            ref = api.call('GET', ENDPOINT + '/git/ref/heads/gh-pages')
            if not ref or ref.get('object', {}).get('sha') != parent: raise ValueError('Pages destination changed before push')
            on_staged(staged)
            try: runner(['push', 'origin', 'HEAD:refs/heads/gh-pages'], checkout, env)
            except (OSError, ValueError, TimeoutError):
                ref = api.call('GET', ENDPOINT + '/git/ref/heads/gh-pages')
                if not ref or ref.get('object', {}).get('sha') != staged: raise
        ref = api.call('GET', ENDPOINT + '/git/ref/heads/gh-pages')
        if not ref or ref.get('object', {}).get('sha') != staged: raise ValueError('Pages changed after publication')
        return staged


def publish(archive, receipt, output, api, token):
    if not token: raise ValueError('website publication token is required')
    qualified(archive, receipt, api)
    destination = pages.preflight(api, REPOSITORY)
    files = pages.prepare_tree(archive, destination, receipt['version'], receipt['site_revision'])
    immutable_schemas(destination, files)
    state = {'phase': 'qualified', 'destination_changed': False, 'site_revision': receipt['site_revision'],
             'release_revision': receipt['release_revision'], 'version': receipt['version'], 'previous_commit': destination['head']}
    write(output, state)
    def attempting(commit):
        state.update(phase='push-attempted', destination_changed='unconfirmed', pages_commit=commit)
        write(output, state)
    staged = push_tree(api, destination, files, receipt['site_revision'], token, on_staged=attempting)
    state.update(phase='pushed', destination_changed=staged != destination['head'], pages_commit=staged); write(output, state)
    # Requesting a build can lose its response; the exact completed build remains authority.
    green_main(api, receipt['site_revision'])
    try: api.call('POST', ENDPOINT + '/pages/builds')
    except (OSError, ValueError): pass
    deadline = time.monotonic() + timing.ARTIFACT_PRODUCER_SECONDS
    while time.monotonic() < deadline:
        ref = api.call('GET', ENDPOINT + '/git/ref/heads/gh-pages')
        if not ref or ref.get('object', {}).get('sha') != staged: raise ValueError('Pages publication conflict')
        build = api.call('GET', ENDPOINT + '/pages/builds/latest')
        if build and build.get('commit') == staged and build.get('status') == 'built':
            state.update(phase='deployed', pages_build='built'); write(output, state); return state
        if build and build.get('commit') == staged and build.get('status') not in ('queued', 'building'):
            raise ValueError('selected Pages deployment failed')
        time.sleep(5)
    raise TimeoutError('selected Pages deployment deadline exceeded')


def verify(archive, receipt, deployment, output, api):
    if receipt.get('archive') != file_entry(archive): raise ValueError('live verification archive differs from qualification')
    if any(deployment.get(k) != receipt.get(k) for k in ('version', 'site_revision', 'release_revision')):
        raise ValueError('deployment source/product identity differs from qualification')
    product = published_product(api, receipt['version'])
    if any(receipt.get(k) != v for k, v in product.items()): raise ValueError('published product changed before live verification')
    commit = pages.commit_sha(deployment.get('pages_commit'))
    value = public.verify(archive, receipt['version'], receipt['site_revision'], 'https://insonic.io/',
                          package_urls=[p['url'] for p in product['packages']], api=api, pages_commit=commit)
    value.update(release_revision=product['release_revision'], phase='content-verified')
    value['content_proof'] = 'downloaded-qualified-export-byte-match'
    write(output, value)
    downloads = []
    for package in product['packages']:
        size = package_response(package['url'], head=True)
        if size != package['size_bytes']: raise ValueError('live package size differs from published metadata')
        downloads.append({**package, 'status': 200, 'verification': 'HEAD-size-and-published-API-digest'})
    public.deployment(api, commit)
    value.update(packages=downloads, phase='links-verified'); write(output, value)
    return value


def archive_tree(directory, output):
    files, total = [], 0
    for path in sorted(directory.rglob('*')):
        name = path.relative_to(directory).as_posix()
        if name == '.git' or name.startswith('.git/'): continue
        pages.member_path(name)
        if path.is_symlink(): raise ValueError('deployed tree has a link')
        if not path.is_file(): continue
        size = path.stat().st_size; total += size
        if size > 16 << 20 or total > 64 << 20 or len(files) >= 10000: raise ValueError('deployed tree exceeds bounds')
        files.append((path, name))
    with tarfile.open(output, 'w:gz') as archive:
        for path, name in files: archive.add(path, arcname=name, recursive=False)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=('source', 'deployment', 'prepare', 'publish', 'verify', 'verify-deployed'))
    parser.add_argument('--archive', type=Path)
    parser.add_argument('--receipt', type=Path)
    parser.add_argument('--deployment', type=Path)
    parser.add_argument('--site-tree', type=Path)
    parser.add_argument('--pages-commit')
    parser.add_argument('--revision')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--worker', action='store_true', help=argparse.SUPPRESS)
    args = parser.parse_args()
    if not args.worker:
        # Includes DNS, API, Git and all descendants in an aggregate owned deadline.
        with processes.ProcessTree([sys.executable, str(Path(__file__).resolve()), *sys.argv[1:], '--worker'], cwd=ROOT, env=dict(os.environ)) as owned:
            try: out, err = owned.process.communicate(timeout=timing.ARTIFACT_PRODUCER_SECONDS + 3 * timing.SUBPROCESS_SECONDS + timing.CLEANUP_SECONDS)
            except subprocess.TimeoutExpired:
                owned.kill(); owned.process.communicate(timeout=timing.CLEANUP_SECONDS); raise TimeoutError('website phase aggregate deadline exceeded') from None
            sys.stdout.buffer.write(out); sys.stderr.buffer.write(err)
            return owned.process.returncode
    api = API(os.environ.get('GH_TOKEN', os.environ.get('GITHUB_TOKEN', '')), args.command == 'publish')
    if args.command == 'source':
        release.exact_source(args.revision)
        write(args.output, {'site_revision': args.revision, 'selected_ci_run': green_main(api, args.revision)})
    elif args.command == 'deployment':
        state = json.loads(args.deployment.read_text(encoding='utf-8'))
        public.deployment(api, state.get('pages_commit'))
        write(args.output, {**state, 'phase': 'verified-complete'})
    elif args.command == 'prepare': prepare(args.archive, args.output, args.revision, api)
    elif args.command == 'verify-deployed':
        public.deployment(api, args.pages_commit)
        with tempfile.TemporaryDirectory(prefix='insonic-live-') as temporary:
            archive = Path(temporary) / 'documentation.tar.gz'; archive_tree(args.site_tree, archive)
            marker = json.loads(pages.archive_files(archive)['documentation-build.json'])
            receipt = {'version': marker['version'], 'site_revision': marker['revision'], 'archive': file_entry(archive), **published_product(api, marker['version'])}
            verify(archive, receipt, {**receipt, 'pages_commit': args.pages_commit}, args.output, api)
    else:
        receipt = json.loads(args.receipt.read_text(encoding='utf-8'))
        if args.command == 'publish': publish(args.archive, receipt, args.output, api, api.token)
        else: verify(args.archive, receipt, json.loads(args.deployment.read_text(encoding='utf-8')), args.output, api)
    print(json.dumps({'website_phase': args.command, 'receipt': args.output.as_posix()}))
    return 0


if __name__ == '__main__':
    try: sys.exit(main())
    except (ValueError, OSError, TimeoutError, KeyError, TypeError):
        print('Website phase failed; inspect retained phase receipts. Credentials and redirect URLs are omitted.', file=sys.stderr)
        sys.exit(1)
