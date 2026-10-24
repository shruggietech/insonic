// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import { validateAggregatePolicy } from '../scripts/qualification-timing.mjs';

import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { parse } from 'yaml';
import { sourceReadiness, evaluateResults, measureTiming, inspectAttempt, attemptMetadata, waitForQualificationInputs } from '../scripts/ci-acceptance.mjs';

test('aggregate policy rejects invalid units, unordered limits and invalid finalization reserves', () => {
  const policy = { qualification_target_seconds: 1800, qualification_limit_seconds: 3600, finalization_reserve_seconds: 60 };
  assert.deepEqual(validateAggregatePolicy(policy), policy);
  for (const changed of [null, [], {}, { ...policy, extra: 1 }, { ...policy, qualification_target_seconds: 0 },
    { ...policy, qualification_limit_seconds: '3600' }, { ...policy, qualification_target_seconds: 3601 },
    { ...policy, finalization_reserve_seconds: 3600 }, { ...policy, finalization_reserve_seconds: 1.5 }]) {
    assert.throws(() => validateAggregatePolicy(changed), /Invalid aggregate/);
  }
});

const flags = value => Object.fromEntries(['core', 'cli', 'native', 'adapters'].map(k => [k, String(value)]));
function results(full = true, smoke = false) {
  const elected = ['scope', 'readiness', 'foundation', 'docs-build', 'docs'];
  const conditional = ['media-linux', 'media-windows', 'dependencies-windows', 'core', 'core-windows', 'qualify-linux', 'qualify-windows', 'package-linux', 'package-windows', 'adapters', 'candidate'];
  const needs = Object.fromEntries([...elected, ...conditional, 'live-site'].map(k => [k, { result: elected.includes(k) || (full && conditional.includes(k)) || (k === 'live-site' && smoke) ? 'success' : 'skipped', outputs: {} }]));
  needs.scope.outputs = flags(full);
  needs.readiness.outputs.ready = 'true';
  return needs;
}

test('source readiness accepts only complete successful warm and OS-specific cold paths', () => {
  const warm = { probe: { result: 'success', outputs: { complete: 'true' } }, groups: { result: 'skipped' }, formats: { result: 'skipped' }, codec: { result: 'skipped' }, ffmpeg: { result: 'skipped' } };
  assert.equal(sourceReadiness(warm, 'windows-2022'), 'warm');
  assert.throws(() => sourceReadiness(warm, 'windows-2022', { campaign: true, attempt: '1' }), /cold source/);
  assert.equal(sourceReadiness(warm, 'windows-2022', { campaign: true, attempt: '2' }), 'warm');
  const cold = structuredClone(warm); cold.probe.outputs.complete = 'false';
  for (const id of ['groups', 'formats', 'codec', 'ffmpeg']) cold[id].result = 'success';
  assert.equal(sourceReadiness(cold, 'windows-2022'), 'cold');
  assert.equal(sourceReadiness(cold, 'windows-2022', { campaign: true, attempt: '1' }), 'cold');
  cold.codec.result = 'skipped'; assert.equal(sourceReadiness(cold, 'ubuntu-24.04'), 'cold');
  assert.throws(() => sourceReadiness(cold, 'windows-2022'), /codec/);
  for (const input of [warm, cold]) for (const result of ['failure', 'cancelled', 'neutral', undefined]) {
    const bad = structuredClone(input); bad.probe.result = result;
    assert.throws(() => sourceReadiness(bad, 'windows-2022'));
  }
  const bad = structuredClone(warm); delete bad.probe.outputs.complete;
  assert.throws(() => sourceReadiness(bad, 'windows-2022'), /complete/);
  bad.probe.outputs.complete = 'true'; bad.ffmpeg.result = 'success';
  assert.throws(() => sourceReadiness(bad, 'windows-2022'), /ffmpeg/);
});

