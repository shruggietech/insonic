#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Bounded query regressions against the exact installed CLI and companions."""
import json
import time
import uuid
from cli_media_qualification import pcm_fixture


def qualify(invoke, supplied, workspace):
    started = time.monotonic()
    directory = workspace / 'query contract fixtures'
    directory.mkdir()
    audio, subtitle = directory / 'source.wav', directory / 'source.srt'
    pcm_fixture(audio, 2)
    subtitle.write_text(''.join(f'{i + 1}\n00:00:00,{i:03d} --> 00:00:00,{i + 1:03d}\nquery-continuation-fixture\n\n'
                                for i in range(607)), encoding='utf-8', newline='\n')
    queued = invoke(['media', 'import', audio, '--new-entry', '--no-pipeline',
                     '--originated-at', '2026-03-08T05:30:00Z', '--timezone', 'UTC'])['result']
    media_id = invoke(['work', 'wait', queued['work_id']])['result']['result']['items'][0]['media_id']
    queued = invoke(['transcript', 'import', subtitle, '--record', media_id])['result']
    invoke(['work', 'wait', queued['work_id']])
    recording = invoke(['recordings', 'show', media_id])['result']['recording']
    definition = {'definition': {'mode': 'normalized', 'operation': 'graph-view',
                    'filters': {'entity_ids': ['document:' + media_id + ':' + recording['document_digest']]},
                    'traversal': {'direction': 'out', 'max_depth': 1, 'relationship_types': ['contains-cue']},
                    'pagination': {'limit': 100}}}
    rows, edges = set(), set()
    for page in range(40):
        result = supplied(['query', 'run'], definition)['result']
        for field, seen in [('items', rows), ('edges', edges)]:
            ids = [item['id'] for item in result[field]]
            if len(set(ids)) != len(ids) or seen.intersection(ids):
                raise ValueError('Query continuation repeated identities')
            seen.update(ids)
        if not result.get('next_cursor'):
            break
        definition['definition']['pagination']['cursor'] = result['next_cursor']
    else:
        raise ValueError('Query continuation did not finish')
    if len(rows) != 608 or len(edges) != 607:
        raise ValueError('Query pages omitted document/cue relationships')
    seen, cursor = set(), ''
    for page in range(10):
        args = ['search', 'query-continuation-fixture', '--media', media_id, '--limit', '100']
        result = invoke([*args, *(['--after', cursor] if cursor else [])])['result']
        ids = {item['id'] for item in result['items']}
        if seen.intersection(ids):
            raise ValueError('Search continuation repeated rows')
        seen.update(ids)
        if not result.get('next_cursor'):
            break
        cursor = result['next_cursor']
    else:
        raise ValueError('Search continuation did not finish')
    if len(seen) != 607:
        raise ValueError('Search continuation omitted matches')
    human = invoke(['search', 'query-continuation-fixture', '--media', media_id], machine=False)
    if 'Next cursor:' not in human or '--after' not in human:
        raise ValueError('Human search lost continuation before its display bound')
    for zone in ['local', 'America/New_York', '+02:00']:
        query = {'definition': {'mode': 'normalized', 'operation': 'media-list',
                     'filters': {'media_ids': [media_id], 'recording_dates': {'timezone': zone}}}}
        supplied(['query', 'validate'], query)
        if supplied(['query', 'run'], query)['result']['total'] != 1:
            raise ValueError('Advertised timezone rejected by installed query')
        supplied(['timeline', 'calendar'], {'filter': {'timezone': zone, 'include_undated': True}})
    help_text = invoke(['query', 'save', '--help'])['result']['help']
    marker = 'JSON file template (replace illustrative IDs/revisions with current records):\n'
    template, _ = json.JSONDecoder().raw_decode(help_text.split(marker, 1)[1])
    if not template['query'].get('title', '').strip():
        raise ValueError('Query-save help omitted its required title')
    supplied(['query', 'save', str(uuid.uuid4())], template)
    query_file, parameters = directory / 'read.cypher', directory / 'parameters.json'
    query_file.write_text('RETURN $value AS exact; /* one read */\n', encoding='utf-8', newline='\n')
    parameters.write_text('{"value":{"type":"integer","value":9223372036854775807}}', encoding='utf-8')
    native = ['query', 'native', '--dialect', 'ladybug-cypher', '--file', query_file, '--parameters', parameters]
    result = invoke(native)['result']
    if '9223372036854775807' not in json.dumps(result):
        raise ValueError('Native parameter rounded or terminating read failed')
    parameters.write_text('{"value":{"type":"integer","value":1},"value":{"type":"integer","value":2}}', encoding='utf-8')
    if invoke(native, expected=2)['error']['code'] != 'invalid_request':
        raise ValueError('Duplicate native parameter reached execution')
    return {'state': 'passed', 'issues': [122, 123, 124, 125, 128, 129], 'rows': len(rows),
            'edges': len(edges), 'search_matches': len(seen), 'timezone_forms': 3,
            'elapsed_seconds': round(time.monotonic() - started, 3)}
