#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Actual CLI answer and speaker journeys on disposable alternative services."""
import argparse
import base64
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import tempfile
import urllib.parse
import urllib.request
import uuid

from cli_contract_qualification import capture_invocation
from process_tree import ProcessTree
import qualification_timing as bounds
import qualify as native


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--executable', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    executable = args.executable.resolve(strict=True)
    selected = {k: os.environ[k] for k in ('INSONIC_FIXTURE_POSTGRES', 'INSONIC_FIXTURE_S3', 'INSONIC_FIXTURE_ARCADE')}
    pg = urllib.parse.urlparse(selected['INSONIC_FIXTURE_POSTGRES'])
    if not pg.hostname or not pg.username or not pg.password: raise ValueError('Explicit PostgreSQL fixture is required')
    session_id, graph_id = str(uuid.uuid4()), str(uuid.uuid4())
    schema, database, bucket = 'answer_' + uuid.uuid4().hex, 'answer' + uuid.uuid4().hex, 'answer-' + uuid.uuid4().hex
    s3 = 'http://' + selected['INSONIC_FIXTURE_S3']
    # These uniquely named resources belong to the elected disposable service
    # campaign; its container teardown removes them together after diagnostics.
    values = {session_id: {'username': pg.username, 'password': pg.password},
              graph_id: {'username': 'root', 'password': 'fixture-only-password'}}
    env = dict(os.environ)
    env.update(native.native_env())
    env['INSONIC_SESSION_CREDENTIALS'] = json.dumps(values)
    for marker in ('CI', 'GITHUB_ACTIONS', 'TF_BUILD', 'BUILD_BUILDID', 'JENKINS_URL'): env.pop(marker, None)
    with tempfile.TemporaryDirectory(prefix='answer-backends-') as temporary:
        folder = Path(temporary)
        workspace = folder / 'library'
        plan = {'kind': 'backend-plan', 'schema_version': (root / 'VERSION').read_text().strip(),
            'display_name': 'Disposable answer qualification', 'profiles': {
                'storage': {'adapter_id': 's3', 'configuration': {'endpoint': s3, 'bucket': bucket,
                    'region': 'us-east-1', 'addressing_style': 'path', 'authentication': 'anonymous'}},
                'catalog': {'adapter_id': 'postgresql', 'configuration': {'host': pg.hostname, 'port': pg.port or 5432,
                    'database': pg.path.lstrip('/'), 'schema': schema, 'tls_mode': 'local', 'credential_id': session_id}},
                'graph': {'adapter_id': 'arcadedb', 'configuration': {'endpoint': selected['INSONIC_FIXTURE_ARCADE'],
                    'database': database, 'credential_id': graph_id, 'preferred_dialect': 'arcade-opencypher'}}}}
        def provision(path, election):
            storage = election['profiles']['storage']['configuration']
            graph = election['profiles']['graph']['configuration']
            with urllib.request.urlopen(urllib.request.Request(storage['endpoint'] + '/' + storage['bucket'],
                    data=b'', method='PUT'), timeout=bounds.REQUEST_SECONDS) as response:
                if response.status not in (200, 201): raise ValueError('Owned fixture bucket was not created')
            graph_request = urllib.request.Request(graph['endpoint'] + '/api/v1/server',
                data=json.dumps({'command': 'create database ' + graph['database']}).encode(),
                headers={'Content-Type': 'application/json', 'Authorization': 'Basic ' + base64.b64encode(b'root:fixture-only-password').decode()})
            with urllib.request.urlopen(graph_request, timeout=bounds.REQUEST_SECONDS) as response:
                if response.status not in (200, 201): raise ValueError('Owned graph database was not created')
            plan_path = folder / 'plan.json'
            plan_path.write_text(json.dumps(election) + '\n', encoding='utf-8', newline='\n')
            stdout, stderr, code, _ = capture_invocation(ProcessTree,
                [str(executable), '--json', 'workspace', 'provision', str(path), '--input', str(plan_path),
                 '--credential-backend', 'session'], cwd=root, env=env, scoped=False,
                input=None, timeout_seconds=bounds.SERVICE_SECONDS)
            if code: raise ValueError('Alternative workspace provisioning failed: ' + stderr.decode()[:2000])
        provision(workspace, plan)
        cueson = Path(env['CUESON_EXECUTABLE'])
        processing = folder / 'processing.json'
        with cueson.open('rb') as source: cueson_hash = hashlib.file_digest(source, 'sha256').hexdigest()
        processing.write_text(json.dumps({'kind': 'processing-tools', 'schema_version': plan['schema_version'],
            'cueson': {'executable': str(cueson), 'executable_sha256': cueson_hash}, 'processing': {}}) + '\n',
            encoding='utf-8', newline='\n')
        spec = importlib.util.spec_from_file_location('answer_campaign', root / 'scripts/qualify-answers.py')
        module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module)
        receipt = module.campaign(executable, workspace, args.output, existing=True, environment=env,
            media_tools=root / 'build/native/media-tools.json', processing_tools=processing)
        # The answer evaluation requires an empty library. Keep the speaker
        # restart campaign in its own workspace and backend namespaces so graph
        # ownership and qualification state cannot cross campaigns.
        speaker_workspace = folder / 'speaker library'
        plan['profiles']['catalog']['configuration']['schema'] = schema + '_speakers'
        plan['profiles']['storage']['configuration']['bucket'] = bucket + '-speakers'
        plan['profiles']['graph']['configuration']['database'] = database + '_speakers'
        provision(speaker_workspace, plan)
        # Exact prepared tools are selected before the explicitly owned runtime
        # starts; no local or default catalog replaces the elected backend plan.
        (speaker_workspace / '.insonic/media-tools.json').write_bytes((root / 'build/native/media-tools.json').read_bytes())
        (speaker_workspace / '.insonic/processing-tools.json').write_bytes(processing.read_bytes())
        from cli_speaker_workflow_qualification import qualify_backend
        speaker = qualify_backend(executable, speaker_workspace, folder, env, plan['schema_version'])
        receipt['backends'] = {'catalog': 'postgresql', 'storage': 's3', 'graph': 'arcadedb'}
        receipt['speaker_workflow'] = speaker
        args.output.write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8', newline='\n')
        if not receipt['passed']: raise ValueError('Alternative-backend answer evaluation failed')
    print(json.dumps({'passed': True, 'backends': receipt['backends'], 'model_quality': receipt['model_quality']}))


if __name__ == '__main__': main()