test('full, documentation-only and maintenance scopes explicitly evaluate non-election', () => {
  assert.equal(evaluateResults(results()).scope.native, true);
  assert.equal(evaluateResults(results(false)).scope.native, false);
  assert.equal(evaluateResults(results(false, true), true).results['live-site'], 'success');
  const missing = results(); delete missing.scope.outputs.cli;
  assert.throws(() => evaluateResults(missing), /scope/);
  for (const key of Object.keys(results())) for (const state of ['failure', 'cancelled', 'skipped', undefined, 'neutral']) {
    if (key === 'live-site') continue;
    const bad = results(); bad[key].result = state;
    assert.throws(() => evaluateResults(bad), undefined, `${key}=${state}`);
  }
  const bad = results(false); bad.candidate.result = 'success';
  assert.throws(() => evaluateResults(bad), /candidate/);
});

test('aggregate always evaluates every top-level production result and is declared required', () => {
  const jobs = parse(readFileSync(new URL('../.github/workflows/ci.yml', import.meta.url), 'utf8')).jobs;
  assert.equal(jobs.acceptance.if, 'always()');
  assert.equal(jobs.acceptance.name, 'elected acceptance');
  assert.deepEqual([...jobs.acceptance.needs].sort(), Object.keys(jobs).filter(k => k !== 'acceptance').sort());
  assert.deepEqual([...jobs.acceptance.needs].sort(), Object.keys(results()).sort());
  const bootstrap = JSON.parse(readFileSync(new URL('../.github/bootstrap.json', import.meta.url), 'utf8'));
  assert.ok(bootstrap.required_checks.includes(jobs.acceptance.name));
});

test('phase timing includes cold preparation, queue and terminal cleanup', () => {
  const start = Date.parse('2026-10-20T00:00:00Z'), ready = start + 1342000;
  const got = measureTiming(start, ready, ready + 630000, false);
  assert.equal(got.preparation_seconds, 1342); assert.equal(got.qualification_seconds, 630);
  assert.equal(got.combined_seconds, 1972); assert.equal(got.accepted, true);
  assert.equal(measureTiming(start, ready, ready + 3540000, false).accepted, true);
  assert.equal(measureTiming(start, ready, ready + 3540001, false).accepted, false);
  assert.equal(measureTiming(start, ready, ready + 3600000, true).accepted, true);
  assert.equal(measureTiming(start, ready, ready + 3600001, true).accepted, false);
  for (const value of [NaN, Infinity, start - 1]) assert.throws(() => measureTiming(start, value, ready, true));
});

function metadata() {
  const start = Date.parse('2026-10-20T00:00:00Z');
  return { run: { id: 42, run_attempt: 2, repository: { full_name: 'shruggietech/insonic' }, head_sha: 'a'.repeat(40), path: '.github/workflows/ci.yml', status: 'completed', conclusion: 'success', run_started_at: new Date(start).toISOString(), updated_at: new Date(start + 2000000).toISOString() },
    jobs: [ { id: 1, run_id: 42, run_attempt: 2, name: 'prerequisite readiness', status: 'completed', conclusion: 'success', started_at: new Date(start + 1330000).toISOString(), completed_at: new Date(start + 1342000).toISOString() },
      { id: 2, run_id: 42, run_attempt: 2, name: 'foundation', status: 'completed', conclusion: 'success', started_at: new Date(start + 1343000).toISOString(), completed_at: new Date(start + 1360000).toISOString() },
      { id: 3, run_id: 42, run_attempt: 2, name: 'elected acceptance', status: 'completed', conclusion: 'success', started_at: new Date(start + 1950000).toISOString(), completed_at: new Date(start + 2003000).toISOString() } ] };
}
const identity = { repository: 'shruggietech/insonic', run: '42', attempt: '2', head: 'a'.repeat(40), revision: 'b'.repeat(40) };

