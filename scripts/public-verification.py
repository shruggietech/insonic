#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Read-only verification of original publication-target documentation over HTTPS."""
import hashlib
from html.parser import HTMLParser
import importlib.util
import json
from pathlib import Path
import re
import time
from urllib.parse import quote, unquote, urljoin, urlsplit


def sibling(name):
    spec = importlib.util.spec_from_file_location('insonic_' + name, Path(__file__).with_name(name + '.py'))
    value = importlib.util.module_from_spec(spec); spec.loader.exec_module(value)
    return value


http = sibling('http_read')
pages = sibling('pages')


class Links(HTMLParser):
    def __init__(self, text):
        super().__init__(convert_charrefs=True)
        self.links, self.canonical = [], []
        self.feed(text)

    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        if tag == 'link' and attrs.get('rel') == 'canonical': self.canonical.append(attrs.get('href'))
        for key in ('href', 'src', 'poster'):
            if key in attrs:
                if tag == 'link' and attrs.get('rel') == 'canonical': continue
                resource = key != 'href' or (tag == 'link' and attrs.get('rel') != 'canonical')
                self.links.append((attrs[key], resource))


def deployment(api, commit):
    pages.commit_sha(commit)
    if api is None: raise ValueError('Pages identity verification requires a read-only API')
    endpoint = 'repos/shruggietech/insonic'
    ref = api.call('GET', endpoint + '/git/ref/heads/gh-pages')
    build = api.call('GET', endpoint + '/pages/builds/latest')
    if (not ref or ref.get('object', {}).get('type') != 'commit'
            or ref['object'].get('sha') != commit or not build
            or build.get('status') != 'built' or build.get('commit') != commit):
        raise ValueError('Pages branch or completed build differs from selected commit')


def inventory(files, version, revision, site, package_urls):
    try:
        identity = json.loads(files['release-documentation.json'])
        marker = json.loads(files['documentation-build.json'])
        manifest = json.loads(files['documentation-manifest.json'])
        latest = [v for v in manifest['versions'] if v['version'] == version]
    except (ValueError, KeyError, TypeError, UnicodeError):
        raise ValueError('public documentation identity is unavailable') from None
    if (identity != {'kind': 'documentation', 'version': version, 'revision': revision}
            or marker != {'version': version, 'revision': revision, 'source_dirty': False, 'documentation_target': 'release'}
            or manifest.get('latest') != version or len(latest) != 1 or latest[0].get('status') != 'released'):
        raise ValueError('public documentation requires exact released latest identity')
    required = {'index.html', 'docs/index.html', 'docs/latest/index.html',
                f'docs/v{version}/index.html', f'schemas/v{version}/master.schema.json'}
    if not required <= files.keys(): raise ValueError('public documentation entry or schema is missing')
    landing = Links(files['index.html'].decode('utf-8'))
    hrefs = {link for link, _ in landing.links}
    if 'docs/latest/index.html' not in hrefs:
        raise ValueError('landing page must use stable latest documentation entry')
    if not set(package_urls) <= hrefs:
        raise ValueError('landing page package links differ from published inventory')
    canonical = Links(files['docs/latest/index.html'].decode('utf-8')).canonical
    if canonical != [f'https://insonic.io/docs/v{version}/index/']:
        raise ValueError('latest documentation canonical differs from release')
    selected = {name for name in files if name.endswith('.html') or name.startswith('schemas/')}
    selected.update(('release-documentation.json', 'documentation-build.json', 'documentation-manifest.json'))
    base_origin = http.origin(site)
    requests = {}
    for name in sorted(selected):
        route = name.removesuffix('index.html') if name.endswith('/index.html') and '/latest/' not in name else name
        if name == 'index.html': route = ''
        requests[site + quote(route, safe='/')] = name
    for name in sorted(selected.copy()):
        if not name.endswith('.html'): continue
        page_url = site + name.removesuffix('index.html') if name.endswith('/index.html') else site + name
        for href, resource in Links(files[name].decode('utf-8')).links:
            if not href or href.startswith('#'): continue
            resolved = urljoin(page_url, href)
            parts = urlsplit(resolved)
            local = (parts.scheme == 'https' and parts.hostname
                     and (parts.hostname, parts.port or 443) == base_origin)
            if not local:
                if resource and not href.startswith('data:'):
                    raise ValueError('public rendering resource leaves selected archive origin')
                continue
            # Fragment identifiers do not change the fetched file identity.
            relative = unquote(parts.path).lstrip('/')
            prefix = urlsplit(site).path.strip('/')
            if prefix and relative.startswith(prefix + '/'): relative = relative[len(prefix) + 1:]
            candidates = [relative, relative + 'index.html' if relative.endswith('/') or not relative else relative + '/index.html']
            matched = next((path for path in candidates if path in files), None)
            if matched is None: raise ValueError('public navigation or resource is absent from original archive: ' + relative)
            if parts.query: raise ValueError('public static archive reference has an unsupported query')
            resolved = parts._replace(path=quote(unquote(parts.path), safe='/'), fragment='').geturl()
            if http.origin(resolved) != base_origin:
                raise ValueError('public reference left the elected HTTPS origin')
            requests[resolved] = matched
    if len(requests) > 1000: raise ValueError('public verification path inventory exceeds limit')
    return sorted(requests.items())


def verify(archive, version, revision, site, *, package_urls, api=None,
           pages_commit=None, reader=http.read, clock=time.monotonic):
    started = clock()
    http.origin(site)
    if not site.endswith('/'): raise ValueError('public site base must end with a slash')
    if not re.fullmatch(r'[0-9a-f]{40}', revision): raise ValueError('public source revision is invalid')
    files = pages.archive_files(archive)
    requests = inventory(files, version, revision, site, package_urls)
    if pages_commit is not None: deployment(api, pages_commit)
    deadline = clock() + 120
    checked = []
    for url, name in requests:
        actual = reader(url, max_bytes=16 << 20, deadline=deadline)
        if clock() > deadline: raise TimeoutError('public verification deadline exceeded')
        digest = hashlib.sha256(actual).hexdigest()
        if digest != hashlib.sha256(files[name]).hexdigest():
            raise ValueError('live public bytes differ from original archive: ' + name)
        checked.append({'path': name, 'url': url, 'sha256': digest})
    if pages_commit is not None: deployment(api, pages_commit)
    return {'version': version, 'revision': revision, 'site': site,
            'pages_commit': pages_commit, 'content_proof': 'downloaded-original-byte-match',
            'paths': checked, 'checked_files': len({item['path'] for item in checked}),
            'checked_urls': len(checked),
            'archive_files': len(files), 'elapsed_seconds': round(clock() - started, 3)}
