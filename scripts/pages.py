#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Verified original-release transport for the existing legacy Pages site."""
import base64
import hashlib
import json
import os
import tempfile
from pathlib import PurePosixPath
import re
import tarfile
import time
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import qualification_timing as timing

DOMAIN = 'insonic.io'
BRANCH = 'gh-pages'
SHA = re.compile(r'[0-9a-f]{40}')
TREE_REQUEST_BYTES = 1 << 20
BLOB_REQUEST_BYTES = 24 << 20
STATIC_PAYLOAD_BYTES = 128 << 20


class PromotionReceipt:
    """Keep the last complete attempt on disk; never reuse an old attempt path."""
    def __init__(self, path, initial):
        self.path = Path(path)
        self.state = {**initial, 'observations': []}
        self.path.parent.mkdir(parents=True, exist_ok=True)
        with self.path.open('x', encoding='utf-8', newline='\n') as stream:
            json.dump(self.state, stream, indent=2, sort_keys=True)
            stream.write('\n'); stream.flush(); os.fsync(stream.fileno())

    def record(self, event):
        state = {**self.state, **event, 'observations': [*self.state['observations'], event]}
        temporary = None
        try:
            with tempfile.NamedTemporaryFile(mode='w', encoding='utf-8', newline='\n',
                                             dir=self.path.parent, delete=False) as stream:
                temporary = Path(stream.name)
                json.dump(state, stream, indent=2, sort_keys=True)
                stream.write('\n'); stream.flush(); os.fsync(stream.fileno())
            os.replace(temporary, self.path)
            self.state = state
        finally:
            if temporary is not None: temporary.unlink(missing_ok=True)

    def failed(self, error):
        self.record({'phase': 'failed', 'failure': {'phase': self.state.get('phase'),
                     'category': type(error).__name__}})


def commit_sha(value):
    if not isinstance(value, str) or not SHA.fullmatch(value):
        raise ValueError('Pages Git object identity is invalid')
    return value


def member_path(name):
    path = PurePosixPath(name)
    if (not name or path.is_absolute() or path.as_posix() != name or '..' in path.parts
            or '\\' in name or ':' in name or any(p.casefold() == '.git' for p in path.parts)):
        raise ValueError('Pages archive has an unsafe or aliased path')
    return name


def preflight(api, repository):
    endpoint = f'repos/{repository}'
    value = api.call('GET', endpoint + '/pages')
    if (not value or value.get('build_type') != 'legacy'
            or value.get('source') != {'branch': BRANCH, 'path': '/'}
            or value.get('cname') != DOMAIN or value.get('https_enforced') is not True):
        raise ValueError('Pages destination differs from the existing legacy domain/source')
    environment = api.call('GET', endpoint + '/environments/github-pages')
    policy = environment.get('deployment_branch_policy', {}) if environment else {}
    rules = api.call('GET', endpoint + '/environments/github-pages/deployment-branch-policies') if policy.get('custom_branch_policies') is True else None
    branches = [(r.get('name'), r.get('type')) for r in rules.get('branch_policies', [])] if rules else []
    if policy.get('protected_branches') is not False or not rules or rules.get('total_count') != 1 or branches != [(BRANCH, 'branch')]:
        raise ValueError('Pages environment branch policy differs from the existing source')
    ref = api.call('GET', endpoint + '/git/ref/heads/' + BRANCH)
    if not ref or ref.get('object', {}).get('type') != 'commit':
        raise ValueError('Pages branch commit is unavailable')
    head = commit_sha(ref['object']['sha'])
    return destination_at(api, repository, head)


def destination_at(api, repository, head):
    endpoint = f'repos/{repository}'
    commit_sha(head)
    commit = api.call('GET', endpoint + '/git/commits/' + head)
    tree_sha = commit_sha(commit.get('tree', {}).get('sha') if commit else None)
    tree = api.call('GET', endpoint + '/git/trees/' + tree_sha + '?recursive=1')
    if not tree or tree.get('truncated') is not False or not isinstance(tree.get('tree'), list):
        raise ValueError('Pages destination tree is incomplete')
    entries = {}
    for item in tree['tree']:
        if item.get('type') == 'tree': continue
        name = member_path(item.get('path', ''))
        if item.get('type') != 'blob' or item.get('mode') not in ('100644', '100755') or name in entries:
            raise ValueError('Pages destination tree contains unsupported entries')
        entries[name] = commit_sha(item.get('sha'))
    if not {'CNAME', '.nojekyll'}.issubset(entries):
        raise ValueError('Pages destination domain/static markers are missing')
    blob = api.call('GET', endpoint + '/git/blobs/' + entries['CNAME'])
    if not blob or blob.get('encoding') != 'base64' or len(blob.get('content', '')) > 1024:
        raise ValueError('Pages domain marker is invalid')
    try:
        domain = base64.b64decode(''.join(blob['content'].split()), validate=True).decode('ascii').strip()
    except (ValueError, UnicodeError):
        raise ValueError('Pages domain marker is invalid') from None
    if domain != DOMAIN: raise ValueError('Pages domain marker differs from destination')
    return {'repository': repository, 'domain': DOMAIN, 'branch': BRANCH, 'head': head, 'tree': tree_sha, 'entries': entries}