function firstAttempt(queue = 120000) {
  const data = metadata();
  data.run.run_attempt = 1;
  data.run.created_at = new Date(Date.parse(data.run.run_started_at) - queue).toISOString();
  for (const job of data.jobs) job.run_attempt = 1;
  return { ...data, identity: { ...identity, attempt: '1' } };
}

test('first-attempt queue belongs to preparation and combined terminal totals', () => {
  for (const queue of [120000, 0]) {
    const data = firstAttempt(queue);
    const got = inspectAttempt(data.identity, data.run, data.jobs, { terminal: true });
    assert.equal(got.preparation_seconds, 1342 + queue / 1000);
    assert.equal(got.combined_seconds, 2003 + queue / 1000);
    assert.equal(got.initial_queue_seconds, queue / 1000);
    assert.equal(got.preparation_started_at, data.run.created_at);
    assert.equal(got.attempt_started_at, data.run.run_started_at);
    assert.equal(got.qualification_seconds, 661);
    assert.equal(got.accepted, true);
  }
});

test('initial queue accounting does not move the provisional qualification deadline', () => {
  const data = firstAttempt();
  data.run.status = data.jobs[2].status = 'in_progress';
  data.run.conclusion = data.jobs[2].conclusion = null;
  data.jobs[2].completed_at = null;
  const boundary = Date.parse(data.jobs[0].completed_at);
  const got = inspectAttempt(data.identity, data.run, data.jobs, { now: boundary + 630000 });
  assert.equal(got.preparation_seconds, 1462);
  assert.equal(got.qualification_seconds, 630);
  assert.equal(got.combined_seconds, 2092);
  assert.equal(got.accepted, true);
  assert.equal(inspectAttempt(data.identity, data.run, data.jobs, { now: boundary + 3540001 }).accepted, false);
});

test('first-attempt creation must be valid and jobs cannot precede attempt start', () => {
  const data = firstAttempt();
  for (const created of [null, 'invalid', new Date(Date.parse(data.run.run_started_at) + 1).toISOString()]) {
    assert.throws(() => inspectAttempt(data.identity, { ...data.run, created_at: created }, data.jobs, { terminal: true }), /timestamp/);
  }
  const early = structuredClone(data.jobs);
  early[0].started_at = data.run.created_at;
  assert.throws(() => inspectAttempt(data.identity, data.run, early, { terminal: true }), /Invalid job timestamps/);
  const retry = metadata();
  retry.run.created_at = data.run.created_at;
  const got = inspectAttempt(identity, retry.run, retry.jobs, { terminal: true });
  assert.equal(got.preparation_seconds, 1342);
  assert.equal(got.combined_seconds, 2003);
  assert.equal(got.preparation_started_at, retry.run.run_started_at);
  assert.equal(got.initial_queue_seconds, null);
});

test('terminal audit uses actual attempt/job completion, not the later invocation clock', () => {
  const data = metadata();
  const got = inspectAttempt(identity, data.run, data.jobs, { terminal: true });
  assert.equal(got.qualification_seconds, 661);
  assert.equal(got.accepted, true);
  for (const field of ['run_attempt', 'id', 'head_sha']) {
    const bad = structuredClone(data.run); bad[field] = 'wrong';
    assert.throws(() => inspectAttempt(identity, bad, data.jobs, { terminal: true }), /identity/);
  }
  const early = structuredClone(data.jobs); early[1].started_at = data.run.run_started_at;
  assert.throws(() => inspectAttempt(identity, data.run, early, { terminal: true }), /before readiness/);
  assert.throws(() => inspectAttempt(identity, data.run, [...data.jobs, data.jobs[0]], { terminal: true }), /ambiguous/);
  const pending = structuredClone(data.jobs); pending[1].status = 'in_progress';
  assert.throws(() => inspectAttempt(identity, data.run, pending, { terminal: true }), /terminal/);
  const wrongAttempt = structuredClone(data.jobs); wrongAttempt[1].run_attempt = 1;
  assert.throws(() => inspectAttempt(identity, data.run, wrongAttempt, { terminal: true }), /identity/);
  const invalidTime = structuredClone(data.jobs); invalidTime[1].completed_at = null;
  assert.throws(() => inspectAttempt(identity, data.run, invalidTime, { terminal: true }), /timestamp/);
});

