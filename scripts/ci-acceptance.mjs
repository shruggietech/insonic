// SPDX-License-Identifier: Apache-2.0
import fs from 'node:fs/promises';
import committedPolicy from './ci-timing-policy.json' with { type: 'json' };
import { milliseconds, validateAggregatePolicy } from './qualification-timing.mjs';
const policy = validateAggregatePolicy(committedPolicy);
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { setTimeout as sleep } from 'node:timers/promises';

export function sourceReadiness(needs, os, { campaign = false, attempt = '1' } = {}) {
  if (!['windows-2022', 'ubuntu-24.04'].includes(os)) throw new Error('Unknown source platform.');
  if (needs.probe?.result !== 'success') throw new Error('Source probe did not succeed.');
  const complete = needs.probe.outputs?.complete;
  if (!['true', 'false'].includes(complete)) throw new Error('Source probe complete output is missing.');
  const warm = complete === 'true';
  if (campaign && (!/^[1-9]\d*$/.test(attempt) || (attempt === '1' && warm))) {
    throw new Error('The first isolated campaign attempt must demonstrate cold source preparation.');
  }
  for (const id of ['groups', 'formats', 'codec', 'ffmpeg']) {
    const expected = warm || (id === 'codec' && os === 'ubuntu-24.04') ? 'skipped' : 'success';
    if (needs[id]?.result !== expected) throw new Error(`Source ${id} must be ${expected}.`);
  }
  return warm ? 'warm' : 'cold';
}

function scopeOf(needs) {
  if (needs.scope?.result !== 'success') throw new Error('Required scope did not succeed.');
  const scope = {};
  for (const key of ['core', 'cli', 'native', 'adapters']) {
    const value = needs.scope.outputs?.[key];
    if (!['true', 'false'].includes(value)) throw new Error(`Missing or invalid scope ${key}.`);
    scope[key] = value === 'true';
  }
  // Existing runtime workflows elect CLI checks within core, and packages within native.
  if ((scope.cli || scope.native) && !scope.core) throw new Error('Inconsistent scope requires core qualification.');
  return scope;
}

export function evaluateResults(needs, siteSmoke = false) {
  const scope = scopeOf(needs), prepare = scope.core || scope.native || scope.adapters;
  const election = { scope: true, readiness: true, foundation: true, 'docs-build': true, docs: true,
    'media-linux': prepare, 'media-windows': prepare, 'dependencies-windows': scope.native, core: scope.core, 'core-windows': scope.core,
    'qualify-linux': scope.core, 'qualify-windows': scope.core,
    'package-linux': scope.native, 'package-windows': scope.native, adapters: scope.adapters,
    candidate: scope.native, 'live-site': siteSmoke };
  if (Object.keys(needs).some(id => !Object.hasOwn(election, id))) throw new Error('Unknown acceptance prerequisite.');
  const results = {};
  for (const [id, elected] of Object.entries(election)) {
    const expected = elected ? 'success' : 'skipped';
    if (needs[id]?.result !== expected) throw new Error(`Required ${id} must be ${expected}, got ${needs[id]?.result ?? 'missing'}.`);
    results[id] = expected;
  }
  if (needs.readiness.outputs?.ready !== 'true') throw new Error('Missing verified readiness output.');
  return { scope, election, results };
}

export function measureTiming(start, ready, end, terminal) {
  if (![start, ready, end].every(Number.isFinite) || start < 0 || ready < start || end < ready) throw new Error('Invalid phase timestamps.');
  const qualification = (end - ready) / 1000, reserve = terminal ? 0 : policy.finalization_reserve_seconds;
  return { preparation_seconds: (ready - start) / 1000, qualification_seconds: qualification,
    combined_seconds: (end - start) / 1000, qualification_target_seconds: policy.qualification_target_seconds,
    qualification_limit_seconds: policy.qualification_limit_seconds,
    target_exceeded: qualification > policy.qualification_target_seconds,
    target_overrun_seconds: Math.max(0, qualification - policy.qualification_target_seconds),
    finalization_reserve_seconds: reserve, terminal, accepted: qualification + reserve <= policy.qualification_limit_seconds };
}

