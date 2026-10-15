#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Bounded unauthenticated HTTPS GETs for explicitly selected public origins."""
import http.client
import ssl
import socket
import threading
import time
from urllib.parse import urljoin, urlsplit

TRANSIENT = {429, 500, 502, 503, 504}


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
    connection = http.client.HTTPSConnection(hostname, port, timeout=min(15, remaining), context=ssl.create_default_context())
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
            response.fp.raw._sock.settimeout(min(15, remaining))
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


def read(url, *, max_bytes, deadline, retry_404=False, transport=request,
         sleep=time.sleep, clock=time.monotonic, on_attempt=None):
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
