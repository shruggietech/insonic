// SPDX-License-Identifier: Apache-2.0
import { setTimeout as sleep } from 'node:timers/promises';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import fs from 'node:fs/promises';
import { milliseconds, effectiveDeadline } from './qualification-timing.mjs';

function requireTime(deadline, now, signal) {
  signal?.throwIfAborted();
  if (now() >= deadline) throw new Error('Original qualification or artifact deadline exceeded.');
}

function requestSignal(deadline, now, signal) {
  const timer = AbortSignal.timeout(Math.max(1, Math.min(milliseconds('request_seconds'), deadline - now())));
  return signal ? AbortSignal.any([signal, timer]) : timer;
}

async function inventory(url, token, signal) {
  const response = await fetch(url, { signal, redirect: 'error', headers: {
    Authorization: `Bearer ${token}`, Accept: 'application/vnd.github+json', 'X-GitHub-Api-Version': '2022-11-28',
  } });
  if (!response.ok) { await response.body?.cancel(); throw new Error(`Source artifact API failed (HTTP ${response.status}).`); }
  let size = 0; const chunks = [];
  for await (const chunk of response.body) {
    size += chunk.length;
    if (size > 1 << 20) { throw new Error('Source artifact inventory exceeds its bound.'); }
    chunks.push(chunk);
  }
  return JSON.parse(Buffer.concat(chunks).toString('utf8'));
}

export async function waitForArtifact({ repository, run, attempt, os, token, deadlineMs = milliseconds('artifact_source_seconds'),
  phaseDeadlineMs, signal, now = Date.now, sleep: pause = sleep, request = inventory }) {
  if (repository !== 'shruggietech/insonic' || !/^[1-9][0-9]*$/.test(run ?? '') || !/^[1-9][0-9]*$/.test(attempt ?? '')
      || !['windows-2022', 'ubuntu-24.04'].includes(os) || !token || !Number.isSafeInteger(deadlineMs)
      || deadlineMs < 1 || deadlineMs > milliseconds('artifact_source_seconds')) throw new Error('Invalid source artifact identity.');
  const deadline = effectiveDeadline(now(), deadlineMs, phaseDeadlineMs), name = `media-source-${os}-${attempt}`;
  const url = `https://api.github.com/repos/${repository}/actions/runs/${run}/artifacts?per_page=100`;
  const jobsUrl = `https://api.github.com/repos/${repository}/actions/runs/${run}/attempts/${attempt}/jobs?per_page=100`;
  while (now() < deadline) {
    requireTime(deadline, now, signal);
    const boundedSignal = requestSignal(deadline, now, signal);
    const [value, gates] = await Promise.all([request(url, token, boundedSignal), request(jobsUrl, token, boundedSignal)]);
    requireTime(deadline, now, signal);
    if (!Array.isArray(gates?.jobs) || gates.total_count !== gates.jobs.length || gates.total_count > 100) throw new Error('Incomplete qualification gate inventory.');
    let ready = true;
    for (const name of ['foundation', 'docs-build']) {
      const matches = gates.jobs.filter(item => item.name === name);
      if (matches.length !== 1) throw new Error('Missing or ambiguous qualification gate.');
      const gate = matches[0];
      if (gate.status === 'completed' && gate.conclusion !== 'success') throw new Error('Required qualification gate failed.');
      ready &&= gate.status === 'completed' && gate.conclusion === 'success';
    }
    if (!Array.isArray(value?.artifacts) || value.total_count !== value.artifacts.length || value.total_count > 100) throw new Error('Incomplete source artifact inventory.');
    const matches = value.artifacts.filter(item => item.name === name);
    if (matches.length > 1) throw new Error('Ambiguous source artifact identity.');
    if (matches.length) {
      const item = matches[0];
      if (item.expired !== false || !Number.isSafeInteger(item.id) || item.id < 1 || String(item.workflow_run?.id) !== run
          || !/^sha256:[a-f0-9]{64}$/.test(item.digest ?? '')) throw new Error('Invalid source artifact authority.');
      if (ready) return item.id;
    }
    await pause(Math.max(0, Math.min(5000, deadline - now())), undefined, { signal });
  }
  throw new Error('Same-run source artifact deadline exceeded.');
}

export async function waitForDesktopProofs(options) {
  return waitForWindowsProducers({ ...options, role: 'proof' });
}

export async function waitForDesktopInput(options) {
  const ready = await waitForWindowsProducers({ ...options, role: 'input' });
  return ready.assembly;
}

export async function waitForWindowsPackages(options) {
  return waitForWindowsProducers({ ...options, role: 'packages' });
}

function windowsProducers(role, revision, attempt) {
  if (role === 'input') return [{ scope: 'assembly', jobName: 'Assemble Windows desktop delivery',
    name: `windows-desktop-input-${revision}-${attempt}` }];
  if (role === 'packages') return [
    { scope: 'cli', jobName: 'package (windows-2022, cli)', name: `candidate-package-windows-2022-cli-${revision}-${attempt}` },
    { scope: 'desktop', jobName: 'Join exact-byte Windows desktop qualification', name: `candidate-package-windows-2022-desktop-${revision}-${attempt}` },
  ];
  return ['archive', 'interactive', 'maintenance', 'setup'].map(scope => ({ scope,
    jobName: `Qualify Windows desktop (${scope})`, name: `windows-desktop-proof-${scope}-${revision}-${attempt}` }));
}

