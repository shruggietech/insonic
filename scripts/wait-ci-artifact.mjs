// SPDX-License-Identifier: Apache-2.0
import { setTimeout as sleep } from 'node:timers/promises';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

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

export async function waitForArtifact({ repository, run, attempt, os, token, deadlineMs = 180000,
  now = Date.now, sleep: pause = sleep, request = inventory }) {
  if (repository !== 'shruggietech/insonic' || !/^[1-9][0-9]*$/.test(run ?? '') || !/^[1-9][0-9]*$/.test(attempt ?? '')
      || !['windows-2022', 'ubuntu-24.04'].includes(os) || !token || deadlineMs < 1 || deadlineMs > 180000) throw new Error('Invalid source artifact identity.');
  const deadline = now() + deadlineMs, name = `media-source-${os}-${attempt}`;
  const url = `https://api.github.com/repos/${repository}/actions/runs/${run}/artifacts?per_page=100`;
  const jobsUrl = `https://api.github.com/repos/${repository}/actions/runs/${run}/attempts/${attempt}/jobs?per_page=100`;
  while (now() < deadline) {
    const signal = AbortSignal.timeout(Math.max(1, Math.min(15000, deadline - now())));
    const [value, gates] = await Promise.all([request(url, token, signal), request(jobsUrl, token, signal)]);
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
    await pause(Math.max(0, Math.min(5000, deadline - now())));
  }
  throw new Error('Same-run source artifact deadline exceeded.');
}

export async function waitForDesktopProofs({ repository, run, attempt, revision, token, deadlineMs = 400000,
  now = Date.now, sleep: pause = sleep, request = inventory }) {
  if (repository !== 'shruggietech/insonic' || !/^[1-9][0-9]*$/.test(run ?? '') || !/^[1-9][0-9]*$/.test(attempt ?? '')
      || !/^[a-f0-9]{40}$/.test(revision ?? '') || !token || deadlineMs < 1 || deadlineMs > 400000) throw new Error('Invalid Windows proof identity.');
  const scopes = ['archive', 'interactive', 'maintenance', 'setup'], deadline = now() + deadlineMs;
  const url = `https://api.github.com/repos/${repository}/actions/runs/${run}/artifacts?per_page=100`;
  const jobsUrl = `https://api.github.com/repos/${repository}/actions/runs/${run}/attempts/${attempt}/jobs?per_page=100`;
  while (now() < deadline) {
    const signal = AbortSignal.timeout(Math.max(1, Math.min(15000, deadline - now())));
    const [value, gates] = await Promise.all([request(url, token, signal), request(jobsUrl, token, signal)]);
    if (!Array.isArray(value?.artifacts) || value.total_count !== value.artifacts.length || value.total_count > 100
        || !Array.isArray(gates?.jobs) || gates.total_count !== gates.jobs.length || gates.total_count > 100) throw new Error('Incomplete Windows proof inventory.');
    const ready = {};
    for (const scope of scopes) {
      const jobName = `Qualify Windows desktop (${scope})`;
      const jobs = gates.jobs.filter(item => item.name === jobName || item.name?.endsWith('/ ' + jobName));
      if (jobs.length > 1) throw new Error('Ambiguous Windows proof producer.');
      const job = jobs[0];
      if (job?.status === 'completed' && job.conclusion !== 'success') throw new Error('Required Windows proof failed: ' + scope);
      const name = `windows-desktop-proof-${scope}-${revision}-${attempt}`;
      const matches = value.artifacts.filter(item => item.name === name);
      if (matches.length > 1) throw new Error('Ambiguous Windows proof artifact.');
      const item = matches[0];
      if (item && (item.expired !== false || !Number.isSafeInteger(item.id) || item.id < 1 || String(item.workflow_run?.id) !== run
          || !/^sha256:[a-f0-9]{64}$/.test(item.digest ?? ''))) throw new Error('Invalid Windows proof artifact authority.');
      if (item && job?.status === 'completed' && job.conclusion === 'success') ready[scope] = item.id;
    }
    if (Object.keys(ready).length === scopes.length) return ready;
    await pause(Math.max(0, Math.min(5000, deadline - now())));
  }
  throw new Error('Same-run Windows proof deadline exceeded.');
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const identity = { repository: process.env.GITHUB_REPOSITORY, run: process.env.GITHUB_RUN_ID,
    attempt: process.env.GITHUB_RUN_ATTEMPT, token: process.env.GH_TOKEN };
  if (process.argv[2] === 'windows-desktop-proofs') {
    await waitForDesktopProofs({ ...identity, revision: process.argv[3] });
    console.log('All four exact-run Windows proof producers passed; independent artifact verification follows.');
  } else {
    await waitForArtifact({ ...identity, os: process.argv[2] });
    console.log('Exact same-run source artifact is available; digest and source checks follow.');
  }
}
