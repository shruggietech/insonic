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
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent))
import qualification_timing as timing


def sibling(name):
    spec = importlib.util.spec_from_file_location('insonic_' + name, Path(__file__).with_name(name + '.py'))
    value = importlib.util.module_from_spec(spec); spec.loader.exec_module(value)
    return value


http = sibling('http_read')
pages = sibling('pages')


CSS_TOKEN = re.compile(r'''/\*.*?\*/|\burl\(\s*(?P<url>"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|(?:\\.|[^()'"\\])*)\s*\)|@import\s+(?P<import>"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*')|"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*' ''', re.I | re.S | re.X)


def css_links(text):
    def unescape(match):
        if match[1]:
            value = int(match[1], 16)
            return chr(value) if 0 < value <= 0x10ffff else '\ufffd'
        return '' if match[2] else match[3]
    links = []
    for token in CSS_TOKEN.finditer(text):
        value = token['url'] if token['url'] is not None else token['import']
        if value is None: continue
        value = value.strip()
        if value[:1] in ('"', "'"): value = value[1:-1]
        value = re.sub(r'\\(?:([0-9a-fA-F]{1,6})[ \t\r\n\f]?|(\r\n|[\r\n\f])|(.))', unescape, value)
        links.append(value)
    return links


class Links(HTMLParser):
    def __init__(self, text):
        super().__init__(convert_charrefs=True)
        self.links, self.canonical = [], []
        self.in_style = False
        self.feed(text)

    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        if tag == 'style': self.in_style = True
        self.links.extend((value, True) for value in css_links(attrs.get('style', '')))
        if tag == 'link' and attrs.get('rel') == 'canonical': self.canonical.append(attrs.get('href'))
        for key in ('href', 'src', 'poster'):
            if key in attrs:
                if tag == 'link' and attrs.get('rel') == 'canonical': continue
                resource = key != 'href' or (tag == 'link' and attrs.get('rel') != 'canonical')
                self.links.append((attrs[key], resource))

    def handle_endtag(self, tag):
        if tag == 'style': self.in_style = False

    def handle_data(self, data):
        if self.in_style: self.links.extend((value, True) for value in css_links(data))


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
    # Include the complete bounded original export, not just HTML entry points.
    # This also covers fonts and indirectly loaded JS/CSS resources.
    selected = set(files)
    base_origin = http.origin(site)
    requests = {}
    for name in sorted(selected):
        route = name.removesuffix('index.html') if name.endswith('/index.html') and '/latest/' not in name else name
        if name == 'index.html': route = ''
        requests[site + quote(route, safe='/')] = name
    pending = list(requests.items())
    for page_url, name in pending:
        if name.endswith('.html'):
            links = Links(files[name].decode('utf-8')).links
        elif name.endswith('.css'):
            links = [(value, True) for value in css_links(files[name].decode('utf-8'))]
        else: continue
        for href, resource in links:
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
            if resolved not in requests:
                requests[resolved] = matched
                pending.append((resolved, matched))
                if len(requests) > http.MAX_PATHS: raise ValueError('public verification path inventory exceeds limit')
    if len(requests) > http.MAX_PATHS: raise ValueError('public verification path inventory exceeds limit')
    return sorted(requests.items())


def verify(archive, version, revision, site, *, package_urls, api=None,
           pages_commit=None, reader=None, clock=time.monotonic):
    started = clock()
    http.origin(site)
    if not site.endswith('/'): raise ValueError('public site base must end with a slash')
    if not re.fullmatch(r'[0-9a-f]{40}', revision): raise ValueError('public source revision is invalid')
    files = pages.archive_files(archive)
    requests = inventory(files, version, revision, site, package_urls)
    if pages_commit is not None: deployment(api, pages_commit)
    deadline = clock() + timing.SUBPROCESS_SECONDS
    checked = []
    if reader is None:
        results = http.read_many([{'url': url} for url, _ in requests], deadline=deadline)
        if len(results) != len(requests) or [item['url'] for item in results] != [url for url, _ in requests]:
            raise ValueError('public worker response inventory differs')
    else:
        results = None
    for index, (url, name) in enumerate(requests):
        if results is None:
            actual = reader(url, max_bytes=16 << 20, deadline=deadline)
            digest = hashlib.sha256(actual).hexdigest()
        else:
            if 'error' in results[index]:
                raise OSError('public request failed for original archive path: ' + name)
            digest = results[index]['sha256']
        if clock() > deadline: raise TimeoutError('public verification deadline exceeded')
        if digest != hashlib.sha256(files[name]).hexdigest():
            raise ValueError('live public bytes differ from original archive: ' + name)
        checked.append({'path': name, 'url': url, 'sha256': digest})
    if pages_commit is not None: deployment(api, pages_commit)
    return {'version': version, 'revision': revision, 'site': site,
            'pages_commit': pages_commit, 'content_proof': 'downloaded-original-byte-match',
            'paths': checked, 'checked_files': len({item['path'] for item in checked}),
            'checked_urls': len(checked),
            'archive_files': len(files), 'elapsed_seconds': round(clock() - started, 3)}
