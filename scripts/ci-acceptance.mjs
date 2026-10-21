// SPDX-License-Identifier: Apache-2.0
import fs from 'node:fs/promises';
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
  const qualification = (end - ready) / 1000, reserve = terminal ? 0 : 15;
  return { preparation_seconds: (ready - start) / 1000, qualification_seconds: qualification,
    combined_seconds: (end - start) / 1000, qualification_limit_seconds: 660,
    finalization_reserve_seconds: reserve, terminal, accepted: qualification + reserve <= 660 };
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
  if (!terminal && (run.status !== 'in_progress' || aggregate[0].status !== 'in_progress')) throw new Error('Provisional acceptance must evaluate its own running attempt.');
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
    preparation_started_at: preparationStart, readiness_at: ready[0].completed_at, evaluated_through: new Date(end).toISOString(),
    worker_terminal_at: terminal ? new Date(workerEnd).toISOString() : null,
    metadata_updated_at: run.updated_at,
    initial_queue_seconds: initialQueue };
}

async function requestJson(url, token) {
  const response = await fetch(url, { redirect: 'error', signal: AbortSignal.timeout(10000),
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

export async function attemptMetadata(identity, token, request = requestJson) {
  if (identity.repository !== 'shruggietech/insonic' || !/^[1-9]\d*$/.test(identity.run ?? '')
      || !/^[1-9]\d*$/.test(identity.attempt ?? '') || !token) throw new Error('Invalid metadata request identity.');
  const base = `https://api.github.com/repos/${identity.repository}/actions/runs/${identity.run}/attempts/${identity.attempt}`;
  const run = await request(base, token), jobs = [];
  let total;
  for (let page = 1; page <= 4; page++) {
    const value = await request(`${base}/jobs?per_page=100&page=${page}`, token);
    if (!Number.isSafeInteger(value.total_count) || value.total_count < 1 || value.total_count > 400
        || !Array.isArray(value.jobs) || value.jobs.length > 100
        || (total !== undefined && value.total_count !== total)) throw new Error('Incomplete job inventory.');
    total = value.total_count;
    jobs.push(...value.jobs);
    if (jobs.length === total) return { run, jobs };
    if (jobs.length > total || value.jobs.length !== 100) throw new Error('Incomplete job inventory.');
  }
  throw new Error('Job inventory exceeds bound.');
}

export async function waitForQualificationInputs(identity, token, { request = requestJson, now = Date.now, pause = sleep } = {}) {
  let deadline;
  for (;;) {
    const { run, jobs } = await attemptMetadata(identity, token, request);
    if (run.head_sha !== identity.head || String(run.id) !== identity.run || String(run.run_attempt) !== identity.attempt
        || run.repository?.full_name !== identity.repository || run.path !== '.github/workflows/ci.yml'
        || run.status !== 'in_progress' || jobs.some(j => String(j.run_id) !== identity.run || String(j.run_attempt) !== identity.attempt)) {
      throw new Error('Qualification input attempt identity differs.');
    }
    const ready = jobs.filter(j => j.name === 'prerequisite readiness');
    if (ready.length !== 1 || ready[0].status !== 'completed' || ready[0].conclusion !== 'success') throw new Error('Setup must follow successful prerequisite readiness.');
    const bound = timestamp(ready[0].completed_at) + 660000;
    if (deadline !== undefined && bound !== deadline) throw new Error('Qualification readiness boundary changed.');
    deadline = bound;
    if (now() >= deadline) throw new Error('Original qualification deadline exceeded.');
    let complete = true;
    for (const name of ['foundation', 'docs-build']) {
      const matches = jobs.filter(j => j.name === name);
      if (matches.length > 1) throw new Error('Ambiguous qualification input.');
      const job = matches[0];
      if (job?.status === 'completed' && job.conclusion !== 'success') throw new Error('Required qualification input failed.');
      complete &&= job?.status === 'completed' && job.conclusion === 'success';
    }
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
  if (receipt.timing && !receipt.timing.accepted) failures.push('Qualification exceeds its 660-second budget/finalization reserve.');
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
