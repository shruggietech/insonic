# SPDX-License-Identifier: Apache-2.0
"""Frozen synthetic speaker semantics and independent accounting; no engines."""
import hashlib
import json
from pathlib import Path
import re

CORPUS = Path(__file__).resolve().parents[1] / 'tests/fixtures/speakers/manifest.json'
CORPUS_SHA256 = 'c24a3b86629c159960b4a6e5b41823d3ed4eb27814e3164d3af522b2470388b4'
OUTCOMES = ('supported-contextual', 'matched', 'unknown', 'ambiguous', 'conflicting')
IDENTITIES = ('registered', 'unregistered', 'unknown')


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError('Duplicate corpus field')
        result[key] = value
    return result


def shape(value, keys):
    if not isinstance(value, dict) or set(value) != set(keys.split()):
        raise ValueError('Speaker reference shape differs')


def label(value):
    return isinstance(value, str) and re.fullmatch(r'[a-z][a-z0-9-]{0,63}', value) is not None


def text(value, maximum=4096):
    return isinstance(value, str) and 0 < len(value.encode('utf-8')) <= maximum and '\0' not in value


def observation(value):
    shape(value, 'voice outcome speaker identity_status')
    if (not label(value['voice']) or value['outcome'] not in OUTCOMES
            or value['identity_status'] not in IDENTITIES
            or (value['speaker'] is not None and not label(value['speaker']))):
        raise ValueError('Speaker observation is invalid')
    if (value['identity_status'] == 'registered') != (value['speaker'] is not None):
        raise ValueError('Registered speaker identity differs')
    if value['outcome'] in ('unknown', 'ambiguous', 'conflicting') and value['identity_status'] != 'unknown':
        raise ValueError('Unresolved observation carries an accepted identity')
    if value['outcome'] in ('supported-contextual', 'matched') and value['identity_status'] == 'unknown':
        raise ValueError('Accepted observation lacks identity status')


def validate_corpus(corpus):
    shape(corpus, 'version id provenance cases')
    if (type(corpus['version']) is not int or corpus['version'] != 1 or not label(corpus['id'])
            or not text(corpus['provenance']) or not isinstance(corpus['cases'], list)
            or not 1 <= len(corpus['cases']) <= 32):
        raise ValueError('Speaker corpus header differs')
    seen = set()
    for case in corpus['cases']:
        shape(case, 'id identities cues expected observations')
        if not label(case['id']) or case['id'] in seen:
            raise ValueError('Duplicate or invalid speaker case')
        seen.add(case['id'])
        if not isinstance(case['identities'], list) or not 1 <= len(case['identities']) <= 16:
            raise ValueError('Speaker roster limit differs')
        keys = set()
        for identity in case['identities']:
            shape(identity, 'key name')
            if not label(identity['key']) or identity['key'] in keys or not text(identity['name'], 128):
                raise ValueError('Speaker identity differs')
            keys.add(identity['key'])
        if not isinstance(case['cues'], list) or not 1 <= len(case['cues']) <= 32:
            raise ValueError('Speaker cue limit differs')
        voices = set()
        for cue in case['cues']:
            shape(cue, 'voice text')
            if not label(cue['voice']) or not text(cue['text']):
                raise ValueError('Speaker cue differs')
            voices.add(cue['voice'])
        if not isinstance(case['expected'], list) or len(case['expected']) != len(voices):
            raise ValueError('Speaker truth coverage differs')
        expected = set()
        for row in case['expected']:
            observation(row)
            if row['voice'] in expected or row['speaker'] is not None and row['speaker'] not in keys:
                raise ValueError('Duplicate or unregistered truth identity')
            expected.add(row['voice'])
        if expected != voices:
            raise ValueError('Speaker truth voices differ')
        if not isinstance(case['observations'], list) or len(case['observations']) > 64:
            raise ValueError('Controlled observations exceed limit')
        for row in case['observations']:
            shape(row, 'cue class name')
            if (type(row['cue']) is not int or not 0 <= row['cue'] < len(case['cues'])
                    or row['class'] not in ('self-identification', 'context-association', 'mention', 'quotation')
                    or not text(row['name'], 128) or row['name'] not in case['cues'][row['cue']]['text']):
                raise ValueError('Controlled observation lacks source support')
    return corpus


def load_corpus(path=CORPUS):
    path = Path(path)
    if path.is_symlink() or not path.is_file():
        raise ValueError('Corpus must be an original regular file')
    with path.open('rb') as source:
        raw = source.read(65537)
    if len(raw) > 65536 or raw.startswith(b'\xef\xbb\xbf'):
        raise ValueError('Speaker corpus encoding or size differs')
    corpus = json.loads(raw.decode('utf-8'), object_pairs_hook=unique_object)
    validate_corpus(corpus)
    digest = hashlib.sha256(raw).hexdigest()
    if digest != CORPUS_SHA256:
        raise ValueError('Frozen speaker corpus identity differs')
    return corpus, digest


def metric(numerator, denominator):
    if type(numerator) is not int or type(denominator) is not int or not 0 <= numerator <= denominator:
        raise ValueError('Metric counts differ')
    return {'numerator': numerator, 'denominator': denominator,
            'value': numerator / denominator if denominator else None}


def score(corpus, rows):
    """Score observations against separate truth, preserving failed cases."""
    validate_corpus(corpus)
    if not isinstance(rows, list) or len(rows) != len(corpus['cases']):
        raise ValueError('Every speaker case requires one terminal outcome')
    indexed = {row.get('id'): row for row in rows if isinstance(row, dict)}
    if len(indexed) != len(rows) or set(indexed) != {case['id'] for case in corpus['cases']}:
        raise ValueError('Speaker case inventory differs')
    counts = {key: 0 for key in OUTCOMES}
    accepted = correct = identifiable = outcome_correct = total = errors = unregistered = unresolved = unresolved_correct = 0
    for case in corpus['cases']:
        row = indexed[case['id']]
        truth = {item['voice']: item for item in case['expected']}
        identifiable += sum(item['speaker'] is not None for item in truth.values())
        total += len(truth)
        unresolved += sum(item['outcome'] in ('unknown', 'ambiguous', 'conflicting') for item in truth.values())
        if row.get('state') == 'error':
            if row.get('observations') != [] or not label(row.get('error')):
                raise ValueError('Failed speaker case lacks bounded error identity')
            errors += 1
            continue
        if row.get('state') != 'completed' or not isinstance(row.get('observations'), list):
            raise ValueError('Speaker case lacks terminal state')
        observed = {item.get('voice'): item for item in row['observations'] if isinstance(item, dict)}
        if len(observed) != len(row['observations']) or set(observed) != set(truth):
            raise ValueError('Speaker observation coverage differs')
        for voice, actual in observed.items():
            observation(actual)
            expected = truth[voice]
            counts[actual['outcome']] += 1
            unregistered += actual['identity_status'] == 'unregistered'
            if actual['speaker'] is not None:
                accepted += 1
                correct += actual['speaker'] == expected['speaker']
            same = actual == expected
            outcome_correct += same
            if expected['outcome'] in ('unknown', 'ambiguous', 'conflicting'):
                unresolved_correct += same
    return {'cases': {'total': len(rows), 'completed': len(rows) - errors, 'errors': errors},
            'voices': total, 'outcome_counts': counts, 'unregistered': unregistered,
            'identity_precision': metric(correct, accepted), 'identity_coverage': metric(correct, identifiable),
            'outcome_accuracy': metric(outcome_correct, total),
            'unresolved_correctness': metric(unresolved_correct, unresolved)}
