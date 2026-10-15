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
  while (now() < deadline) {
    const value = await request(url, token, AbortSignal.timeout(Math.max(1, Math.min(15000, deadline - now()))));
    if (!Array.isArray(value?.artifacts) || value.total_count !== value.artifacts.length || value.total_count > 100) throw new Error('Incomplete source artifact inventory.');
    const matches = value.artifacts.filter(item => item.name === name);
    if (matches.length > 1) throw new Error('Ambiguous source artifact identity.');
    if (matches.length) {
      const item = matches[0];
      if (item.expired !== false || !Number.isSafeInteger(item.id) || item.id < 1 || String(item.workflow_run?.id) !== run
          || !/^sha256:[a-f0-9]{64}$/.test(item.digest ?? '')) throw new Error('Invalid source artifact authority.');
      return item.id;
    }
    await pause(Math.max(0, Math.min(5000, deadline - now())));
  }
  throw new Error('Same-run source artifact deadline exceeded.');
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  await waitForArtifact({ repository: process.env.GITHUB_REPOSITORY, run: process.env.GITHUB_RUN_ID,
    attempt: process.env.GITHUB_RUN_ATTEMPT, os: process.argv[2], token: process.env.GH_TOKEN });
  console.log('Exact same-run source artifact is available; digest and source checks follow.');
}
