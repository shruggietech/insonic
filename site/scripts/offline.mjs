// SPDX-License-Identifier: Apache-2.0
import fs from 'node:fs/promises';
import path from 'node:path';
import { root, getManifest } from '../lib/docs.mjs';
import { walk, relativeTarget } from './postbuild.mjs';

export async function copyOfflineFiles(out, offline) {
  const files = await walk(out);
  for (const filename of files) {
    const relative = path.relative(out, filename).split(path.sep).join('/');
    const include = (relative.startsWith('docs/') && relative.endsWith('.html')) || relative.startsWith('brand/') || relative.startsWith('schemas/') || relative.startsWith('reference-sources/') || /^screenshots\/v[0-9]+\.[0-9]+\.[0-9]+\/[a-z0-9-]+(?:\.png|-capture\.json)$/.test(relative) || relative === 'theme-init.js' || relative === 'site-runtime.js' || relative === 'docs-runtime.js' || relative === 'docs-runtime.js.LEGAL.txt' || (relative.startsWith('_next/static/') && relative.endsWith('.css'));
    if (!include) continue;
    const destination = path.join(offline, relative);
    await fs.mkdir(path.dirname(destination), { recursive: true });
    await fs.copyFile(filename, destination);
  }
}

export async function buildOffline() {
  const out = path.join(root, 'site/out');
  const offline = path.join(root, 'site/offline');
  // Both paths are fixed descendants of this repository, never user input.
  if (path.dirname(offline) !== path.join(root, 'site')) throw new Error('Invalid offline output path.');
  await fs.rm(offline, { recursive: true, force: true });
  await copyOfflineFiles(out, offline);
  const version = getManifest().latest;
  // The entry file is the full current article, not a timed redirect. Rebase
  // the already-portable docs/index.html links one directory upwards.
  const sourceIndex = path.join(out, 'docs', 'index.html');
  const entryIndex = path.join(offline, 'index.html');
  let index = await fs.readFile(sourceIndex, 'utf8');
  index = index.replace(/\b(href|src|srcSet|srcset)="([^" ]*)"/g, (attribute, name, value) => {
    if (!value || /^(?:[a-z]+:|\/\/|#)/i.test(value)) return attribute;
    const [, pathname, query = '', fragment = ''] = value.match(/^([^?#]*)(\?[^#]*)?(#.*)?$/);
    const target = path.resolve(path.dirname(sourceIndex), pathname);
    const exportedPath = path.relative(out, target).split(path.sep).join('/');
    if (exportedPath.startsWith('../')) throw new Error('Offline entry link leaves the export.');
    return name + '="' + relativeTarget(entryIndex, '/' + exportedPath + query + fragment, offline) + '"';
  });
  await fs.writeFile(entryIndex, index, 'utf8');
  await fs.writeFile(path.join(offline, 'offline-manifest.json'), `${JSON.stringify({ version, entrypoint: 'docs/index.html', public_landing_included: false }, null, 2)}\n`, 'utf8');
  await fs.copyFile(path.join(out, 'documentation-build.json'), path.join(offline, 'documentation-build.json'));
  return offline;
}
