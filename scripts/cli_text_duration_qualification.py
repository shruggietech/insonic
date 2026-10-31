#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Installed duration and Unicode contracts using an owned disposable catalog."""
import hashlib
import json
from pathlib import Path
import time
import uuid


def qualify(invoke, workspace, version):
    started = time.monotonic()
    workspace = Path(workspace)
    directory = workspace / 'text and duration contract fixtures'
    directory.mkdir()
    input_file = directory / 'term input.json'
    manifest_file = directory / 'unicode model.json'
    jobs = {}
    terms = {}
    models = {}

    def supplied(args, data, **kwargs):
        input_file.write_text(json.dumps(data, ensure_ascii=False) + '\n', encoding='utf-8', newline='\n')
        return invoke([*args, '--input', input_file], **kwargs)

    def rejected(response, label):
        if response.get('error', {}).get('code') != 'invalid_request' or response.get('result') is not None:
            raise ValueError(label + ' did not reject before creating a result')

    # Read both the live result and the subsequent catalog export. A successful
    # parser/schema check alone cannot prove that the durable job accepted it.
    for duration in [1, 60000, 60001, 600000]:
        created = invoke(['jobs', 'start', duration])['result']
        job_id = created['job_id']
        try:
            if created['duration_ms'] != duration:
                raise ValueError('Accepted job duration changed')
        finally:
            # The one-millisecond case may already have finished by this call.
            invoke(['jobs', 'cancel', job_id])
        shown = invoke(['jobs', 'show', job_id])['result']
        if shown['duration_ms'] != duration or shown['state'] not in ['cancelled', 'succeeded']:
            raise ValueError('Job duration/state was not durably preserved after immediate cancellation')
        jobs[job_id] = duration
    for duration in [0, 600001]:
        rejected(invoke(['jobs', 'start', duration], expected=2), 'Out-of-range synthetic job')

    alphabets = {'ascii': 'a', 'composed_accent': 'é', 'decomposed_accent': 'e\u0301',
                 'cjk': '漢', 'emoji': '😀'}

    def exact(unit, count):
        # Python len and the public schema both count code points. In particular,
        # the combining accent remains a separate code point and original byte.
        return unit * (count // len(unit)) + unit[:count % len(unit)]

    for label, unit in alphabets.items():
        term_id = str(uuid.uuid4())
        expected_revision = 0
        for count in [511, 512]:
            canonical = exact(unit, count)
            term = {'id': term_id, 'canonical': canonical, 'variants': [canonical],
                    'language': exact(unit, 80), 'context': exact(unit, 256), 'state': 'active', 'provenance': {}}
            changed = supplied(['terms', 'set', term_id], {'expected_revision': expected_revision, 'term': term})['result']
            expected_revision = changed['revision']
            shown = invoke(['terms', 'show', term_id])['result']
            for key in ['canonical', 'variants', 'language', 'context']:
                if shown[key] != term[key]:
                    raise ValueError(label + ' term text was normalized or changed')
            if shown['revision'] != expected_revision or len(shown['canonical']) != count:
                raise ValueError(label + ' term character boundary did not reach the catalog')
        # Rejection must leave the exact previous accepted value and revision.
        invalid = dict(term, canonical=exact(unit, 513))
        rejected(supplied(['terms', 'set', term_id], {'expected_revision': expected_revision, 'term': invalid}, expected=2),
                 label + ' oversized term')
        retained = invoke(['terms', 'show', term_id])['result']
        if retained['canonical'] != term['canonical'] or retained['revision'] != expected_revision:
            raise ValueError(label + ' rejected term changed current authority')
        terms[term_id] = term

        manifest = {'kind': 'base-model-manifest', 'schema_version': version, 'name': exact(unit, 256),
                    'model_version': exact(unit, 256), 'upstream_revision': exact(unit, 512),
                    'capabilities': ['transcription'], 'license': 'synthetic Unicode registration fixture',
                    'files': [{'role': 'weights', 'sha256': hashlib.sha256(b'fixture').hexdigest(), 'size': 7,
                               'url': 'http://127.0.0.1:9/not-downloaded', 'local_http': True}]}
        manifest_file.write_text(json.dumps(manifest, ensure_ascii=False) + '\n', encoding='utf-8', newline='\n')
        queued = invoke(['models', 'register', manifest_file])['result']
        completed = invoke(['work', 'wait', queued['work_id']])['result']
        if completed['state'] != 'succeeded':
            raise ValueError(label + ' schema-valid model did not finish registration')
        model_id = completed['result']['model_id']
        shown = invoke(['models', 'show', model_id])['result']
        if shown['name'] != manifest['name'] or shown['version'] != manifest['model_version'] or shown['manifest'] != manifest:
            raise ValueError(label + ' model metadata changed between admission and inspection')
        if shown['registry_state'] != 'registered' or shown['readiness'] != 'not_acquired':
            raise ValueError('Registration performed an unelected acquisition')
        invalid = dict(manifest, name=exact(unit, 257))
        manifest_file.write_text(json.dumps(invalid, ensure_ascii=False) + '\n', encoding='utf-8', newline='\n')
        rejected(invoke(['models', 'register', manifest_file], expected=2), label + ' oversized model identity')
        retained = invoke(['models', 'show', model_id])['result']
        if retained['manifest'] != manifest or retained['manifest_digest'] != shown['manifest_digest']:
            raise ValueError(label + ' rejected model changed immutable metadata/digest')
        models[model_id] = {'manifest': manifest, 'digest': shown['manifest_digest']}

    # Export is an independent persisted observation, including exact operational
    # duration rows and original model/term text rather than their human labels.
    snapshot_file = directory / 'persisted unicode and duration.json'
    invoke(['catalog', 'export', snapshot_file])
    snapshot = json.loads(snapshot_file.read_text(encoding='utf-8'))
    if snapshot['catalog_schema'] < 10:
        raise ValueError('Installed catalog predates the widened duration and Unicode contract')
    exported_jobs = {row[0]: row for table in snapshot['state'] if table['name'] == 'job' for row in table['rows']}
    for identity, duration in jobs.items():
        row = exported_jobs.get(identity)
        if not row or row[2] != duration or row[4] not in ['cancelled', 'succeeded']:
            raise ValueError('Export lost a cancelled boundary job or its exact duration')
    exported_terms = {term['id']: term for term in snapshot['records']['terms']}
    for identity, expected in terms.items():
        actual = exported_terms.get(identity)
        if not actual or any(actual[key] != expected[key] for key in ['canonical', 'variants', 'language', 'context']):
            raise ValueError('Export changed accepted Unicode term bytes')
    exported_models = {model['id']: model for model in snapshot['records']['base_models']}
    for identity, expected in models.items():
        actual = exported_models.get(identity)
        if not actual or actual['manifest'] != expected['manifest'] or actual['digest'] != expected['digest']:
            raise ValueError('Export changed accepted Unicode model identities or digest')

    return {'state': 'passed', 'issues': [132, 133], 'accepted_job_durations_ms': list(jobs.values()),
            'rejected_job_durations_ms': [0, 600001], 'unicode_alphabets': list(alphabets),
            'term_character_boundaries': [511, 512, 513], 'model_name_character_boundaries': [256, 257],
            'stored_model_identities': list(models), 'persisted_catalog_schema': snapshot['catalog_schema'],
            'elapsed_seconds': round(time.monotonic() - started, 3)}
