#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Bounded unauthenticated HTTPS GETs for explicitly selected public origins."""
import http.client
import base64
from concurrent.futures import ThreadPoolExecutor
import hashlib
import json
import os
from pathlib import Path
import ssl
import socket
import subprocess
import sys
import threading
import time
from urllib.parse import urljoin, urlsplit

sys.path.insert(0, str(Path(__file__).resolve().parent))
import qualification_timing as timing

TRANSIENT = {429, 500, 502, 503, 504}
MAX_PATHS = 10000


def origin(url):
    try:
        parts = urlsplit(url)
        port = parts.port
    except ValueError:
        raise ValueError('public HTTPS destination is invalid') from None
    if (parts.scheme != 'https' or not parts.hostname or parts.username is not None
            or parts.password is not None or parts.query or parts.fragment
            or any(ord(c) <= 32 or ord(c) >= 127 for c in url) or '\\' in url
            or '%' in parts.netloc):
        raise ValueError('public HTTPS destination must have no credentials, query or fragment')
    return parts.hostname.lower(), port or 443


def request(url, limit, deadline, clock):
    """One GET, no ambient cookies/proxy credentials, redirects or decompression."""
    hostname, port = origin(url)
    remaining = deadline - clock()
    if remaining <= 0: raise TimeoutError('public request deadline exceeded')
    connection = http.client.HTTPSConnection(hostname, port, timeout=min(timing.REQUEST_SECONDS, remaining), context=ssl.create_default_context())
    response = None
    def expire():
        # A peer can trickle headers often enough to avoid a socket's inactivity
        # timeout. Interrupt only this request's owned socket at its deadline.
        try:
            selected = connection.sock
            if selected is None and response is not None and response.fp is not None:
                selected = response.fp.raw._sock
            if selected is not None: selected.shutdown(socket.SHUT_RDWR)
        except (OSError, AttributeError):
            pass
        finally:
            connection.close()
    watchdog = threading.Timer(max(0, deadline - clock()), expire)
    watchdog.daemon = True
    watchdog.start()
    try:
        parts = urlsplit(url)
        connection.request('GET', parts.path or '/', headers={
            'User-Agent': 'insonic-public-verification', 'Accept-Encoding': 'identity',
            'Cache-Control': 'no-cache',
        })
        response = connection.getresponse()
        headers = {key.lower(): value for key, value in response.getheaders()}
        if response.status != 200: return response.status, headers, b''
        if headers.get('content-encoding', 'identity') != 'identity':
            raise ValueError('public response uses unexpected content encoding')
        if 'content-length' in headers:
            try: length = int(headers['content-length'])
            except ValueError: raise ValueError('public response size is invalid') from None
            if length < 0 or length > limit: raise ValueError('public response size exceeds limit')
        data = bytearray()
        while True:
            remaining = deadline - clock()
            if remaining <= 0: raise TimeoutError('public request deadline exceeded')
            if response.isclosed(): break
            # HTTPResponse owns the socket after a Connection: close response.
            # Set each read's timeout against the aggregate deadline as well.
            response.fp.raw._sock.settimeout(min(timing.REQUEST_SECONDS, remaining))
            chunk = response.read1(min(65536, limit + 1 - len(data)))
            if not chunk: break
            data.extend(chunk)
            if len(data) > limit: raise ValueError('public response size exceeds limit')
        if 'content-length' in headers and len(data) != length:
            raise OSError('public response was truncated')
        return response.status, headers, bytes(data)
    finally:
        watchdog.cancel()
        if response is not None: response.close()
        connection.close()


