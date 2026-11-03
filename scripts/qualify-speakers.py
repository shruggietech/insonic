#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Measure synthetic speaker semantics and elected real CPU/NVIDIA profiles."""
import argparse
from contextlib import nullcontext
from datetime import datetime, timezone
import importlib.util
import json
import os
from pathlib import Path
import sys
import tempfile
import time

import acoustic_reference
import ci_timing_policy as phase
import qualification_timing as bounds
import speaker_quality as quality
import speaker_reference

ROOT = Path(__file__).resolve().parents[1]


def journey_module():
    spec = importlib.util.spec_from_file_location('speaker_journey', ROOT / 'scripts/qualify-journey.py')
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def utc():
    return datetime.now(timezone.utc).isoformat()


def prepare(journey, module, args):
    version = journey.receipt['product_version']
    if journey.engine_child([journey.cli, 'version'], maximum=bounds.READINESS_SECONDS).decode().strip() != version:
        raise ValueError('Selected CLI differs from current version')
    journey.command('workspace', 'init', journey.workspace, workspace=False)
    journey.start_runtime()
    graph = journey.command('graph', 'capabilities')
    if graph.get('state') != 'available' or graph.get('adapter_id') != 'ladybugdb':
        raise ValueError('Speaker qualification requires the native graph')
    journey.receipt['graph'] = {key: graph.get(key) for key in ('state', 'adapter_id', 'backend_version')}
    tools = dict(journey.tools, schema_version=version)
    journey.command('media', 'tools', journey.payload(tools))
    processing = {'threads': 2, 'timeout_ms': 600000, 'max_duration_us': 60000000,
                  'max_input_bytes': 67108864,
                  'worker': {'path': str(journey.worker), 'sha256': module.engine.sha(journey.worker)}}
    journey.command('processing', 'tools', journey.payload({'kind': 'processing-tools', 'schema_version': version,
        'cueson': {'executable': str(journey.cueson), 'executable_sha256': module.engine.sha(journey.cueson)},
        'processing': processing}))
    journey.receipt['companion_sha256'] = {**{name: tools[name]['sha256'] for name in ('ffmpeg', 'ffprobe', 'exiftool')},
                                         'cueson': module.engine.sha(journey.cueson)}
    if args.mode != 'real': return None
    interpreter = Path(args.diarization_python).resolve(strict=True)
    inspected = journey.input('workers', 'inspect', value={'role': 'diarization', 'python_path': str(interpreter)})
    election = journey.input('workers', 'plan', value={'role': 'diarization', 'mode': 'reuse', 'python_path': str(interpreter)})
    setup = journey.finish(journey.input('workers', 'setup', value=election['election']))
    journey.receipt['acoustic_python_sha256'] = module.engine.sha(interpreter)
    journey.receipt['worker_environment_setup'] = {key: setup.get(key) for key in ('state', 'environment_id', 'role')}
    # The product captures and validates the exact reusable environment. Do not
    # copy interpreter paths, raw inventory or credentials into public receipts.
    root = module.engine.verify_model('diarize', args.diarization_model)
    prepared = journey.input('models', 'prepare', value={'choice': 'community-1', 'local_directory': str(root), 'acquire': True})
    if prepared.get('acquisition_work_id'):
        journey.finish({'work_id': prepared['acquisition_work_id']})
    pin = module.engine.PINS['diarize']
    journey.receipt['acoustic_model'] = {'id': pin['id'], 'revision': pin['revision'], 'license': pin['license'],
        'model_id': prepared['model_id'], 'files': {role: {'size_bytes': value[0], 'sha256': value[1]}
                                                for role, value in pin['files'].items()}}
    ready = journey.command('processing', 'readiness')
    devices = ready['environments']['diarization']['devices']
    if not all(devices.get(device, {}).get('available') is True for device in ('cpu', 'cuda')):
        raise ValueError('CPU and NVIDIA prerequisites are not both available')
    journey.receipt['readiness'] = {'cpu': True, 'cuda': True, 'includes_inference': False}
    return prepared['model_id']