function timestamp(value) {
  if (typeof value !== 'string' || !/^\d{4}-\d\d-\d\dT/.test(value) || !Number.isFinite(Date.parse(value))) throw new Error('Missing or invalid API timestamp.');
  return Date.parse(value);
}

export function inspectAttempt(identity, run, jobs, { terminal = false, now = Date.now(), reportFailures = false } = {}) {
  if (identity.repository !== 'shruggietech/insonic' || !/^[1-9]\d*$/.test(identity.run ?? '')
      || !/^[1-9]\d*$/.test(identity.attempt ?? '') || !/^[a-f0-9]{40}$/.test(identity.head ?? '')
      || !/^[a-f0-9]{40}$/.test(identity.revision ?? '')
      || String(run.id) !== identity.run || String(run.run_attempt) !== identity.attempt
      || run.repository?.full_name !== identity.repository || run.head_sha !== identity.head
      || run.path !== '.github/workflows/ci.yml') throw new Error('Workflow attempt identity differs.');
  if (!Array.isArray(jobs) || !jobs.length || jobs.length > 400 || new Set(jobs.map(j => j.id)).size !== jobs.length) throw new Error('Incomplete or ambiguous job inventory.');
  const ready = jobs.filter(j => j.name === 'prerequisite readiness');
  const aggregate = jobs.filter(j => j.name === 'elected acceptance');
  if (ready.length !== 1 || aggregate.length !== 1) throw new Error('Missing or ambiguous phase jobs.');
  if (ready[0].status !== 'completed' || ready[0].conclusion !== 'success') throw new Error('Prerequisite readiness did not succeed.');
  // GitHub can retain a queued workflow summary after its bound jobs run.
  // The unique aggregate must itself be running; terminal audit stays strict.
  if (!terminal && (!['queued', 'in_progress'].includes(run.status) || aggregate[0].status !== 'in_progress')) throw new Error('Provisional acceptance must evaluate its own running attempt.');
  const attemptStart = timestamp(run.run_started_at);
  const preparationStart = identity.attempt === '1' ? run.created_at : run.run_started_at;
  const start = timestamp(preparationStart), boundary = timestamp(ready[0].completed_at);
  const initialQueue = identity.attempt === '1' ? (attemptStart - start) / 1000 : null;
  if (initialQueue !== null && initialQueue < 0) throw new Error('Invalid initial queue timestamps.');
  const completed = [], failures = [];
  for (const job of jobs) {
    if (!Number.isSafeInteger(job.id) || job.id < 1 || String(job.run_id) !== identity.run
        || String(job.run_attempt) !== identity.attempt) throw new Error('Job attempt identity differs.');
    if (job.status !== 'completed') {
      if (!terminal && job.id === aggregate[0].id && job.status === 'in_progress') continue;
      throw new Error('Required job is not terminal.');
    }
    if (!['success', 'skipped'].includes(job.conclusion)) {
      if (!reportFailures) throw new Error(`Job ${job.name} did not succeed.`);
      failures.push(`Job ${job.name} did not succeed.`);
    }
    if (job.conclusion === 'skipped') continue;
    const begin = timestamp(job.started_at), end = timestamp(job.completed_at);
    if (begin < attemptStart || end < begin) throw new Error('Invalid job timestamps.');
    if (!['scope', 'prerequisite readiness', 'Windows Go dependency preparation'].includes(job.name) && !/^media-(linux|windows)\s*\//.test(job.name) && begin < boundary) {
      throw new Error(`Qualification job ${job.name} began before readiness.`);
    }
    completed.push(end);
  }
  if (terminal && run.status !== 'completed') throw new Error('Workflow attempt is not terminal.');
  if (terminal && run.conclusion !== 'success') {
    if (!reportFailures) throw new Error('Workflow attempt is not terminal success.');
    failures.push('Workflow attempt is not terminal success.');
  }
  // updated_at is metadata, not a documented completion field. Include it only
  // as a conservative envelope after the attempt is terminal; also report it.
  const workerEnd = Math.max(...completed), metadataUpdated = timestamp(run.updated_at);
  const end = terminal ? Math.max(workerEnd, metadataUpdated) : now;
  if (!terminal && workerEnd > end) throw new Error('Evaluation predates job completion.');
  return { ...measureTiming(start, boundary, end, terminal), failures, attempt_started_at: run.run_started_at,
    attempt_record_created_at: run.attempt_record_created_at ?? null,
    preparation_started_at: preparationStart, readiness_at: ready[0].completed_at, evaluated_through: new Date(end).toISOString(),
    worker_terminal_at: terminal ? new Date(workerEnd).toISOString() : null,
    metadata_updated_at: run.updated_at,
    initial_queue_seconds: initialQueue };
}