test('provisional evaluation permits only its own unique running aggregate', () => {
  const data = metadata(); data.run.status = 'in_progress'; data.run.conclusion = null;
  data.jobs[2].status = 'in_progress'; data.jobs[2].completed_at = null; data.jobs[2].conclusion = null;
  const now = Date.parse(data.run.run_started_at) + 1970000;
  assert.equal(inspectAttempt(identity, data.run, data.jobs, { now }).accepted, true);
  data.jobs[1].status = 'in_progress';
  assert.throws(() => inspectAttempt(identity, data.run, data.jobs, { now }), /terminal/);
});

test('a rejected terminal attempt retains phase durations without becoming accepted', () => {
  const data = metadata(); data.run.conclusion = 'failure'; data.jobs[1].conclusion = 'failure';
  const got = inspectAttempt(identity, data.run, data.jobs, { terminal: true, reportFailures: true });
  assert.equal(got.preparation_seconds, 1342);
  assert.equal(got.qualification_seconds, 661);
  assert.equal(got.combined_seconds, 2003);
  assert.ok(got.failures.length >= 2);
});

test('metadata requests are read-only attempt-specific and reject incomplete pagination', async () => {
  const calls = [], values = Array.from({ length: 101 }, (_, id) => ({ id }));
  const got = await attemptMetadata(identity, 'test-only', async url => {
    calls.push(url);
    if (!url.includes('/jobs?')) return metadata().run;
    return { total_count: 101, jobs: url.endsWith('page=1') ? values.slice(0, 100) : values.slice(100) };
  });
  assert.equal(got.jobs.length, 101); assert.equal(calls.length, 3);
  assert.ok(calls.every(url => url.startsWith('https://api.github.com/repos/shruggietech/insonic/actions/runs/42/attempts/2')));
  for (const value of [{ total_count: 101, jobs: values.slice(0, 99) }, { total_count: 401, jobs: [] }, { total_count: 2, jobs: [] }]) {
    await assert.rejects(attemptMetadata(identity, 'test-only', async url => url.includes('/jobs?') ? value : metadata().run), /inventory/);
  }
  await assert.rejects(attemptMetadata({ ...identity, repository: 'other/repo' }, 'test-only'), /identity/);
  await assert.rejects(attemptMetadata(identity, 'test-only', async () => { throw new Error('network unavailable'); }), /network unavailable/);
});

test('overlapped setup waits for slow docs inside the original qualification deadline', async () => {
  const data = metadata(); data.run.status = 'in_progress';
  data.jobs = [data.jobs[0], data.jobs[1], { ...data.jobs[1], id: 4, name: 'docs-build' }];
  const boundary = Date.parse(data.jobs[0].completed_at);
  let clock = boundary + 5000;
  const request = async url => {
    if (!url.includes('/jobs?')) return data.run;
    const jobs = structuredClone(data.jobs);
    if (clock < boundary + 185000) jobs[2].status = 'in_progress';
    return { total_count: jobs.length, jobs };
  };
  const deadline = await waitForQualificationInputs(identity, 'test-only', { request, now: () => clock, pause: async ms => { clock += ms; } });
  assert.equal(deadline, boundary + 3600000);
  assert.ok(clock >= boundary + 185000);
  clock = boundary + 3600000;
  await assert.rejects(waitForQualificationInputs(identity, 'test-only', { request, now: () => clock }), /deadline/);
  clock = boundary + 5000; data.jobs[2].status = 'completed'; data.jobs[2].conclusion = 'failure';
  await assert.rejects(waitForQualificationInputs(identity, 'test-only', { request: async url => url.includes('/jobs?') ? { total_count: data.jobs.length, jobs: data.jobs } : data.run, now: () => clock }), /failed/);
});

