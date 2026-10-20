// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { parse } from 'yaml';
import { sourceReadiness, evaluateResults, measureTiming, inspectAttempt, attemptMetadata, waitForQualificationInputs } from '../scripts/ci-acceptance.mjs';

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
  assert.equal(measureTiming(start, ready, ready + 645000, false).accepted, true);
  assert.equal(measureTiming(start, ready, ready + 645001, false).accepted, false);
  assert.equal(measureTiming(start, ready, ready + 660000, true).accepted, true);
  assert.equal(measureTiming(start, ready, ready + 660001, true).accepted, false);
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

test('terminal audit uses actual attempt/job completion, not the later invocation clock', () => {
  const data = metadata();
  const got = inspectAttempt(identity, data.run, data.jobs, { terminal: true });
  assert.equal(got.qualification_seconds, 661);
  assert.equal(got.accepted, false);
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
  assert.equal(deadline, boundary + 660000);
  assert.ok(clock >= boundary + 185000);
  clock = boundary + 660000;
  await assert.rejects(waitForQualificationInputs(identity, 'test-only', { request, now: () => clock }), /deadline/);
  clock = boundary + 5000; data.jobs[2].status = 'completed'; data.jobs[2].conclusion = 'failure';
  await assert.rejects(waitForQualificationInputs(identity, 'test-only', { request: async url => url.includes('/jobs?') ? { total_count: data.jobs.length, jobs: data.jobs } : data.run, now: () => clock }), /failed/);
});