def archive_files(archive_path):
    """Read a confined, bounded regular-file static archive without extraction."""
    files, aliases, total = {}, set(), 0
    with tarfile.open(archive_path, 'r:gz') as archive:
        for member in archive:
            name = member_path(member.name)
            if (not member.isfile() or name.casefold() in aliases or member.size < 0
                    or member.size > 16 << 20 or len(files) >= 10000):
                raise ValueError('Pages archive contains unsupported or excessive members')
            total += member.size
            if total > STATIC_PAYLOAD_BYTES: raise ValueError('Pages archive exceeds its bounded static payload')
            aliases.add(name.casefold())
            data = archive.extractfile(member).read(member.size + 1)
            if len(data) != member.size: raise ValueError('Pages archive member size differs')
            files[name] = data
    return files


def prepare_tree(archive_path, destination, version, revision):
    commit_sha(revision)
    files = archive_files(archive_path)
    aliases, total = {name.casefold() for name in files}, sum(map(len, files.values()))
    try:
        identity = json.loads(files['release-documentation.json'])
        marker = json.loads(files['documentation-build.json'])
    except (KeyError, ValueError, UnicodeError):
        raise ValueError('Pages archive identity is unavailable') from None
    if (identity != {'version': version, 'revision': revision, 'kind': 'documentation'}
            or marker != {'version': version, 'revision': revision, 'source_dirty': False, 'documentation_target': 'release'}
            or not {'index.html', f'docs/v{version}/index.html'}.issubset(files)):
        raise ValueError('Pages requires matching complete release-target documentation')
    if destination['domain'] != DOMAIN or destination['branch'] != BRANCH:
        raise ValueError('Pages destination differs from the elected domain/source')
    if 'CNAME' in files and files['CNAME'].strip() != DOMAIN.encode():
        raise ValueError('Pages archive domain differs from the existing destination')
    if '.nojekyll' in files and files['.nojekyll']:
        raise ValueError('Pages archive static marker differs')
    historical = {name for name in destination['entries'] if name.startswith(('docs/v', 'schemas/v'))}
    for name in sorted(historical - files.keys()):
        # Next's segment export uses both nested paths and flat filenames
        # across renderer versions. Keep the old URL only when its exact
        # counterpart exists beside the same verified public page.
        parts = PurePosixPath(name).parts
        segment = next((index for index, part in enumerate(parts) if part.startswith('__next.')), None)
        replacement = None
        if name.startswith('docs/v') and segment is not None and segment >= 2:
            directory = '/'.join(parts[:segment])
            if directory + '/index.html' in files:
                if segment < len(parts) - 1 and parts[-1] == '__PAGE__.txt':
                    replacement = directory + '/' + '.'.join(parts[segment:])
                elif segment == len(parts) - 1 and parts[-1].endswith('.__PAGE__.txt'):
                    tokens = parts[-1].removesuffix('.__PAGE__.txt').split('.')
                    if len(tokens) >= 2 and all(tokens):
                        replacement = directory + '/' + '.'.join(tokens[:2]) + '/' + '/'.join(tokens[2:] + ['__PAGE__.txt'])
        if replacement not in files:
            raise ValueError('Pages archive would remove historical version routes')
        if name.casefold() in aliases:
            raise ValueError('Pages historical compatibility alias is ambiguous')
        total += len(files[replacement])
        if total > STATIC_PAYLOAD_BYTES or len(files) >= 10000:
            raise ValueError('Pages historical aliases exceed the bounded static payload')
        files[name] = files[replacement]
        aliases.add(name.casefold())
    files['CNAME'], files['.nojekyll'] = (DOMAIN + '\n').encode(), b''
    return files


def git_blob_sha(data):
    return hashlib.sha1(b'blob ' + str(len(data)).encode() + b'\0' + data).hexdigest()


