// SPDX-License-Identifier: Apache-2.0
import { build } from 'esbuild';
import { mkdir, copyFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
const root = path.dirname(fileURLToPath(import.meta.url));
const assets = path.resolve(root, '../assets/bundle');
await mkdir(assets, { recursive: true });
await build({
  entryPoints: [path.join(root, 'src/main.tsx')],
  bundle: true,
  format: 'iife',
  minify: true,
  target: 'es2022',
  outfile: path.join(assets, 'app.js'),
  jsx: 'automatic',
  define: { 'process.env.NODE_ENV': '"production"' },
  nodePaths: [path.join(root, 'node_modules')],
  loader: { '.woff2': 'file' },
  assetNames: 'fonts/[name]-[hash]',
});
await copyFile(
  path.join(root, 'src/app.css'),
  path.join(assets, 'product.css'),
);
if (process.argv.includes('--tests')) {
  await mkdir(path.join(root, '.test-build'), { recursive: true });
  await build({
    entryPoints: [
      path.join(root, 'src/App.tsx'),
      path.join(root, 'src/client.ts'),
      path.join(root, 'src/forms.ts'),
      path.join(root, 'src/qualification.ts'),
    ],
    bundle: true,
    packages: 'external',
    platform: 'node',
    format: 'esm',
    outdir: path.join(root, '.test-build'),
    jsx: 'automatic',
    nodePaths: [path.join(root, 'node_modules')],
  });
}
