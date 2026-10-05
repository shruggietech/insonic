// SPDX-License-Identifier: Apache-2.0
import http from 'node:http';
import fs from 'node:fs/promises';
import path from 'node:path';
import { root } from '../lib/docs.mjs';

const out = path.join(root, 'site/out');
const port = Number(process.env.PORT || 4173);
const types = { '.html': 'text/html; charset=utf-8', '.css': 'text/css; charset=utf-8', '.js': 'text/javascript; charset=utf-8', '.json': 'application/json; charset=utf-8', '.txt': 'text/plain; charset=utf-8', '.svg': 'image/svg+xml', '.woff2': 'font/woff2', '.png': 'image/png' };
http.createServer(async (request, response) => {
  try {
    const pathname = decodeURIComponent(new URL(request.url, 'http://localhost').pathname);
    const filename = path.resolve(out, `.${pathname}`, pathname.endsWith('/') ? 'index.html' : '');
    if (!filename.startsWith(`${out}${path.sep}`)) throw new Error('Invalid path');
    const bytes = await fs.readFile(filename);
    response.writeHead(200, { 'Content-Type': types[path.extname(filename)] || 'application/octet-stream', 'X-Content-Type-Options': 'nosniff' });
    response.end(bytes);
  } catch {
    response.writeHead(404, { 'Content-Type': 'text/plain; charset=utf-8' });
    response.end('Not found');
  }
}).listen(port, '127.0.0.1', () => console.log(`Documentation preview: http://127.0.0.1:${port}`));
