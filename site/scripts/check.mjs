// SPDX-License-Identifier: Apache-2.0
import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { root, getManifest, getRepositoryReferences } from '../lib/docs.mjs';
import { walk } from './postbuild.mjs';

export async function checkExport({ out = path.join(root, 'site/out'), offline = false } = {}) {
  const manifest = getManifest();
  const failures = [];
  const expected = ['index.html', 'docs/index.html', ...manifest.versions.flatMap((entry) => [`docs/v${entry.version}/index.html`, ...entry.pages.map((page) => `docs/v${entry.version}/${page.slug}/index.html`)]), ...getRepositoryReferences().map((item) => `docs/v${item.entry.version}/references/${item.slug}/index.html`)];
  for (const filename of expected) {
    try { await fs.access(path.join(out, filename)); } catch { failures.push(`Missing exported page: ${filename}`); }
  }
  const files = await walk(out);
  for (const filename of files.filter((file) => file.endsWith('.html'))) {
    const html = await fs.readFile(filename, 'utf8');
    const relativePage = path.relative(out, filename).split(path.sep).join('/');
    const scripts = [...html.matchAll(/<script\b[^>]*\bsrc="([^"]+)"/g)].map(([, source]) => path.posix.basename(source));
    if (!offline && relativePage === 'index.html' && scripts.includes('docs-runtime.js')) failures.push('The public landing page must not load the Mermaid runtime.');
    if (relativePage.startsWith('docs/') && (!scripts.includes('docs-runtime.js') || !scripts.includes('site-runtime.js'))) failures.push(`Documentation controls or Mermaid runtime missing: ${relativePage}`);
    if (offline && html.includes('data-public-landing')) failures.push(`Public landing page leaked into offline help: ${path.relative(out, filename)}`);
    if (/\b(?:href|src)="\/(?!\/)/.test(html)) failures.push(`Root-absolute URL in ${path.relative(out, filename)}`);
    for (const [, value] of html.matchAll(/\b(?:href|src|srcset)="([^" ]+)"/g)) {
      if (/^(?:[a-z]+:|\/\/)/i.test(value)) continue;
      const [relative, fragment] = value.split('#');
      const target = relative ? path.resolve(path.dirname(filename), relative) : filename;
      if (!target.startsWith(`${out}${path.sep}`) && target !== out) { failures.push(`Link leaves export: ${value}`); continue; }
      try {
        await fs.access(target);
        if (fragment && target.endsWith('.html')) {
          const linked = target === filename ? html : await fs.readFile(target, 'utf8');
          const escaped = decodeURIComponent(fragment).replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
          if (!new RegExp(`\\bid="${escaped}"`).test(linked)) failures.push(`Missing anchor ${value} in ${path.relative(out, filename)}`);
        }
      } catch { failures.push(`Broken link ${value} in ${path.relative(out, filename)}`); }
    }
  }
  if (failures.length) throw new Error(failures.join('\n'));
  console.log(`Documentation export checked: ${expected.length} pages; relative links and anchors resolve.`);
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) await checkExport();