def verify_staged_tree(tree, files):
    """Require the complete elected inventory before making a commit public."""
    if not tree or tree.get('truncated') is not False or not isinstance(tree.get('tree'), list):
        raise ValueError('Pages staged tree inventory is incomplete')
    expected = {name: (git_blob_sha(data), len(data)) for name, data in files.items()}
    directories = {parent.as_posix() for name in files for parent in PurePosixPath(name).parents if parent.as_posix() != '.'}
    seen, blobs, folders = set(), {}, set()
    for item in tree['tree']:
        name = member_path(item.get('path', ''))
        if name in seen: raise ValueError('Pages staged tree inventory has duplicate paths')
        seen.add(name)
        if item.get('type') == 'tree' and item.get('mode') == '040000':
            commit_sha(item.get('sha'))
            folders.add(name)
        elif item.get('type') == 'blob' and item.get('mode') == '100644' and type(item.get('size')) is int:
            blobs[name] = (item.get('sha'), item['size'])
        else: raise ValueError('Pages staged tree inventory has unsupported entries')
    if blobs != expected or folders != directories:
        raise ValueError('Pages staged tree inventory differs from original files')


def ref_head(value):
    if value is None: return None
    obj = value.get('object') if isinstance(value, dict) else None
    if not isinstance(obj, dict) or obj.get('type') != 'commit':
        raise ValueError('Pages branch identity is invalid')
    return commit_sha(obj.get('sha'))


def verify_commit(request, endpoint, staged, tree, previous, previous_tree):
    commit = request('GET', endpoint + '/git/commits/' + commit_sha(staged))
    if not isinstance(commit, dict) or commit.get('tree', {}).get('sha') != commit_sha(tree):
        raise ValueError('Pages staged commit tree differs')
    if staged == previous:
        if tree != previous_tree: raise ValueError('Pages unchanged commit tree differs')
    else:
        parents = commit.get('parents')
        if (not isinstance(parents, list) or len(parents) != 1
                or not isinstance(parents[0], dict) or parents[0].get('sha') != previous):
            raise ValueError('Pages staged commit parent differs')


def reconcile_tree(api, repository, files, attempt):
    """Read-only proof for an already completed exact deployment."""
    endpoint = f'repos/{repository}'
    staged, tree = commit_sha(attempt.get('pages_commit')), commit_sha(attempt.get('pages_tree'))
    previous, previous_tree = commit_sha(attempt.get('previous_commit')), commit_sha(attempt.get('previous_tree'))
    def read(method, route):
        if method != 'GET': raise ValueError('Pages reconciliation permits GET only')
        return api.call(method, route)
    verify_commit(read, endpoint, staged, tree, previous, previous_tree)
    verify_staged_tree(read('GET', endpoint + '/git/trees/' + tree + '?recursive=1'), files)
    if ref_head(read('GET', endpoint + '/git/ref/heads/' + BRANCH)) != staged:
        raise ValueError('Pages reconciliation branch differs from staged commit')
    build = read('GET', endpoint + '/pages/builds/latest')
    if not build or build.get('status') != 'built' or build.get('commit') != staged:
        raise ValueError('Pages reconciliation requires the exact completed build')
    if ref_head(read('GET', endpoint + '/git/ref/heads/' + BRANCH)) != staged:
        raise ValueError('Pages reconciliation branch changed')
    return {'pages_commit': staged, 'pages_build': 'built', 'domain': DOMAIN, 'reconciled': True}


