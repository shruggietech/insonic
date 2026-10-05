// SPDX-License-Identifier: Apache-2.0
import fs from 'node:fs/promises';
import path from 'node:path';
import { root } from '../lib/docs.mjs';

export async function walk(directory) {
  const results = [];
  for (const entry of await fs.readdir(directory, { withFileTypes: true })) {
    const filename = path.join(directory, entry.name);
    if (entry.isDirectory()) results.push(...await walk(filename));
    else results.push(filename);
  }
  return results;
}

function relativeTarget(filename, url, out) {
  if (!url.startsWith('/') || url.startsWith('//')) return url;
  const [pathname, fragment = ''] = url.split('#');
  const target = path.join(out, pathname, pathname.endsWith('/') ? 'index.html' : '');
  const relative = path.relative(path.dirname(filename), target).split(path.sep).join('/');
  return `${relative || 'index.html'}${fragment ? `#${fragment}` : ''}`;
}

export async function postbuild() {
  const out = path.join(root, 'site/out');
  for (const filename of await walk(out)) {
    if (filename.endsWith('.html')) {
      let html = await fs.readFile(filename, 'utf8');
      // The site uses server-rendered pages and plain links. Keeping only the local
      // scripts avoids a hydration runtime and makes file-based docs portable.
      html = html.replace(/<script\b[^>]*>[\s\S]*?<\/script>/g, (script) => /\bsrc="\/(?:site-runtime|docs-runtime|theme-init)\.js"/.test(script) ? script : '');
      html = html.replace(/<link\b[^>]*\bas="script"[^>]*>/g, '');
      html = html.replace(/\b(href|src|srcSet|srcset)="(\/[^" ]*)"/g, (_, attribute, value) => `${attribute}="${relativeTarget(filename, value, out)}"`);
      await fs.writeFile(filename, html, 'utf8');
    } else if (filename.endsWith('.css')) {
      const css = (await fs.readFile(filename, 'utf8')).replace(/url\((['"]?)(\/[^)'" ]+)\1\)/g, (_, quote, value) => `url(${quote}${relativeTarget(filename, value, out)}${quote})`);
      await fs.writeFile(filename, css, 'utf8');
    }
  }
  await fs.writeFile(path.join(out, '.nojekyll'), '', 'utf8');
}