test('first attempt uses workflow creation while retaining later attempt-record creation', async () => {
  const data = firstAttempt(0), calls = [];
  const attempt = { ...data.run, created_at: new Date(Date.parse(data.run.run_started_at) + 1000).toISOString() };
  const got = await attemptMetadata(data.identity, 'test-only', async url => {
    calls.push(url);
    if (url.includes('/jobs?')) return { total_count: data.jobs.length, jobs: data.jobs };
    return url.includes('/attempts/') ? attempt : data.run;
  });
  assert.equal(got.run.created_at, data.run.created_at);
  assert.equal(got.run.attempt_record_created_at, attempt.created_at);
  assert.equal(calls.filter(url => !url.includes('/attempts/')).length, 1);
  assert.equal(inspectAttempt(data.identity, got.run, got.jobs, { terminal: true }).initial_queue_seconds, 0);
  assert.equal(inspectAttempt(data.identity, got.run, got.jobs, { terminal: true }).attempt_record_created_at, attempt.created_at);
  assert.equal(inspectAttempt(data.identity, got.run, got.jobs, { terminal: true }).qualification_seconds, 661);
});

test('workflow creation authority rejects mismatched identities and invalid queue ordering', async () => {
  const data = firstAttempt(120000);
  for (const change of [{ id: 99 }, { head_sha: 'c'.repeat(40) }, { repository: { full_name: 'other/repo' } }, { path: 'other.yml' }]) {
    await assert.rejects(attemptMetadata(data.identity, 'test-only', async url => url.includes('/jobs?')
      ? { total_count: data.jobs.length, jobs: data.jobs }
      : url.includes('/attempts/') ? data.run : { ...data.run, ...change }), /identity/);
  }
  const got = await attemptMetadata(data.identity, 'test-only', async url => url.includes('/jobs?')
    ? { total_count: data.jobs.length, jobs: data.jobs }
    : url.includes('/attempts/') ? data.run : { ...data.run, created_at: new Date(Date.parse(data.run.run_started_at) + 1000).toISOString() });
  assert.throws(() => inspectAttempt(data.identity, got.run, got.jobs, { terminal: true }), /queue timestamps/);
});

test('qualification target overruns are advisory through the exact terminal hard boundary', () => {
  const ready = 1000000;
  for (const seconds of [1800, 1800.001, 2400, 3600, 3600.001]) {
    const got = measureTiming(0, ready, ready + seconds * 1000, true);
    assert.equal(got.qualification_target_seconds, 1800);
    assert.equal(got.qualification_limit_seconds, 3600);
    assert.equal(got.target_exceeded, seconds > 1800);
    assert.equal(got.target_overrun_seconds, Math.max(0, seconds - 1800));
    assert.equal(got.accepted, seconds <= 3600);
    assert.equal(got.finalization_reserve_seconds, 0);
  }
  assert.equal(measureTiming(0, ready, ready + 3540000, false).finalization_reserve_seconds, 60);
  assert.equal(measureTiming(0, ready, ready + 3540000, false).accepted, true);
  assert.equal(measureTiming(0, ready, ready + 3540001, false).accepted, false);
});

test('input polling rejects a changed readiness origin rather than extending its deadline', async () => {
  const data = metadata(); data.run.status = 'in_progress';
  data.jobs = [data.jobs[0], { ...data.jobs[1], name: 'docs-build', status: 'in_progress' }];
  let clock = Date.parse(data.jobs[0].completed_at) + 5000;
  const request = async url => url.includes('/jobs?') ? { total_count: data.jobs.length, jobs: data.jobs } : data.run;
  await assert.rejects(waitForQualificationInputs(identity, 'test-only', { request, now: () => clock, pause: async ms => {
    clock += ms; data.jobs[0].completed_at = new Date(Date.parse(data.jobs[0].completed_at) + 1000).toISOString();
  } }), /boundary changed/);
});