def campaign(args):
    configuration = None
    if args.configuration:
        with Path(args.configuration).open('rb') as source:
            raw = source.read(65537)
        if len(raw) > 65536: raise ValueError('Provider configuration exceeds limit')
        configuration = json.loads(raw, object_pairs_hook=speaker_reference.unique_object)
    quality.validate_election(args.mode, args.elect_models, configuration, os.environ)
    if args.mode == 'real' and not all((args.diarization_python, args.diarization_model, args.model_digest)):
        raise ValueError('Real qualification requires exact acoustic/model selections')
    if args.mode != 'real' and any((args.diarization_python, args.diarization_model, args.model_digest)):
        raise ValueError('Deterministic qualification cannot select acoustic/model inputs')
    output = Path(args.output).resolve()
    if output.exists(): raise ValueError('Retain the existing receipt and select a new output path')
    output.parent.mkdir(parents=True, exist_ok=True)
    module = journey_module()
    module.validate_selection(args)
    original_mode = args.mode
    args.mode = 'real' if original_mode == 'real' else 'controlled'
    # Journey's controlled mode is a harness detail; our public mode remains
    # deterministic, and no controlled acoustic model is executed here.
    args.timeout = phase.QUALIFICATION_LIMIT_SECONDS
    preparation_started, preparation_utc = time.monotonic(), utc()
    qualification_started = None
    journey = None
    receipt = {'kind': 'insonic-speaker-quality-v1', 'mode': original_mode, 'state': 'preparing',
               'preparation_started_utc': preparation_utc, 'required_check': original_mode == 'deterministic'}
    primary = None
    temporary = tempfile.TemporaryDirectory(prefix='speaker-qualification-', dir=output.parent)
    try:
        journey = module.Journey(args, Path(temporary.name))
        receipt.update(journey.receipt)
        journey.receipt = receipt
        receipt.update(kind='insonic-speaker-quality-v1', mode=original_mode,
                       required_check=original_mode == 'deterministic')
        receipt.pop('device', None)
        if original_mode == 'real' and receipt['source_dirty']:
            raise ValueError('Real acceptance requires a clean source revision')
        contextual_corpus, contextual_digest = speaker_reference.load_corpus()
        corpus = acoustic_reference.load_frozen_corpus(args.reference_corpus)
        quality.independent_sources(corpus['clips'])
        model_id = prepare(journey, module, args)
        before = quality.model_snapshot(configuration) if configuration else None
        if before: quality.same_model(before, before, args.model_digest)
        provider = quality.controlled_provider() if original_mode == 'deterministic' else nullcontext((None, configuration))
        with provider as (control, effective):
            receipt['configuration_sha256'] = quality.value_digest(effective)
            # The endpoint is local routing, not a quality variable; retain the
            # elected limits/model and distinguish the controlled provider.
            receipt['contextual_configuration'] = {key: value for key, value in effective.items() if key != 'endpoint'}
            receipt['contextual_model'] = before
            receipt['contextual_corpus_sha256'] = contextual_digest
            receipt['preparation_seconds'] = time.monotonic() - preparation_started
            qualification_started = time.monotonic()
            receipt.update(state='running', qualification_started_utc=utc(), target_seconds=phase.QUALIFICATION_TARGET_SECONDS,
                           hard_limit_seconds=phase.QUALIFICATION_LIMIT_SECONDS,
                           finalization_reserve_seconds=phase.FINALIZATION_RESERVE_SECONDS)
            journey.deadline = qualification_started + phase.QUALIFICATION_LIMIT_SECONDS - phase.FINALIZATION_RESERVE_SECONDS
            def checkpoint(): module.write_json(output, receipt)
            checkpoint()
            receipt['contextual'] = quality.contextual(journey, effective, controlled=control, corpus=contextual_corpus)
            checkpoint()
            if original_mode == 'real':
                acoustic, speakers, recordings = quality.acoustic(journey, corpus, model_id, checkpoint=checkpoint)
                receipt['acoustic'] = acoustic
                receipt['integrated'] = quality.integrated(journey, effective, corpus, model_id, speakers, recordings)
                after = quality.model_snapshot(configuration, timeout=journey.remaining(10))
                quality.same_model(before, after, args.model_digest)
                receipt['contextual_model_unchanged'] = True
            else:
                receipt['acoustic_quality'] = 'not-measured'
            if receipt['contextual']['metrics']['cases']['errors']:
                raise ValueError('Contextual campaign has failed cases')
            if original_mode == 'deterministic' and receipt['contextual']['metrics']['outcome_accuracy']['value'] != 1:
                raise ValueError('Controlled speaker outcomes differ from frozen truth')
            if original_mode == 'real' and any(row['state'] != 'completed' for row in receipt['acoustic']['devices']):
                raise ValueError('Real acoustic campaign has failed rows')
            revision, dirty, identity = module.harness_identity()
            if (revision != receipt['source_revision'] or original_mode == 'real' and dirty
                    or identity != receipt.get('source_inputs_sha256', receipt.get('harness_inputs_sha256'))
                    or module.engine.sha(journey.cli) != receipt['cli_sha256']
                    or module.engine.sha(journey.worker) != receipt['worker_sha256']):
                raise ValueError('Qualification source or executable changed during execution')
            receipt['inputs_unchanged'] = True
    except BaseException as error:
        primary = error
        receipt.update(state='failed', failure_type=type(error).__name__)
    finally:
        if journey is not None and journey.runtime_tree is not None:
            try:
                if qualification_started is not None: journey.deadline = qualification_started + phase.QUALIFICATION_LIMIT_SECONDS
                if primary is None:
                    journey.idle_runtime()
                else:
                    journey.runtime_tree.kill()
                    journey.runtime.wait(timeout=bounds.CLEANUP_SECONDS)
                    if not journey.runtime_tree.wait_for_descendants(bounds.CLEANUP_SECONDS):
                        raise ValueError('Owned runtime descendants remain')
                    for reader in journey.runtime_readers: reader.join(bounds.CLEANUP_SECONDS)
                    if any(reader.is_alive() for reader in journey.runtime_readers):
                        raise ValueError('Owned runtime reader remains')
                    journey.runtime_tree.close()
                receipt['owned_runtime_released'] = True
            except BaseException as error:
                receipt['cleanup_error'] = type(error).__name__
                if primary is None: primary = error
        try:
            temporary.cleanup()
            receipt['temporary_workspace_removed'] = not Path(temporary.name).exists()
        except BaseException as error:
            receipt['temporary_cleanup_error'] = type(error).__name__
            if primary is None: primary = error
        terminal = time.monotonic()
        receipt['finished_utc'] = utc()
        receipt['combined_seconds'] = terminal - preparation_started
        if qualification_started is not None:
            receipt['qualification_seconds'] = terminal - qualification_started
            receipt['target_exceeded'] = receipt['qualification_seconds'] > phase.QUALIFICATION_TARGET_SECONDS
            if receipt['qualification_seconds'] > phase.QUALIFICATION_LIMIT_SECONDS and primary is None:
                primary = ValueError('Qualification hard limit exceeded')
        else:
            receipt['preparation_seconds'] = receipt['combined_seconds']
        receipt['state'] = 'failed' if primary else 'passed'
        if primary: receipt['failure_type'] = type(primary).__name__
        module.write_json(output, receipt)
    if primary is not None: raise primary
    return receipt


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--mode', required=True, choices=('deterministic', 'real'))
    parser.add_argument('--elect-models', action='store_true')
    parser.add_argument('--configuration', type=Path)
    parser.add_argument('--model-digest')
    parser.add_argument('--cli', type=Path)
    parser.add_argument('--tools', type=Path)
    parser.add_argument('--cueson', type=Path)
    parser.add_argument('--package', type=Path)
    parser.add_argument('--package-receipt', type=Path)
    parser.add_argument('--revision')
    parser.add_argument('--diarization-python', type=Path)
    parser.add_argument('--diarization-model', type=Path)
    parser.add_argument('--reference-corpus', type=Path, default=ROOT / 'tests/fixtures/acoustic/manifest.json')
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    receipt = campaign(args)
    print(json.dumps({key: receipt[key] for key in ('state', 'mode', 'preparation_seconds',
                                                   'qualification_seconds', 'combined_seconds')}))
    return 0


if __name__ == '__main__': sys.exit(main())