def read(url, *, max_bytes, deadline, retry_404=False, transport=None,
         sleep=time.sleep, clock=time.monotonic, on_attempt=None):
    if transport is None:
        origin(url)
        result = json.loads(worker({'mode': 'fetch', 'url': url, 'max_bytes': max_bytes,
                                    'deadline': deadline, 'retry_404': retry_404}, deadline))
        data = base64.b64decode(result['data'], validate=True)
        if len(data) > max_bytes: raise ValueError('public response size exceeds limit')
        if on_attempt is not None: on_attempt(result['attempts'])
        return data
    elected = origin(url)
    retryable = TRANSIENT | ({404} if retry_404 else set())
    for attempt in range(3):
        if on_attempt is not None: on_attempt(attempt + 1)
        current = url
        retry = False
        for redirects in range(6):
            if deadline - clock() <= 0: raise TimeoutError('public request deadline exceeded')
            try:
                status, headers, data = transport(current, max_bytes, deadline, clock)
            except ssl.SSLCertVerificationError:
                raise ValueError('public TLS certificate verification failed') from None
            except (OSError, http.client.HTTPException):
                retry = True
                break
            if status in (301, 302, 303, 307, 308):
                if redirects == 5: raise ValueError('public redirect limit exceeded')
                current = urljoin(current, headers.get('location', ''))
                if not headers.get('location') or origin(current) != elected:
                    raise ValueError('public redirect left the selected HTTPS origin')
                continue
            if status == 200:
                if len(data) > max_bytes: raise ValueError('public response size exceeds limit')
                if clock() > deadline: raise TimeoutError('public request deadline exceeded')
                return data
            if status not in retryable:
                raise OSError(f'public request failed with HTTP {status}')
            retry = True
            break
        if retry and attempt < 2:
            remaining = deadline - clock()
            if remaining <= 0: raise TimeoutError('public request deadline exceeded')
            sleep(min(1 << attempt, remaining))
            continue
        raise OSError('public request failed after three attempts')


def worker(payload, deadline, *, argv=None):
    """Bound DNS, connect, TLS and body work by this owned child's lifetime.

    subprocess.run kills and reaps its own child on timeout. The child launches
    no subprocesses. Batch requests reuse one process and four bounded threads.
    """
    remaining = deadline - time.monotonic()
    if remaining <= 0: raise TimeoutError('public worker deadline exceeded')
    allowed = {'PATH', 'SYSTEMROOT', 'WINDIR', 'TEMP', 'TMP', 'TMPDIR', 'LANG', 'LC_ALL'}
    environment = {key: value for key, value in os.environ.items() if key.upper() in allowed}
    environment['PYTHONUTF8'] = '1'
    try:
        result = subprocess.run(argv or [sys.executable, str(Path(__file__).resolve()), '--worker'],
                                input=json.dumps(payload).encode('utf-8'), stdout=subprocess.PIPE,
                                stderr=subprocess.PIPE, timeout=remaining, env=environment,
                                creationflags=subprocess.CREATE_NO_WINDOW if os.name == 'nt' else 0)
    except subprocess.TimeoutExpired:
        raise TimeoutError('public worker deadline exceeded') from None
    if result.returncode:
        message = result.stderr.decode('utf-8', errors='replace').strip()
        raise OSError(message[:1000] or 'public network worker failed')
    if time.monotonic() > deadline: raise TimeoutError('public worker deadline exceeded')
    return result.stdout


def read_many(jobs, *, deadline):
    """Return downloaded hashes for one bounded batch, without product binaries."""
    if len(jobs) > MAX_PATHS: raise ValueError('public batch path limit exceeded')
    origins = {origin(job['url']) for job in jobs}
    if len(origins) > 1: raise ValueError('public batch left the selected HTTPS origin')
    return json.loads(worker({'mode': 'batch', 'jobs': jobs, 'deadline': deadline}, deadline))


def worker_main():
    payload = json.loads(sys.stdin.buffer.read(16 << 20))
    deadline = payload['deadline']
    if payload['mode'] == 'fetch':
        attempts = [0]
        data = read(payload['url'], max_bytes=payload['max_bytes'], deadline=deadline,
                    retry_404=payload.get('retry_404', False), transport=request,
                    on_attempt=lambda value: attempts.__setitem__(0, value))
        value = {'data': base64.b64encode(data).decode('ascii'), 'attempts': attempts[0]}
    elif payload['mode'] == 'batch':
        jobs = payload['jobs']
        if len(jobs) > MAX_PATHS: raise ValueError('public batch path limit exceeded')
        def fetch(job):
            try:
                data = read(job['url'], max_bytes=16 << 20, deadline=deadline, transport=request)
            except (OSError, ValueError, http.client.HTTPException):
                return {'url': job['url'], 'error': 'public request failed'}
            return {'url': job['url'], 'sha256': hashlib.sha256(data).hexdigest(), 'size_bytes': len(data)}
        with ThreadPoolExecutor(max_workers=4) as pool:
            value = list(pool.map(fetch, jobs))
    else: raise ValueError('public worker mode is invalid')
    sys.stdout.write(json.dumps(value))


if __name__ == '__main__':
    try:
        if sys.argv[1:] != ['--worker']: raise ValueError('public network worker requires its internal mode')
        worker_main()
    except (OSError, ValueError, KeyError, http.client.HTTPException):
        # Never echo URLs, response bodies, environment or exception payloads.
        print('public network worker failed or exceeded its request budget', file=sys.stderr)
        sys.exit(1)
