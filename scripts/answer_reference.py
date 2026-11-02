#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Bounded synthetic answer corpus and conservative, explicit support rubric."""
import hashlib
import json
from pathlib import Path
import re

ROOT = Path(__file__).resolve().parents[1]
MANIFEST = ROOT / 'tests/fixtures/answers/manifest.json'
LABEL = re.compile(r'[a-z][a-z0-9-]{0,63}')


def unique(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError('Duplicate answer corpus member')
        result[key] = value
    return result


def shape(value, keys):
    if not isinstance(value, dict) or set(value) != set(keys):
        raise ValueError('Invalid answer corpus shape')


def text(value, maximum=4096):
    if not isinstance(value, str) or not value.strip() or len(value) > maximum:
        raise ValueError('Invalid answer corpus text')


def label(value):
    if not isinstance(value, str) or not LABEL.fullmatch(value):
        raise ValueError('Invalid answer corpus identifier')


def load_corpus(path=MANIFEST, *, root=ROOT):
    path, root = Path(path).resolve(), Path(root).resolve()
    if not path.is_relative_to(root) or not path.is_file() or path.stat().st_size > 65536:
        raise ValueError('Answer corpus is missing, oversized or outside its root')
    raw = path.read_bytes()
    if len(raw) > 65536:
        raise ValueError('Answer corpus changed beyond its bound')
    value = json.loads(raw.decode('utf-8'), object_pairs_hook=unique)
    shape(value, ['kind', 'corpus_version', 'license', 'provenance', 'sources', 'cases'])
    if value['kind'] != 'answer-evaluation-corpus' or type(value['corpus_version']) is not int or value['corpus_version'] != 1 or value['license'] != 'Apache-2.0':
        raise ValueError('Unsupported answer corpus authority')
    text(value['provenance'])
    sources = {}
    if not isinstance(value['sources'], list) or not 1 <= len(value['sources']) <= 32:
        raise ValueError('Invalid answer source count')
    families = {}
    for source in value['sources']:
        shape(source, ['id', 'audio_family', 'channels', 'cues'])
        label(source['id']); label(source['audio_family'])
        if source['id'] in sources or type(source['channels']) is not int or not 1 <= source['channels'] <= 8:
            raise ValueError('Invalid answer source identity')
        if source['audio_family'] in families and families[source['audio_family']] != source['channels']:
            raise ValueError('Duplicate family must use identical PCM')
        families[source['audio_family']] = source['channels']
        if not isinstance(source['cues'], list) or not 1 <= len(source['cues']) <= 16:
            raise ValueError('Invalid answer cue count')
        for cue in source['cues']:
            shape(cue, ['start_ms', 'end_ms', 'text'])
            if type(cue['start_ms']) is not int or type(cue['end_ms']) is not int or not 0 <= cue['start_ms'] < cue['end_ms'] <= 1000:
                raise ValueError('Invalid synthetic source interval')
            text(cue['text'])
        sources[source['id']] = source
    if not isinstance(value['cases'], list) or not 1 <= len(value['cases']) <= 32:
        raise ValueError('Invalid answer case count')
    seen = set()
    for case in value['cases']:
        shape(case, ['id', 'question', 'sources', 'independent_families', 'expected', 'uncertainty'])
        label(case['id']); text(case['question'])
        if case['id'] in seen or not (case['uncertainty'] is False or isinstance(case['uncertainty'], str) and case['uncertainty'] in ('conflicting', 'unsupported')):
            raise ValueError('Invalid answer case identity')
        seen.add(case['id'])
        if not isinstance(case['sources'], list) or not case['sources'] or len(set(case['sources'])) != len(case['sources']) or any(s not in sources for s in case['sources']):
            raise ValueError('Invalid elected corpus sources')
        expected_families = len({sources[s]['audio_family'] for s in case['sources']})
        if type(case['independent_families']) is not int or case['independent_families'] != expected_families:
            raise ValueError('Reference independence count differs from source authority')
        if not isinstance(case['expected'], list) or len(case['expected']) > 16:
            raise ValueError('Invalid proposition count')
        propositions = set()
        for proposition in case['expected']:
            shape(proposition, ['id', 'aliases', 'sources'])
            label(proposition['id'])
            if proposition['id'] in propositions:
                raise ValueError('Duplicate proposition')
            propositions.add(proposition['id'])
            if not isinstance(proposition['aliases'], list) or not 1 <= len(proposition['aliases']) <= 16:
                raise ValueError('Invalid reference paraphrase inventory')
            for alias in proposition['aliases']: text(alias)
            if not isinstance(proposition['sources'], list) or not proposition['sources'] or any(s not in case['sources'] for s in proposition['sources']):
                raise ValueError('Reference support is not elected')
    return value, hashlib.sha256(raw).hexdigest()


def normalized(text):
    return ' '.join(text.casefold().split()).rstrip('.')


def metric(numerator, denominator, *, zero=False):
    if denominator == 0:
        return {'numerator': numerator, 'denominator': 0, 'value': None,
                'criterion': 'zero' if zero else 'all', 'passed': numerator == 0, 'reason': 'not-applicable'}
    return {'numerator': numerator, 'denominator': denominator, 'value': numerator / denominator,
            'criterion': 'zero' if zero else 'all', 'passed': numerator == 0 if zero else numerator == denominator}


def score_case(case, recordings, package, answer):
    """Exact rubric aliases are conservative labels, never a general entailment judge."""
    items = {item['id']: item for item in package['items']}
    visible = {n['id']: n['text'] for n in package['context']['nodes'] if n['level'] == 'cue'}
    required = {recordings[s] for s in case['sources']}
    retrieved = {item['reference']['recording_id'] for item in items.values()}
    families = {item['family'] for item in items.values()}
    independent = len(families) == case['independent_families'] == package['independent_families']
    valid, faithful, total = 0, 0, 0
    supported, unsupported, matched, failures = 0, 0, set(), []
    for claim in answer['claims']:
        references, claim_valid = set(), True
        citations = claim.get('citations', [])
        for citation in citations:
            total += 1
            item = items.get(citation.get('evidence_id'))
            member = bool(item and citation.get('reference') == item['reference'] and item['id'] in visible)
            quote = citation.get('quote')
            quote_ok = bool(member and isinstance(quote, str) and quote and quote in visible[item['id']] and quote in item['text'])
            valid += int(member)
            faithful += int(quote_ok)
            claim_valid &= member and quote_ok
            if member: references.add(item['reference']['recording_id'])
        if claim.get('kind') not in ('quote', 'paraphrase'):
            claim_valid = False
        if claim.get('kind') == 'quote' and not any(c.get('quote') == claim.get('text') for c in citations):
            claim_valid = False
        if claim.get('support') == 'supported':
            supported += 1
            covered = [p['id'] for p in case['expected']
                       if normalized(claim.get('text', '')) in {normalized(a) for a in p['aliases']}
                       and references & {recordings[s] for s in p['sources']} and claim_valid]
            if covered:
                matched.update(covered)
            else:
                unsupported += 1
                failures.append({'claim_id': claim.get('id'), 'reason': 'not-supported-by-frozen-rubric', 'claim': claim})
        elif not claim.get('uncertainty') or answer.get('state') == 'complete':
            failures.append({'claim_id': claim.get('id'), 'reason': 'missing-explicit-uncertainty', 'claim': claim})
    uncertain = answer.get('state') in ('partial', 'insufficient') and any(
        c.get('support') == case['uncertainty'] and c.get('uncertainty') for c in answer['claims'])
    metrics = {
        'retrieval_coverage': metric(len(required & retrieved), len(required)),
        'independence': metric(int(independent), 1),
        'citation_validity': metric(valid, total),
        'quote_fidelity': metric(faithful, total),
        'support_coverage': metric(len(matched), len(case['expected'])),
        'unsupported_claims': metric(unsupported, supported, zero=True),
        'uncertainty': metric(int(uncertain) if case['uncertainty'] else 0, int(bool(case['uncertainty']))),
    }
    for name, value in metrics.items():
        if not value['passed']: failures.append({'metric': name, 'reason': 'threshold-not-met'})
    if answer.get('package_id') != package['id']:
        failures.append({'reason': 'foreign-package'})
    return {'id': case['id'], 'passed': not failures and all(m['passed'] for m in metrics.values()),
            'metrics': metrics, 'failures': failures, 'answer': answer}


def make_report(corpus, digest, cases, *, mode, election):
    if mode not in ('deterministic', 'elected-model'):
        raise ValueError('Unknown evaluation mode')
    if mode == 'elected-model' and (not isinstance(election, dict) or not all(election.get(k) for k in ('adapter', 'model', 'configuration_digest', 'route'))):
        raise ValueError('Elected evaluation requires exact route/model/configuration identity')
    if mode == 'deterministic' and election is not None:
        raise ValueError('Fixture report cannot claim a model election')
    complete = len(cases) == len(corpus['cases']) and {c['id'] for c in cases} == {c['id'] for c in corpus['cases']}
    return {'kind': 'answer-evaluation-report', 'report_version': 1, 'corpus_version': corpus['corpus_version'],
            'corpus_sha256': digest, 'mode': mode, 'model_quality': 'measured-frozen-rubric' if mode == 'elected-model' else 'not-evaluated',
            'election': election, 'passed': complete and all(c['passed'] for c in cases), 'cases': cases,
            'limitations': ['Synthetic small corpus does not establish general answer accuracy.',
                            'Support uses exact normalized frozen proposition/paraphrase labels; unrecognized valid paraphrases fail conservatively.',
                            'Citation validity and quote fidelity do not establish semantic entailment.',
                            'Provenance families do not certify acoustic independence or unknown shared origins.']}