def promote_tree(api, repository, destination, files, version, revision, *, sleep=time.sleep, clock=time.monotonic, observe=lambda event: None):
    if destination['repository'] != repository:
        raise ValueError('Pages repository election differs')
    started = clock()
    endpoint = f'repos/{repository}'
    deadline = started + timing.ARTIFACT_PRODUCER_SECONDS
    last_write = None

    def request(method, route, value=None, *, limit=BLOB_REQUEST_BYTES):
        nonlocal last_write
        if clock() >= deadline: raise ValueError('Pages build deadline exceeded')
        if value is not None and len(json.dumps(value).encode()) > limit:
            raise ValueError('Pages request exceeds its bounded payload')
        # GitHub recommends at least one second between mutative requests.
        # Staging more batches must not trade the payload failure for throttling.
        if method in ('POST', 'PATCH') and last_write is not None:
            delay = max(0, last_write + 1 - clock())
            if delay >= deadline - clock(): raise ValueError('Pages build deadline exceeded')
            if delay: sleep(delay)
        remaining = deadline - clock()
        if remaining <= 0: raise ValueError('Pages build deadline exceeded')
        if method in ('POST', 'PATCH'): last_write = clock()
        result = api.call(method, route, value, timeout=min(timing.SUBPROCESS_SECONDS, remaining))
        if clock() >= deadline: raise ValueError('Pages build deadline exceeded')
        return result

    entries, uploaded, tree_sha = [], set(), None
    # Reserve the largest envelope, including a later base tree and separators.
    envelope = len(json.dumps({'tree': [], 'base_tree': '0' * 40}).encode())
    batch_size = envelope

    def flush():
        nonlocal entries, batch_size, tree_sha
        value = {'tree': entries}
        if tree_sha is not None: value['base_tree'] = tree_sha
        tree = request('POST', endpoint + '/git/trees', value, limit=TREE_REQUEST_BYTES)
        tree_sha = commit_sha(tree.get('sha') if tree else None)
        entries, batch_size = [], envelope

    for name, data in sorted(files.items()):
        item = {'path': name, 'mode': '100644', 'type': 'blob'}
        digest = git_blob_sha(data)
        if destination['entries'].get(name) == digest or digest in uploaded:
            item['sha'] = digest
        else:
            try: text = data.decode('utf-8')
            except UnicodeDecodeError: text = None
            if text is not None: item['content'] = text
            if text is None or envelope + len(json.dumps(item).encode()) > TREE_REQUEST_BYTES:
                item.pop('content', None)
                blob = request('POST', endpoint + '/git/blobs', {'encoding': 'base64', 'content': base64.b64encode(data).decode('ascii')})
                if commit_sha(blob.get('sha') if blob else None) != digest:
                    raise ValueError('Pages staged blob identity differs from original bytes')
                uploaded.add(digest)
                item['sha'] = digest
        size = len(json.dumps(item).encode())
        if entries and batch_size + 2 + size > TREE_REQUEST_BYTES: flush()
        batch_size += (2 if entries else 0) + size
        entries.append(item)
    flush()
    verify_staged_tree(request('GET', endpoint + '/git/trees/' + tree_sha + '?recursive=1'), files)
    if tree_sha == destination['tree']:
        staged = destination['head']
    else:
        commit = request('POST', endpoint + '/git/commits', {'message': f'Publish insonic {version} documentation ({revision})', 'tree': tree_sha, 'parents': [destination['head']]})
        staged = commit_sha(commit.get('sha') if commit else None)
    observe({'phase': 'staged', 'pages_commit': staged, 'pages_tree': tree_sha})
    verify_commit(request, endpoint, staged, tree_sha, destination['head'], destination['tree'])
    if staged != destination['head']:
        observe({'phase': 'update-attempted', 'destination_changed': 'unconfirmed'})
        try:
            request('PATCH', endpoint + '/git/refs/heads/' + BRANCH, {'sha': staged, 'force': False})
        except (OSError, ValueError) as error:
            observe({'phase': 'update-attempted', 'event': 'update-response-unavailable', 'category': type(error).__name__})

    def confirm_head():
        for attempt in range(3):
            try:
                ref = request('GET', endpoint + '/git/ref/heads/' + BRANCH)
            except (OSError, ValueError) as error:
                observe({'phase': 'readback', 'event': 'ref-response-unavailable', 'category': type(error).__name__})
                ref = None
            # Parsing is outside the retry handler: malformed identity fails.
            head = ref_head(ref)
            observe({'phase': 'readback', 'observed_head': head})
            if head == staged: return
            if head is not None and head != destination['head']:
                raise ValueError('Pages concurrent branch update detected')
            if attempt < 2:
                remaining = deadline - clock()
                if remaining <= 0: raise ValueError('Pages build deadline exceeded')
                sleep(min(1, remaining))
        raise ValueError('Pages branch reconciliation exhausted')

    confirm_head()
    observe({'phase': 'updated', 'destination_changed': staged != destination['head']})
    observe({'phase': 'build-requested'})
    try:
        requested = request('POST', endpoint + '/pages/builds')
        if not requested: raise ValueError('Pages build request was not accepted')
    except (OSError, ValueError) as error:
        observe({'phase': 'build-requested', 'event': 'build-response-unavailable', 'category': type(error).__name__})
    while True:
        if clock() - started >= timing.ARTIFACT_PRODUCER_SECONDS: raise ValueError('Pages build deadline exceeded')
        build = request('GET', endpoint + '/pages/builds/latest')
        if not build: raise ValueError('Pages build result is unavailable')
        if build.get('status') == 'built':
            if build.get('commit') == staged:
                confirm_head()
                observe({'phase': 'deployed', 'pages_build': 'built'})
                return {'promoted': True, 'version': version, 'revision': revision, 'pages_commit': staged, 'domain': DOMAIN, 'pages_build': 'built'}
            if build.get('commit') != destination['head']:
                raise ValueError('Pages completed build differs from staged commit')
            # The latest endpoint can briefly show the previous completed build
            # while the explicitly requested new build enters its queue.
        else:
            if build.get('status') not in ('queued', 'building'): raise ValueError('Pages build failed')
            if build.get('commit') not in (None, staged): raise ValueError('Pages pending build differs from staged commit')
        remaining = timing.ARTIFACT_PRODUCER_SECONDS - (clock() - started)
        if remaining <= 0: raise ValueError('Pages build deadline exceeded')
        sleep(min(5, remaining))
