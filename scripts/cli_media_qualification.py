#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Deterministic media lifecycle cases inside installed CLI qualification.

No inference, owner media, network downloads or ambient media executables are used.
"""
import base64
import hashlib
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
import struct
import threading
import time
import wave

from process_tree import ProcessTree


def pcm_fixture(destination, channels):
    """One second of deterministic integer PCM, with distinct channel samples."""
    with wave.open(str(destination), 'wb') as output:
        output.setparams((channels, 2, 16000, 16000, 'NONE', 'not compressed'))
        output.writeframes(b''.join(struct.pack('<h', ((frame * 97 + channel * 31) % 2000) - 1000)
                                   for frame in range(16000) for channel in range(channels)))


def qualify(invoke, supplied, workspace, executable, environment, version):
    started = time.monotonic()
    directory = workspace / 'media lifecycle résumé'
    directory.mkdir()
    companions = Path(executable).parent / 'companions' / 'media'
    suffix = '.exe' if os.name == 'nt' else ''
    ffmpeg, ffprobe = (companions / (name + suffix) for name in ['ffmpeg', 'ffprobe'])
    if not all(tool.is_file() for tool in [ffmpeg, ffprobe]):
        raise ValueError('Media lifecycle requires exact installed companions')

    def tool(exe, *args):
        with ProcessTree([exe, *args], cwd=directory, env=environment) as tree:
            stdout, stderr = tree.process.communicate(timeout=20)
            if tree.process.returncode or len(stdout) > 1 << 20 or len(stderr) > 65536:
                raise ValueError('Installed media lifecycle companion failed')
            return stdout

    def wait(queued):
        return invoke(['work', 'wait', queued['work_id']])['result']['result']['items']

    stereo, mono = directory / 'stereo.wav', directory / 'mono.wav'
    pcm_fixture(stereo, 2)
    pcm_fixture(mono, 1)
    compressed_mono = directory / 'mono.m4a'
    tool(ffmpeg, '-nostdin', '-v', 'error', '-y', '-i', mono, '-c:a', 'aac', compressed_mono)
    video = Path(__file__).resolve().parents[1] / 'tests/fixtures/media/sintel-dialogue.mkv'
    records = []
    for source, canonical_format, expected in [(stereo, 'flac', 'audio/flac'),
                                               (compressed_mono, 'mp3', 'audio/wav'), (video, None, None)]:
        item = wait(invoke(['media', 'import', source, '--new-entry', '--no-pipeline'])['result'])[0]
        entry = invoke(['media', 'show', item['media_id']])['result']
        if canonical_format and entry['facts']['canonical']['format'] != canonical_format:
            raise ValueError('Canonical codec differs for the generated audio fixture')
        original_metadata = entry['metadata']
        descriptor = supplied(['media', 'playback', entry['id']], {'revision': entry['revision']})['result']
        try:
            actual = Path(descriptor['path']).read_bytes()
            probe = json.loads(tool(ffprobe, '-v', 'error', '-show_format', '-of', 'json', descriptor['path']))
            observed = {'flac': 'audio/flac', 'mp3': 'audio/mpeg', 'wav': 'audio/wav', 'matroska,webm': 'audio/x-matroska'}[probe['format']['format_name']]
            if descriptor['mime_type'] != observed or expected and descriptor['mime_type'] != expected:
                raise ValueError('Playback MIME differs for ' + source.name + ': returned '
                                 + descriptor['mime_type'] + ', probed ' + observed + ', expected ' + str(expected))
            if descriptor['size_bytes'] != len(actual) or descriptor['digest'] != hashlib.sha256(actual).hexdigest():
                raise ValueError('Playback identity differs from returned bytes')
            if invoke(['media', 'show', entry['id']])['result']['metadata'] != original_metadata:
                raise ValueError('Playback rewrote original source observations')
        finally:
            supplied(['media', 'playback-close', entry['id']], {'playback_id': descriptor['playback_id']})
        records.append(entry['id'])

    # Stream-copy fixture construction works before the broken SubRip encoder
    # selection, so this exercises actual admission rather than an inventory.
    texts = {
        'srt': '1\n00:00:00,000 --> 00:00:00,500\nSelected résumé track\n\n',
        'vtt': 'WEBVTT\n\n00:00:00.000 --> 00:00:00.500\nSelected résumé track\n\n',
        'ass': '[Script Info]\nScriptType: v4.00+\n[V4+ Styles]\n'
               'Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\n'
               'Style: Default,Arial,20,&H00FFFFFF,&H000000FF,&H00000000,&H00000000,0,0,0,0,100,100,0,0,1,2,0,2,10,10,10,1\n'
               '[Events]\nFormat: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n'
               'Dialogue: 0,0:00:00.00,0:00:00.50,Default,,0,0,0,,Selected résumé track\n',
    }
    for extension, text in texts.items():
        track = directory / ('selected.' + extension)
        track.write_text(text, encoding='utf-8', newline='\n')
        embedded = directory / ('embedded-' + extension + '.mkv')
        tool(ffmpeg, '-nostdin', '-v', 'error', '-y', '-i', stereo, '-i', track,
             '-map', '0:a:0', '-map', '1:s:0', '-c', 'copy', '-metadata:s:s:0', 'language=eng', embedded)
        item = wait(invoke(['media', 'import', embedded, '--subtitle-language', 'eng', '--new-entry', '--no-pipeline'])['result'])[0]
        document = invoke(['recordings', 'document', item['media_id']])['result']
        decoded = json.loads(base64.b64decode(document['data']))
        cues = decoded['cues']
        if (len(cues) != 1 or cues[0]['payload']['plain_text'] != 'Selected résumé track'
                or cues[0]['timing']['start_milliseconds'] != 0
                or cues[0]['timing']['end_milliseconds'] != 500):
            raise ValueError('Installed embedded admission lost Unicode text or timing')

    manifest = directory / 'transcript batch.json'
    manifest.write_text(json.dumps({'kind': 'import-manifest', 'schema_version': version,
        'items': [{'record': identity, 'transcript': str(directory / 'selected.srt')}
                  for identity in records[:2]]}) + '\n', encoding='utf-8', newline='\n')
    items = wait(invoke(['transcript', 'import', '--manifest', manifest])['result'])
    if len(items) != 2 or any(item['state'] != 'admitted' for item in items):
        raise ValueError('Manifest-only transcript batch failed')

    # A normal raw reader must not turn a committed refresh into failed work.
    identity = records[0]
    leases = invoke(['media', 'raw', identity])['result']
    before = invoke(['media', 'show', identity])['result']
    try:
        queued = invoke(['media', 'refresh', identity, '--no-pipeline'])['result']
        completed = invoke(['work', 'wait', queued['work_id']])['result']
        accepted = invoke(['media', 'show', identity])['result']
        if completed['state'] != 'succeeded' or accepted['revision'] <= before['revision']:
            raise ValueError('Leased report caused a false refresh failure')
        for lease in leases:
            publication = lease['publication_id']
            if invoke(['artifacts', 'show', publication])['result']['state'] != 'available':
                raise ValueError('Refresh retired a leased report')
            blocked = invoke(['artifacts', 'retire', publication], expected=1)
            if blocked['error']['code'] != 'conflict' or 'artifact retirement is deferred' in json.dumps(blocked):
                raise ValueError('Internal deferred-retirement marker escaped its public conflict')
    finally:
        for lease in leases:
            invoke(['artifacts', 'lease-release', lease['publication_id'], lease['lease']['id']])
    queued = invoke(['media', 'refresh', identity, '--no-pipeline'])['result']
    invoke(['work', 'wait', queued['work_id']])
    for lease in leases:
        if invoke(['artifacts', 'show', lease['publication_id']])['result']['state'] != 'retired':
            raise ValueError('Deferred report cleanup did not finish after lease release')

    payload = texts['srt'].encode('utf-8')
    class Handler(BaseHTTPRequestHandler):
        def do_GET(self):
            if self.path == '/slow.srt':
                time.sleep(0.25)
            try:
                self.send_response(200)
                if self.path != '/streamed.srt':
                    self.send_header('Content-Length', str(len(payload)))
                self.end_headers()
                self.wfile.write(payload)
            except ConnectionError:
                pass

        def log_message(self, *_args):
            pass

    server = ThreadingHTTPServer(('127.0.0.1', 0), Handler)
    thread = threading.Thread(target=server.serve_forever, kwargs={'poll_interval': 0.05}, daemon=True)
    thread.start()
    failures = []
    current = invoke(['recordings', 'show', identity])['result']['recording']
    try:
        for route, options, cause, flag in [
            ('slow', ['--transcript-timeout-ms', '50'], 'acquisition_timeout', '--transcript-timeout-ms'),
            ('declared', ['--transcript-max-bytes', '8'], 'input_limit', '--transcript-max-bytes'),
            ('streamed', ['--transcript-max-bytes', '8'], 'input_limit', '--transcript-max-bytes'),
        ]:
            queued = invoke(['transcript', 'import', f'http://127.0.0.1:{server.server_port}/{route}.srt',
                             '--record', identity, '--replace-transcript', *options])['result']
            failed = invoke(['work', 'wait', queued['work_id']], expected=1)
            if failed['error']['code'] != cause or flag not in json.dumps(failed):
                raise ValueError('Acquisition wait lost its durable cause/correction')
            for action in ['show', 'results']:
                result = invoke(['work', action, queued['work_id']])
                if result['result']['error'] != cause or flag not in json.dumps(result):
                    raise ValueError('Acquisition readback lost its durable cause/correction')
            human = invoke(['work', 'wait', queued['work_id']], expected=1, machine=False)
            if flag not in human or 'child output' in human:
                raise ValueError('Human acquisition correction is not executable')
            unchanged = invoke(['recordings', 'show', identity])['result']['recording']
            if (unchanged['revision'], unchanged['document_digest']) != (current['revision'], current['document_digest']):
                raise ValueError('Failed acquisition replaced the accepted transcript')
            failures.append({'work_id': queued['work_id'], 'code': cause})
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=5)
        if thread.is_alive():
            raise ValueError('Owned transcript fixture server did not stop')
    return {'issues': [140, 141, 142, 143, 145], 'state': 'passed', 'playback_cases': 3,
            'embedded_formats': list(texts), 'transcript_manifest_items': 2, 'failures': failures,
            'elapsed_seconds': round(time.monotonic() - started, 3)}