async function waitForWindowsProducers({ role, repository, run, attempt, revision, token, deadlineMs = milliseconds('artifact_producer_seconds'),
  phaseDeadlineMs, signal, now = Date.now, sleep: pause = sleep, request = inventory, observeFailure = async () => {} }) {
  if (repository !== 'shruggietech/insonic' || !/^[1-9][0-9]*$/.test(run ?? '') || !/^[1-9][0-9]*$/.test(attempt ?? '')
      || !['input', 'proof', 'packages'].includes(role) || !/^[a-f0-9]{40}$/.test(revision ?? '') || !token
      || !Number.isSafeInteger(deadlineMs) || deadlineMs < 1 || deadlineMs > milliseconds('artifact_producer_seconds')) throw new Error(`Invalid Windows ${role} identity.`);
  const producers = windowsProducers(role, revision, attempt), deadline = effectiveDeadline(now(), deadlineMs, phaseDeadlineMs);
  const url = `https://api.github.com/repos/${repository}/actions/runs/${run}/artifacts?per_page=100`;
  const jobsUrl = `https://api.github.com/repos/${repository}/actions/runs/${run}/attempts/${attempt}/jobs?per_page=100`;
  let lastIncomplete = '';
  while (now() < deadline) {
    requireTime(deadline, now, signal);
    const boundedSignal = requestSignal(deadline, now, signal);
    const [value, gates] = await Promise.all([request(url, token, boundedSignal), request(jobsUrl, token, boundedSignal)]);
    requireTime(deadline, now, signal);
    for (const [items, count] of [[value?.artifacts, value?.total_count], [gates?.jobs, gates?.total_count]]) {
      if (!Array.isArray(items) || items.length > 100 || !Number.isSafeInteger(count) || count < 0 || count > 100) {
        throw new Error(`Incomplete Windows ${role} inventory.`);
      }
    }
    // Scheduling and artifact uploads can race an API page's reported count.
    // Inspect known failures immediately, but never consume an incomplete snapshot.
    const complete = value.total_count === value.artifacts.length && gates.total_count === gates.jobs.length;
    lastIncomplete = complete ? '' : `; incomplete inventory (artifacts ${value.artifacts.length}/${value.total_count}, jobs ${gates.jobs.length}/${gates.total_count})`;
    const ready = {};
    for (const { scope, jobName, name } of producers) {
      const jobs = gates.jobs.filter(item => item.name === jobName || item.name?.endsWith('/ ' + jobName));
      if (jobs.length > 1) throw new Error(`Ambiguous Windows ${role} producer.`);
      const job = jobs[0];
      if (job?.status === 'completed' && job.conclusion !== 'success') {
        try { await observeFailure({ kind: `windows-${role}-producer-failure`, repository, run, attempt, revision,
          scope, job_id: Number.isSafeInteger(job.id) ? job.id : null, status: 'completed',
          conclusion: ['failure','cancelled','skipped','timed_out','action_required','stale','neutral'].includes(job.conclusion) ? job.conclusion : 'unknown' }); } catch { /* Diagnostics cannot replace the failed producer. */ }
        throw new Error(`Required Windows ${role} failed: ${scope}`);
      }
      const matches = value.artifacts.filter(item => item.name === name);
      if (matches.length > 1) throw new Error(`Ambiguous Windows ${role} artifact.`);
      const item = matches[0];
      if (item && (item.expired !== false || !Number.isSafeInteger(item.id) || item.id < 1 || String(item.workflow_run?.id) !== run
          || !/^sha256:[a-f0-9]{64}$/.test(item.digest ?? ''))) throw new Error(`Invalid Windows ${role} artifact authority.`);
      if (item && job?.status === 'completed' && job.conclusion === 'success') ready[scope] = item.id;
    }
    if (complete && Object.keys(ready).length === producers.length) return ready;
    await pause(Math.max(0, Math.min(5000, deadline - now())), undefined, { signal });
  }
  throw new Error(`Same-run Windows ${role} deadline exceeded${lastIncomplete}.`);
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const identity = { repository: process.env.GITHUB_REPOSITORY, run: process.env.GITHUB_RUN_ID,
    attempt: process.env.GITHUB_RUN_ATTEMPT, token: process.env.GH_TOKEN,
    phaseDeadlineMs: process.env.INSONIC_QUALIFICATION_DEADLINE === undefined ? undefined
      : Number(process.env.INSONIC_QUALIFICATION_DEADLINE) };
  if (process.argv[2] === 'windows-packages') {
    await waitForWindowsPackages({ ...identity, revision: process.argv[3], observeFailure: async observation => {
      const directory = path.resolve('build/native');
      await fs.mkdir(directory, { recursive: true });
      await fs.writeFile(path.join(directory, 'windows-package-failure.json'), JSON.stringify(observation) + '\n', { encoding: 'utf8', flag: 'wx' });
    } });
    console.log('Both exact-run Windows package producers passed; independent candidate verification follows.');
  } else if (process.argv[2] === 'windows-desktop-input') {
    await waitForDesktopInput({ ...identity, revision: process.argv[3], observeFailure: async observation => {
      const directory = path.resolve('build/native');
      await fs.mkdir(directory, { recursive: true });
      await fs.writeFile(path.join(directory, 'windows-input-failure.json'), JSON.stringify(observation) + '\n', { encoding: 'utf8', flag: 'wx' });
    } });
    console.log('Exact-run Windows assembly passed; digest and independent package checks follow.');
  } else if (process.argv[2] === 'windows-desktop-proofs') {
    await waitForDesktopProofs({ ...identity, revision: process.argv[3], observeFailure: async observation => {
      const directory = path.resolve('build/native');
      await fs.mkdir(directory, { recursive: true });
      await fs.writeFile(path.join(directory, 'windows-proof-failure.json'), JSON.stringify(observation) + '\n', { encoding: 'utf8', flag: 'wx' });
    } });
    console.log('All four exact-run Windows proof producers passed; independent artifact verification follows.');
  } else {
    await waitForArtifact({ ...identity, os: process.argv[2] });
    console.log('Exact same-run source artifact is available; digest and source checks follow.');
  }
}