test('qualification input identity failure records bounded observed fields without weakening rejection', async () => {
  const data = metadata(); data.run.status = 'in_progress';
  data.jobs = [data.jobs[0], data.jobs[1], { ...data.jobs[1], id: 4, name: 'docs-build' }];
  const now = () => Date.parse(data.jobs[0].completed_at) + 5000;
  for (const changed of [{ status: 'completed' }, { head_sha: 'c'.repeat(40) }, { run_attempt: 3 }]) {
    await assert.rejects(waitForQualificationInputs(identity, 'test-only', {
      request: async url => url.includes('/jobs?') ? { total_count: data.jobs.length, jobs: data.jobs } : { ...data.run, ...changed }, now,
    }), error => {
      assert.match(error.message, /Qualification input attempt identity differs:/);
      const evidence = JSON.parse(error.message.split('differs: ')[1]);
      assert.equal(evidence.expected.head, identity.head);
      for (const [field, value] of Object.entries(changed)) assert.equal(evidence.observed[{ head_sha: 'head', run_attempt: 'attempt', status: 'status' }[field]], value);
      return true;
    });
  }
  const foreign = Array.from({ length: 12 }, (_, id) => ({ id: id + 20, run_id: 42, run_attempt: null, status: 'queued', private_input: 'never retain this' }));
  await assert.rejects(waitForQualificationInputs(identity, 'test-only', {
    request: async url => url.includes('/jobs?') ? { total_count: foreign.length, jobs: foreign } : data.run, now,
  }), error => {
    const evidence = JSON.parse(error.message.split('differs: ')[1]);
    assert.equal(evidence.mismatched_job_count, 12); assert.equal(evidence.mismatched_jobs.length, 8);
    assert.equal(evidence.mismatched_jobs[0].attempt, null);
    assert.ok(!error.message.includes('private_input')); assert.ok(!error.message.includes('never retain this'));
    return true;
  });
});

test('a queued workflow summary cannot invalidate matching successful prerequisite job authority', async () => {
  const data = metadata(); data.run.status = 'queued'; data.run.conclusion = null;
  const ready = data.jobs[0];
  data.jobs = [ready, data.jobs[1], { ...data.jobs[1], id: 4, name: 'docs-build' }];
  const boundary = Date.parse(ready.completed_at), now = () => boundary + 5000;
  const request = async url => url.includes('/jobs?') ? { total_count: data.jobs.length, jobs: data.jobs } : data.run;
  assert.equal(await waitForQualificationInputs(identity, 'test-only', { request, now }), boundary + 3600000);
  data.jobs[0].conclusion = 'failure';
  await assert.rejects(waitForQualificationInputs(identity, 'test-only', { request, now }), /successful prerequisite/);
});

test('queued workflow summary still requires its unique running aggregate and every other terminal job', () => {
  const data = metadata(); data.run.status = 'queued'; data.run.conclusion = null;
  data.jobs[2].status = 'in_progress'; data.jobs[2].completed_at = null; data.jobs[2].conclusion = null;
  const now = Date.parse(data.run.run_started_at) + 1970000;
  assert.equal(inspectAttempt(identity, data.run, data.jobs, { now }).accepted, true);
  assert.throws(() => inspectAttempt(identity, data.run, data.jobs, { terminal: true }), /terminal|running/);
  data.jobs[2].status = 'queued';
  assert.throws(() => inspectAttempt(identity, data.run, data.jobs, { now }), /running attempt/);
  data.jobs[2].status = 'in_progress'; data.jobs[1].status = 'in_progress';
  assert.throws(() => inspectAttempt(identity, data.run, data.jobs, { now }), /not terminal/);
});
