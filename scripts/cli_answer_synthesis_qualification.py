#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Installed answer workflows; deterministic transport is never model-quality evidence."""
import copy
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import threading
import time
import uuid
import qualification_timing as bounds

from answer_reference import load_corpus, make_report, score_case
from cli_corpus import finish_server
from cli_media_qualification import pcm_fixture


def fixture_reply(context):
    """Respond only from visible input. Reference rubric labels never enter the provider."""
    evidence = context['evidence']
    question = context['question'].casefold()
    citations = lambda selected, quote=None: [{'evidence_id': e['id'], 'reference': e['reference'],
        'quote': quote or e['text']} for e in selected]
    claim = {'id': 'answer', 'kind': 'paraphrase', 'text': 'The elected evidence does not answer this question.',
             'support': 'unsupported', 'uncertainty': 'No supporting proposition is present in the elected evidence.'}
    state = 'insufficient'
    if 'launch' in question and evidence:
        positive = [e for e in evidence if e['text'] == 'Launch is on Monday.']
        negative = [e for e in evidence if e['text'] == 'Launch is not on Monday.']
        if positive and negative:
            claim.update(text='The sources disagree about Monday.', support='conflicting',
                         uncertainty='One source states Monday and another explicitly rejects Monday.',
                         citations=citations([positive[0], negative[0]]))
            state = 'partial'
        elif positive:
            claim = {'id': 'answer', 'kind': 'quote', 'text': positive[0]['text'], 'support': 'supported',
                     'citations': citations(positive)}
            state = 'partial'
    elif 'budget' in question:
        selected = [e for e in evidence if 'Budget is 100 credits.' in e['text']]
        if selected:
            claim = {'id': 'answer', 'kind': 'quote', 'text': 'Budget is 100 credits.', 'support': 'supported',
                     'citations': citations(selected, 'Budget is 100 credits.')}
            state = 'partial'
    return {'version': context['version'], 'package_id': context['package_id'], 'state': state, 'claims': [claim]}


