// SPDX-License-Identifier: Apache-2.0
import fs from 'node:fs/promises';
import path from 'node:path';
import { build } from 'esbuild';
import { spawnSync } from 'node:child_process';
import { root, getManifest, getRepositoryReferences } from '../lib/docs.mjs';
import { loadSchemaCatalog } from '../../scripts/schema-catalog.mjs';
import { documentationTarget } from '../lib/release-status.mjs';

export async function prepare() {
  const site = path.join(root, 'site');
  const kit = path.join(root, 'brand', 'kit');
  const manifest = getManifest();
  const releaseVersion = (await fs.readFile(path.join(root, 'VERSION'), 'utf8')).trim();
  const packageVersion = JSON.parse(await fs.readFile(path.join(site, 'package.json'), 'utf8')).version;
  if (manifest.latest !== releaseVersion || packageVersion !== releaseVersion) throw new Error(`Documentation latest (${manifest.latest}), site package (${packageVersion}), and VERSION (${releaseVersion}) must match.`);
  await fs.mkdir(path.join(site, 'public'), { recursive: true });
  for (const directory of ['logos', 'fonts', 'tokens']) {
    await fs.cp(path.join(kit, directory), path.join(site, 'public', 'brand', directory), { recursive: true, force: true });
  }
  await fs.cp(path.join(kit, 'icons', 'web'), path.join(site, 'public', 'brand', 'icons', 'web'), { recursive: true, force: true });
  const styles = [];
  for (const name of ['colors', 'interface', 'spacing', 'typography']) {
    const css = await fs.readFile(path.join(kit, 'tokens', `${name}.css`), 'utf8');
    styles.push(css);
    const light = css.match(/\.(?:in|bb)-light\s*\{[^}]+\}/);
    if (light) styles.push(`@media (prefers-color-scheme: light) { ${light[0].replace(/\.(?:in|bb)-light/, ':root')} }`);
    const dark = css.match(/:root\s*\{[^}]+\}/);
    if (light) styles.push(light[0].replace(/\.(?:in|bb)-light/, ':root[data-theme="light"]'));
    if (light && dark) styles.push(dark[0].replace(':root', ':root[data-theme="dark"]'));
  }
  await fs.writeFile(path.join(site, 'public', 'brand', 'theme.css'), `${styles.join('\n')}\n`, 'utf8');
  await fs.writeFile(path.join(site, 'public', 'documentation-manifest.json'), `${JSON.stringify(manifest, null, 2)}\n`, 'utf8');
  const git = args => {
    const result = spawnSync('git', args, { cwd: root, windowsHide: true, shell: false, encoding: 'utf8', timeout: 30000, env: { ...process.env, GIT_TERMINAL_PROMPT: '0' } });
    if (result.status !== 0) throw new Error('Documentation source identity could not be read.');
    return result.stdout.trim();
  };
  await fs.writeFile(path.join(site, 'public', 'documentation-build.json'), `${JSON.stringify({ version: releaseVersion, revision: git(['rev-parse', 'HEAD']), source_dirty: Boolean(git(['status', '--porcelain', '--untracked-files=normal'])), documentation_target: documentationTarget() }, null, 2)}\n`, 'utf8');
  for (const entry of manifest.versions) {
    const catalog = loadSchemaCatalog(entry.version);
    const destination = path.join(site, 'public', 'schemas', `v${entry.version}`);
    await fs.mkdir(destination, { recursive: true });
    for (const item of catalog.schemas) await fs.writeFile(path.join(destination, item.filename), `${JSON.stringify(item.schema, null, 2)}\n`, 'utf8');
  }
  for (const item of getRepositoryReferences()) {
    const directory = path.join(site, 'public', 'reference-sources', `v${item.entry.version}`);
    await fs.mkdir(directory, { recursive: true });
    await fs.writeFile(path.join(directory, `${item.slug}.txt`), item.source, 'utf8');
  }
  await build({
    entryPoints: [path.join(site, 'scripts', 'theme-init.js')],
    outfile: path.join(site, 'public', 'theme-init.js'),
    bundle: true,
    minify: true,
    format: 'iife',
    platform: 'browser',
    target: ['es2022'],
  });
  await build({
    entryPoints: [path.join(site, 'scripts', 'site-runtime.js')],
    outfile: path.join(site, 'public', 'site-runtime.js'),
    bundle: true,
    minify: true,
    format: 'iife',
    platform: 'browser',
    target: ['es2022'],
  });
  await build({
    entryPoints: [path.join(site, 'scripts', 'runtime.js')],
    outfile: path.join(site, 'public', 'docs-runtime.js'),
    bundle: true,
    minify: true,
    format: 'iife',
    platform: 'browser',
    target: ['es2022'],
    legalComments: 'linked',
  });
}