async function requestJson(url, token) {
  const response = await fetch(url, { redirect: 'error', signal: AbortSignal.timeout(milliseconds('request_seconds')),
    headers: { Authorization: `Bearer ${token}`, Accept: 'application/vnd.github+json', 'X-GitHub-Api-Version': '2022-11-28' } });
  if (!response.ok) { await response.body?.cancel(); throw new Error(`Acceptance API failed (HTTP ${response.status}).`); }
  let length = 0; const chunks = [];
  for await (const chunk of response.body) {
    length += chunk.length;
    if (length > 2 << 20) { await response.body?.cancel().catch(() => {}); throw new Error('Acceptance API response exceeds bound.'); }
    chunks.push(chunk);
  }
  return JSON.parse(Buffer.concat(chunks).toString('utf8'));
}

class IncompleteJobInventory extends Error {
  constructor(run, jobs, page, expected, stage) {
    super(`Incomplete job inventory (${stage}; page ${page}; expected ${expected}; observed ${jobs.length}).`);
    this.partial = { run, jobs };
  }
}

export async function attemptMetadata(identity, token, request = requestJson) {
  if (identity.repository !== 'shruggietech/insonic' || !/^[1-9]\d*$/.test(identity.run ?? '')
      || !/^[1-9]\d*$/.test(identity.attempt ?? '') || !token) throw new Error('Invalid metadata request identity.');
  const base = `https://api.github.com/repos/${identity.repository}/actions/runs/${identity.run}/attempts/${identity.attempt}`;
  let run = await request(base, token);
  const jobs = [];
  if (identity.attempt === '1') {
    // The attempt record can be created after run_started_at. Initial queue
    // belongs to the workflow event, while all jobs/status remain attempt-specific.
    const workflow = await request(base.replace('/attempts/1', ''), token);
    for (const record of [run, workflow]) {
      if (String(record.id) !== identity.run || record.head_sha !== identity.head
          || record.repository?.full_name !== identity.repository || record.path !== '.github/workflows/ci.yml') {
        throw new Error('Workflow creation identity differs.');
      }
    }
    run = { ...run, attempt_record_created_at: run.created_at, created_at: workflow.created_at };
  }
  let total;
  for (let page = 1; page <= 4; page++) {
    const value = await request(`${base}/jobs?per_page=100&page=${page}`, token);
    if (!Number.isSafeInteger(value.total_count) || value.total_count < 1 || value.total_count > 400
        || !Array.isArray(value.jobs) || value.jobs.length > 100) throw new Error('Incomplete job inventory.');
    if (total !== undefined && value.total_count !== total) {
      throw new IncompleteJobInventory(run, jobs, page, value.total_count, 'count changed');
    }
    total = value.total_count;
    jobs.push(...value.jobs);
    if (jobs.some(job => !job || !Number.isSafeInteger(job.id))) throw new Error('Invalid job inventory identifiers.');
    if (new Set(jobs.map(job => job.id)).size !== jobs.length) {
      throw new IncompleteJobInventory(run, jobs, page, total, 'duplicate identifiers');
    }
    if (jobs.length === total) return { run, jobs };
    if (jobs.length > total || value.jobs.length !== 100) {
      throw new IncompleteJobInventory(run, jobs, page, total, 'short or excess page');
    }
  }
  throw new Error('Job inventory exceeds bound.');
}

