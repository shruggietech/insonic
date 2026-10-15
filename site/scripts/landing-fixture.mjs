// SPDX-License-Identifier: Apache-2.0
// Maintainer-only rendering of trusted repository JSX; never part of the export.
import { mkdir, mkdtemp, rm } from 'node:fs/promises';
import path from 'node:path';
import { createRequire } from 'node:module';
import { build } from 'esbuild';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { root } from '../lib/docs.mjs';
import { getBrandMessaging } from '../lib/brand.mjs';

export async function landingRenderer() {
  const cache = path.join(root, 'site/node_modules/.cache');
  await mkdir(cache, { recursive: true });
  const fixture = await mkdtemp(path.join(cache, 'insonic-landing-'));
  const filename = path.join(fixture, 'landing.cjs');
  const require = createRequire(import.meta.url);
  try {
    await build({ entryPoints: [path.join(root, 'site/app/components/LandingPage.jsx')], outfile: filename,
      bundle: true, platform: 'node', format: 'cjs', packages: 'external', jsx: 'automatic', logLevel: 'silent' });
    const LandingPage = require(filename).default;
    const brand = getBrandMessaging();
    return latest => renderToStaticMarkup(createElement(LandingPage, { latest, brand }));
  } finally {
    delete require.cache[filename];
    if (path.dirname(fixture) !== cache || !path.basename(fixture).startsWith('insonic-landing-')) throw new Error('Unexpected owned landing render path.');
    await rm(fixture, { recursive: true, force: true });
  }
}

export function replaceLanding(html, rendered) {
  const matches = html.match(/<main\b[^>]*data-public-landing="true"[\s\S]*?<\/main>/g) || [];
  if (matches.length !== 1 || !rendered.startsWith('<main ') || !rendered.endsWith('</main>')) throw new Error('Expected one real public landing frame.');
  return html.replace(matches[0], () => rendered);
}
