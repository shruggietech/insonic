// SPDX-License-Identifier: Apache-2.0
import { spawn } from 'node:child_process';
import path from 'node:path';
import { root } from '../lib/docs.mjs';
import { prepare } from './prepare.mjs';
import { postbuild } from './postbuild.mjs';
import { checkExport } from './check.mjs';
import { buildOffline } from './offline.mjs';

const command = process.argv[2];
if (!['build', 'dev'].includes(command)) throw new Error('Usage: node scripts/run.mjs build|dev');
await prepare();
await new Promise((resolve, reject) => {
  const child = spawn(process.execPath, [path.join(root, 'site/node_modules/next/dist/bin/next'), command], {
    cwd: path.join(root, 'site'),
    windowsHide: true,
    shell: false,
    stdio: ['ignore', 'inherit', 'inherit'],
    env: { ...process.env, NEXT_TELEMETRY_DISABLED: '1', CI: command === 'build' ? '1' : process.env.CI },
  });
  child.once('error', reject);
  child.once('exit', (code) => code === 0 ? resolve() : reject(new Error(`Next.js ${command} exited with ${code}.`)));
});
if (command === 'build') {
  await postbuild();
  await checkExport();
  const offline = await buildOffline();
  await checkExport({ out: offline, offline: true });
}
