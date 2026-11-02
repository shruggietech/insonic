#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Current evidence, cache and citation checks against the installed CLI."""
import copy
import json
import time

from cli_media_qualification import pcm_fixture


def qualify(invoke, supplied, workspace, version):
    started = time.monotonic()
    directory = workspace / 'answer evidence fixtures'
    directory.mkdir()
    subtitle = directory / 'source.srt'
    subtitle.write_text('1\n00:00:00,000 --> 00:00:01,000\nSolar research evidence.\n\n',
                        encoding='utf-8', newline='\n')
    recordings = []
    for channels in [2, 3]:
        audio = directory / f'source-{channels}.wav'
        pcm_fixture(audio, channels)
        bounds = ['--originated-earliest', '2026-01-01T00:00:00Z', '--originated-latest', '2026-01-02T00:00:00Z'] if channels == 2 else ['--originated-earliest', '2026-02-01T00:00:00Z']
        queued = invoke(['media', 'import', audio, '--new-entry', '--no-pipeline', *bounds])['result']
        media_id = invoke(['work', 'wait', queued['work_id']])['result']['result']['items'][0]['media_id']
        queued = invoke(['transcript', 'import', subtitle, '--record', media_id])['result']
        invoke(['work', 'wait', queued['work_id']])
        recordings.append(invoke(['recordings', 'show', media_id])['result']['recording'])
    invoke(['graph', 'publish'])
    request = {'version': version, 'question': 'What solar research evidence exists?',
               'filters': {'media_ids': [r['id'] for r in recordings]},
               'budget': {'rounds': 128, 'context_bytes': 8192, 'context_tokens': 8192}}
    first = supplied(['answers', 'prepare'], request)['result']
    package = first['package']
    if first['freshness'] != 'current' or first['cache_hit'] or len(package['items']) != 2:
        raise ValueError('Installed answer preparation lost current multi-recording evidence')
    if package['independent_families'] != 2 or package['context']['bytes'] > 8192 or package['context']['tokens'] > 8192:
        raise ValueError('Installed answer family or context bound differs')
    for item in package['items']:
        if item['reference']['source'] != {'start_us': '0', 'end_us': '1000000'}:
            raise ValueError('Installed answer lost exact source timing')
    cached = supplied(['answers', 'prepare'], request)['result']
    if not cached['cache_hit'] or cached['package']['id'] != package['id']:
        raise ValueError('Unchanged installed answer request did not reuse cache')
    for include_undated in [False, True]:
        dated_request = copy.deepcopy(request)
        dated_request['filters']['recording_dates'] = {'from': '2026-01-01', 'through': '2026-01-31', 'timezone': 'UTC', 'include_undated': include_undated}
        dated = supplied(['answers', 'prepare'], dated_request)['result']['package']['items']
        if len(dated) != 1 or dated[0]['reference']['recording_id'] != recordings[0]['id']:
            raise ValueError('Installed answer date filter lost approximate or open-bound authority')
    for action in ['show', 'export']:
        shown = invoke(['answers', action, package['id']])['result']
        if shown['freshness'] != 'current' or shown['package'] != package:
            raise ValueError('Installed answer inspection/export changed historical bytes')
    human = invoke(['answers', 'show', package['id']], machine=False)
    if package['id'] not in human or 'freshness: current' not in human:
        raise ValueError('Installed answer human output lost identity or freshness')
    item = package['items'][0]
    envelope = {'version': version, 'package_id': package['id'], 'state': 'partial', 'claims': [
        {'id': 'solar', 'text': item['text'], 'support': 'supported', 'citations': [
            {'evidence_id': item['id'], 'reference': item['reference'], 'quote': item['text']}]}]}
    if not supplied(['answers', 'validate'], envelope)['result']['valid']:
        raise ValueError('Installed current citation validation failed')
    forged = copy.deepcopy(envelope)
    forged['claims'][0]['citations'][0]['quote'] = 'Invented source quotation.'
    if supplied(['answers', 'validate'], forged, expected=1)['error']['code'] != 'conflict':
        raise ValueError('Installed invented quote was accepted')
    for recording in recordings:
        after = invoke(['recordings', 'show', recording['id']])['result']['recording']
        if after != recording:
            raise ValueError('Answer preparation changed accepted recording authority')
    subtitle.write_text('1\n00:00:00,000 --> 00:00:01,000\nRevised solar research evidence.\n\n',
                        encoding='utf-8', newline='\n')
    queued = invoke(['transcript', 'import', subtitle, '--record', recordings[0]['id'], '--replace-transcript'])['result']
    invoke(['work', 'wait', queued['work_id']])
    stale = invoke(['answers', 'show', package['id']])['result']
    if stale['freshness'] != 'stale' or stale['package'] != package:
        raise ValueError('Source replacement lost immutable history or stale indication')
    if supplied(['answers', 'validate'], envelope, expected=1)['error']['code'] != 'conflict':
        raise ValueError('Installed stale citation accepted')
    # The package is the serialized historical observation. Current request work
    # accounting is separately returned and must also respect the elected caps.
    for key in ['rows', 'source_bytes', 'rounds', 'edges']:
        if first['usage'][key] > package['request']['budget'][key]:
            raise ValueError('Installed answer aggregate work bound exceeded')
    return {'state': 'passed', 'recordings': 2, 'commands': ['prepare', 'show', 'export', 'validate'],
            'cache_reused': True, 'stale_rejected': True, 'source_unchanged': True,
            'elapsed_seconds': round(time.monotonic() - started, 3)}