export async function waitForQualificationInputs(identity, token, { request = requestJson, now = Date.now, pause = sleep } = {}) {
  let deadline, inventoryRetries = 0;
  for (;;) {
    if (deadline !== undefined && now() >= deadline) throw new Error('Original qualification deadline exceeded.');
    let metadata, incomplete;
    try { metadata = await attemptMetadata(identity, token, request); }
    catch (error) {
      if (!(error instanceof IncompleteJobInventory)) throw error;
      metadata = error.partial;
      incomplete = error;
    }
    const { run, jobs } = metadata;
    if (run.head_sha !== identity.head || String(run.id) !== identity.run || String(run.run_attempt) !== identity.attempt
        || run.repository?.full_name !== identity.repository || run.path !== '.github/workflows/ci.yml'
        || !['queued', 'in_progress'].includes(run.status) || jobs.some(j => String(j.run_id) !== identity.run || String(j.run_attempt) !== identity.attempt)) {
      const mismatchedJobs = jobs.filter(j => String(j.run_id) !== identity.run || String(j.run_attempt) !== identity.attempt);
      throw new Error('Qualification input attempt identity differs: ' + JSON.stringify({
        expected: { repository: identity.repository, run: identity.run, attempt: identity.attempt, head: identity.head },
        observed: { repository: run.repository?.full_name ?? null, run: run.id ?? null, attempt: run.run_attempt ?? null,
          head: run.head_sha ?? null, path: run.path ?? null, status: run.status ?? null },
        mismatched_jobs: mismatchedJobs.slice(0, 8).map(j => ({ id: j.id ?? null, run: j.run_id ?? null,
          attempt: j.run_attempt ?? null, status: j.status ?? null })), mismatched_job_count: mismatchedJobs.length,
      }));
    }
    const ready = jobs.filter(j => j.name === 'prerequisite readiness');
    if (ready.length !== 1 || ready[0].status !== 'completed' || ready[0].conclusion !== 'success') throw new Error('Setup must follow successful prerequisite readiness.');
    const bound = timestamp(ready[0].completed_at) + policy.qualification_limit_seconds * 1000;
    if (deadline !== undefined && bound !== deadline) throw new Error('Qualification readiness boundary changed.');
    deadline = bound;
    if (now() >= deadline) throw new Error('Original qualification deadline exceeded.');
    // Partial observations may anchor the earlier deadline and reject known
    // failures, but can never authorize qualification. Resnapshot from page 1.
    let complete = !incomplete;
    for (const name of ['foundation', 'docs-build']) {
      const matches = jobs.filter(j => j.name === name);
      if (matches.length > 1) throw new Error('Ambiguous qualification input.');
      const job = matches[0];
      if (job?.status === 'completed' && job.conclusion !== 'success') throw new Error('Required qualification input failed.');
      complete &&= job?.status === 'completed' && job.conclusion === 'success';
    }
    if (incomplete && inventoryRetries++ >= 3) throw incomplete;
    if (complete) return deadline;
    await pause(Math.max(0, Math.min(10000, deadline - now())));
  }
}