def qualify(invoke, supplied, workspace, version, *, configuration=None, deadline=None):
    started = time.monotonic()
    deadline = min(deadline if deadline is not None else float('inf'), started + bounds.phase_bound(bounds.SUBPROCESS_SECONDS))
    original_invoke, original_supplied = invoke, supplied
    def remaining():
        allowance = min(deadline - time.monotonic(), bounds.phase_bound(bounds.SUBPROCESS_SECONDS))
        if allowance <= 0: raise TimeoutError('Answer campaign enclosing deadline exceeded')
        return allowance
    def invoke(args, **options):
        options['timeout_seconds'] = min(options.get('timeout_seconds', remaining()), remaining())
        return original_invoke(args, **options)
    def supplied(args, value, **options):
        options['timeout_seconds'] = min(options.get('timeout_seconds', remaining()), remaining())
        return original_supplied(args, value, **options)
    corpus, digest = load_corpus()
    directory = workspace / 'answer workflow fixtures'
    directory.mkdir()
    recordings, originals = {}, {}
    def wait(work, expected=0):
        return invoke(['work', 'wait', work['work_id']], expected=expected)
    def subtitle(path, cues):
        def stamp(ms): return '00:00:%02d,%03d' % (ms // 1000, ms % 1000)
        path.write_text(''.join(str(i + 1) + '\n' + stamp(c['start_ms']) + ' --> ' + stamp(c['end_ms']) + '\n' + c['text'] + '\n\n'
            for i, c in enumerate(cues)), encoding='utf-8', newline='\n')
    for source in corpus['sources']:
        audio, text = directory / (source['id'] + '.wav'), directory / (source['id'] + '.srt')
        pcm_fixture(audio, source['channels'])
        subtitle(text, source['cues'])
        media_id = wait(invoke(['media', 'import', audio, '--new-entry', '--no-pipeline'])['result'])['result']['result']['items'][0]['media_id']
        wait(invoke(['transcript', 'import', text, '--record', media_id])['result'])
        recordings[source['id']] = media_id
        originals[media_id] = invoke(['recordings', 'show', media_id])['result']['recording']
    invoke(['graph', 'publish'])
    control = {'mode': 'valid'}
    entered, release = threading.Event(), threading.Event()
    class Handler(BaseHTTPRequestHandler):
        def do_POST(self):
            length = int(self.headers.get('Content-Length', '0'))
            if not 0 < length <= 128 * 1024:
                self.send_error(413); return
            request = json.loads(self.rfile.read(length))
            mode = control['mode']
            if mode == 'gate':
                entered.set()
                release.wait(max(0, min(bounds.READINESS_SECONDS, deadline - time.monotonic())))
            if mode == 'fail':
                self.send_error(503); return
            body = json.dumps(fixture_reply(request['context'])).encode()
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.send_header('Content-Length', str(len(body)))
            self.end_headers()
            try: self.wfile.write(body)
            except (BrokenPipeError, ConnectionResetError, ConnectionAbortedError): pass
        def log_message(self, *_args): pass
    server, thread = None, None
    elected = configuration is not None
    if not elected:
        server = ThreadingHTTPServer(('127.0.0.1', 0), Handler)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        configuration = {'provider': {'enabled': True, 'adapter': 'insonic-http', 'contract_version': '1',
            'route': 'local', 'endpoint': 'http://127.0.0.1:' + str(server.server_port) + '/answer',
            'model': 'deterministic-answer-fixture', 'limits': {'timeout_ms': 30000}}, 'max_repairs': 1}
    cases, accepted = [], []
    election = None
    try:
        for index, case in enumerate(corpus['cases']):
            request = {'request': {'version': version, 'question': case['question'],
                'filters': {'media_ids': [recordings[s] for s in case['sources']]},
                'budget': {'rounds': 128, 'context_bytes': 16384, 'context_tokens': 16384}},
                'configuration': configuration}
            if index == 0:
                pipeline_id = str(uuid.uuid4())
                preset = 'local' if configuration['provider']['route'] == 'local' else 'connected'
                saved = supplied(['pipelines', 'set', pipeline_id], {'expected_revision': 0,
                    'pipeline': {'id': pipeline_id, 'name': 'Answer evaluation', 'preset': preset,
                        'configuration': {'answer_synthesis': configuration}}})['result']
                inspection = invoke(['pipelines', 'inspect', pipeline_id])['result']
                if inspection['capabilities'] != ['answer-synthesis']:
                    raise ValueError('Answer-only pipeline lost its capability')
                request.pop('configuration')
                request.update(pipeline_id=pipeline_id, pipeline_revision=saved['revision'])
            work = supplied(['answers', 'request'], request)['result']
            terminal = wait(work)['result']
            if terminal['state'] != 'succeeded' or not terminal['result']['accepted']:
                raise ValueError('Answer workflow did not atomically accept')
            answer_id = terminal['result']['answer_id']
            shown = invoke(['answers', 'show', answer_id])['result']
            exported = invoke(['answers', 'export', answer_id])['result']
            if shown != exported or shown['freshness'] != 'current' or not shown['accepted']:
                raise ValueError('Answer inspection and export differ')
            grounded = shown['answer']
            raw = json.dumps(grounded)
            if configuration['provider']['endpoint'] in raw or str(directory) in raw:
                raise ValueError('Grounded export leaked provider or source transport paths')
            package = invoke(['answers', 'show', grounded['package_id']])['result']['package']
            cases.append(score_case(case, recordings, package, grounded['answer']))
            accepted.append(grounded)
            provider = grounded['provider']
            election = {'adapter': configuration['provider']['adapter'], 'model': provider['model'],
                'configuration_digest': provider['configuration_digest'], 'route': grounded['route']}
        for media_id, original in originals.items():
            if invoke(['recordings', 'show', media_id])['result']['recording'] != original:
                raise ValueError('Synthesis altered accepted source authority')
        lifecycle = {'source_unchanged': True}
        if not elected:
            request = {'request': {'version': version, 'question': 'When is launch?',
                'filters': {'media_ids': [recordings['plan-a'], recordings['plan-b']]}, 'budget': {'rounds': 128}},
                'configuration': configuration}
            control['mode'] = 'fail'
            failed = supplied(['answers', 'request'], request)['result']
            failure = wait(failed, expected=4)
            before = invoke(['work', 'show', failed['work_id']])['result']
            if before['state'] != 'failed' or failure['error']['code'] != 'unavailable':
                raise ValueError('Provider failure lost durable classification')
            control['mode'] = 'valid'
            retried = invoke(['answers', 'retry', failed['work_id']])['result']
            wait(retried)
            after = invoke(['work', 'show', failed['work_id']])['result']
            if after != before or retried['work_id'] == failed['work_id'] or retried['retry_of'] != failed['work_id']:
                raise ValueError('Retry replaced prior history')
            control['mode'] = 'gate'
            pending = supplied(['answers', 'request'], request)['result']
            if not entered.wait(min(bounds.READINESS_SECONDS, remaining())): raise ValueError('Cancellation fixture never entered provider')
            invoke(['work', 'cancel', pending['work_id']])
            release.set()
            wait(pending, expected=1)
            if invoke(['work', 'show', pending['work_id']])['result']['state'] != 'cancelled':
                raise ValueError('Cancelled synthesis was accepted')
            control['mode'] = 'valid'
            partial = copy.deepcopy(request)
            partial['request']['budget']['items'] = 1
            limited = wait(supplied(['answers', 'request'], partial)['result'])['result']['result']['answer_id']
            if invoke(['answers', 'show', limited])['result']['answer']['answer']['state'] == 'complete':
                raise ValueError('Bounded partial evidence claimed completeness')
            replacement = directory / 'replacement.srt'
            subtitle(replacement, [{'start_ms': 0, 'end_ms': 1000, 'text': 'Launch is on Friday.'}])
            wait(invoke(['transcript', 'import', replacement, '--record', recordings['plan-a'], '--replace-transcript'])['result'])
            historical = invoke(['answers', 'show', accepted[0]['id']])['result']
            if historical['freshness'] != 'stale' or historical['answer'] != accepted[0]:
                raise ValueError('Changed source lost immutable history or staleness')
            lifecycle.update(failed_retry=True, cancellation=True, partial_budget=True, stale_source=True)
        report = make_report(corpus, digest, cases, mode='elected-model' if elected else 'deterministic',
                             election=election if elected else None)
        report.update(lifecycle=lifecycle, elapsed_seconds=round(time.monotonic() - started, 3))
        return report
    finally:
        release.set()
        if server is not None: finish_server(server, thread)
