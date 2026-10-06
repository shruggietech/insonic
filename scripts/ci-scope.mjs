// SPDX-License-Identifier: Apache-2.0
import { appendFileSync, readFileSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const full = () => ({ core: true, cli: true, native: true, adapters: true });
export function classify(files) {
  if (!Array.isArray(files)) return full();
  const result = { core: false, cli: false, native: false, adapters: false };
  for (const file of files) {
    if (typeof file !== 'string' || file.includes('..') || file.startsWith('/')) return full();
    if (/^(docs\/|specs\/)/.test(file) || /^(README|CHANGELOG|CONTRIBUTING|SECURITY|AGENTS|CLAUDE)\.md$/.test(file) || file.startsWith('.specify/')) continue;
    if (/^(package(-lock)?\.json|scripts\/(check|check-schemas|github-bootstrap|sync-brand-kit)\.mjs|tests\/[^/]+\.test\.mjs)$/.test(file)) continue;
    // Runtime/catalog/schema changes can affect the desktop's shared IPC bridge.
    // Unknown/new areas qualify fully instead of silently skipping new dependencies.
    Object.assign(result, full());
  }
  return result;
}

function changedFiles() {
  if (process.env.GITHUB_EVENT_NAME === 'workflow_dispatch') return null;
  try {
    const event = JSON.parse(readFileSync(process.env.GITHUB_EVENT_PATH, 'utf8'));
    const base = event.pull_request?.base?.sha ?? event.before;
    const head = process.env.GITHUB_SHA;
    if (!/^[a-f0-9]{40}$/.test(base ?? '') || /^0+$/.test(base) || !/^[a-f0-9]{40}$/.test(head ?? '')) return null;
    const diff = spawnSync('git', ['diff', '--name-only', '-z', base, head], { encoding: 'utf8', windowsHide: true, maxBuffer: 16 << 20 });
    if (diff.status !== 0) return null;
    return diff.stdout.split('\0').filter(Boolean);
  } catch { return null; }
}
if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const scope = classify(changedFiles());
  if (process.env.GITHUB_OUTPUT) appendFileSync(process.env.GITHUB_OUTPUT, Object.entries(scope).map(([key,value])=>`${key}=${value}\n`).join(''));
  process.stdout.write(`${JSON.stringify(scope)}\n`);
}