async function main() {
  const mode = process.argv[2];
  const needs = () => JSON.parse(process.env.NEEDS_JSON ?? '{}');
  if (mode === 'source') {
    const readiness = sourceReadiness(needs(), process.env.SOURCE_RUNNER,
      { campaign: process.env.MEDIA_COLD_CACHE === 'true', attempt: process.env.GITHUB_RUN_ATTEMPT });
    await fs.appendFile(process.env.GITHUB_OUTPUT, `ready=true\npath=${readiness}\n`);
    console.log(`Verified ${readiness} source producers succeeded.`);
    return;
  }
  if (mode === 'ready') {
    const input = needs(), scope = scopeOf(input), prepare = scope.core || scope.native || scope.adapters;
    for (const id of ['media-linux', 'media-windows']) {
      const expected = prepare ? 'success' : 'skipped';
      if (input[id]?.result !== expected || (prepare && input[id].outputs?.ready !== 'true')) throw new Error(`${id} is not verified ready or explicitly unelected.`);
    }
    const dependencies = input['dependencies-windows']?.result;
    if (dependencies !== (scope.native ? 'success' : 'skipped')) throw new Error('Windows Go dependencies did not prepare successfully or explicitly remain unelected.');
    await fs.appendFile(process.env.GITHUB_OUTPUT, 'ready=true\n');
    return;
  }
  if (mode === 'wait-inputs') {
    const deadline = await waitForQualificationInputs({ repository: process.env.GITHUB_REPOSITORY,
      run: process.env.GITHUB_RUN_ID, attempt: process.env.GITHUB_RUN_ATTEMPT,
      head: process.env.EXPECTED_HEAD, revision: process.env.GITHUB_SHA }, process.env.GH_TOKEN);
    await fs.appendFile(process.env.GITHUB_ENV, `INSONIC_QUALIFICATION_DEADLINE=${deadline}\n`);
    return;
  }
  const terminal = mode === '--audit';
  // Audit a digest-verified receipt downloaded from the exact run. Its context
  // is retained, while phase boundaries and completion are re-read from GitHub.
  const previous = terminal ? JSON.parse(await fs.readFile(process.argv[3], 'utf8')) : null;
  const identity = previous?.identity ?? { repository: process.env.GITHUB_REPOSITORY,
    run: process.env.GITHUB_RUN_ID, attempt: process.env.GITHUB_RUN_ATTEMPT,
    head: process.env.EXPECTED_HEAD, revision: process.env.GITHUB_SHA };
  const failures = [], receipt = { format: 'insonic-ci-acceptance-v1', identity, terminal };
  try { receipt.evaluation = evaluateResults(previous?.needs ?? needs(), previous?.site_smoke ?? process.env.SITE_SMOKE === 'true'); }
  catch (error) { failures.push(error.message); }
  const data = await attemptMetadata(identity, process.env.GH_TOKEN);
  try {
    receipt.timing = inspectAttempt(identity, data.run, data.jobs, { terminal, reportFailures: true });
    failures.push(...receipt.timing.failures);
  }
  catch (error) { failures.push(error.message); }
  if (receipt.timing && !receipt.timing.accepted) failures.push(`Qualification exceeds its ${policy.qualification_limit_seconds}-second hard limit/finalization reserve.`);
  receipt.advisories = receipt.timing?.target_exceeded
    ? [`Qualification exceeded its ${policy.qualification_target_seconds}-second target by ${receipt.timing.target_overrun_seconds} seconds.`] : [];
  receipt.needs = previous?.needs ?? needs();
  receipt.site_smoke = previous?.site_smoke ?? process.env.SITE_SMOKE === 'true';
  receipt.failures = failures;
  receipt.accepted = failures.length === 0;
  await fs.mkdir('build/ci-acceptance', { recursive: true });
  await fs.writeFile(`build/ci-acceptance/${terminal ? 'terminal' : 'gate'}.json`, JSON.stringify(receipt, null, 2) + '\n');
  if (process.env.GITHUB_STEP_SUMMARY) await fs.appendFile(process.env.GITHUB_STEP_SUMMARY,
    `## Elected acceptance (${terminal ? 'terminal' : 'provisional'})\n\n\`\`\`json\n${JSON.stringify(receipt, null, 2)}\n\`\`\`\n`);
  console.log(JSON.stringify(receipt));
  if (!receipt.accepted) process.exitCode = 1;
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try { await main(); } catch (error) { console.error(error.message); process.exitCode = 1; }
}
