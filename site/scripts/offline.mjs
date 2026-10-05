// SPDX-License-Identifier: Apache-2.0
import fs from 'node:fs/promises';
import path from 'node:path';
import { root, getManifest } from '../lib/docs.mjs';
import { walk } from './postbuild.mjs';

export async function buildOffline() {
  const out = path.join(root, 'site/out');
  const offline = path.join(root, 'site/offline');
  // Both paths are fixed descendants of this repository, never user input.
  if (path.dirname(offline) !== path.join(root, 'site')) throw new Error('Invalid offline output path.');
  await fs.rm(offline, { recursive: true, force: true });
  const files = await walk(out);
  for (const filename of files) {
    const relative = path.relative(out, filename).split(path.sep).join('/');
    const include = (relative.startsWith('docs/') && relative.endsWith('.html')) || relative.startsWith('brand/') || relative.startsWith('schemas/') || relative.startsWith('reference-sources/') || relative === 'theme-init.js' || relative === 'site-runtime.js' || relative === 'docs-runtime.js' || relative === 'docs-runtime.js.LEGAL.txt' || (relative.startsWith('_next/static/') && relative.endsWith('.css'));
    if (!include) continue;
    const destination = path.join(offline, relative);
    await fs.mkdir(path.dirname(destination), { recursive: true });
    await fs.copyFile(filename, destination);
  }
  const version = getManifest().latest;
  await fs.writeFile(path.join(offline, 'index.html'), `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta http-equiv="refresh" content="0;url=docs/index.html"><title>insonic ${version} documentation</title></head><body><main><h1>insonic documentation</h1><a href="docs/index.html">Open versioned documentation</a></main></body></html>\n`, 'utf8');
  await fs.writeFile(path.join(offline, 'offline-manifest.json'), `${JSON.stringify({ version, entrypoint: 'docs/index.html', public_landing_included: false }, null, 2)}\n`, 'utf8');
  return offline;
}
