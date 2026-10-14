// SPDX-License-Identifier: Apache-2.0
import fs from 'node:fs/promises';
import path from 'node:path';
import crypto from 'node:crypto';

const hash = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
export async function prepareScreenshots(repositoryRoot, publicRoot) {
  const source = path.join(repositoryRoot, 'docs/screenshots');
  const publicDirectory = path.resolve(publicRoot);
  const destinationRoot = path.resolve(publicDirectory, 'screenshots');
  if (path.dirname(destinationRoot) !== publicDirectory) throw new Error('Invalid screenshot output directory.');
  await fs.rm(destinationRoot, { recursive: true, force: true });
  try { await fs.access(source); } catch { return []; }
  const copied = [];
  for (const version of await fs.readdir(source, { withFileTypes: true })) {
    if (!version.isDirectory() || !/^v[0-9]+\.[0-9]+\.[0-9]+$/.test(version.name)) throw new Error('Invalid screenshot version directory.');
    const directory = path.join(source, version.name);
    const names = await fs.readdir(directory);
    for (const name of names) if (!(await fs.lstat(path.join(directory, name))).isFile()) throw new Error('Screenshot inputs must be regular files.');
    const witnessed = new Set();
    const receipts = names.filter(name => /^[a-z0-9-]+-capture\.json$/.test(name));
    for (const name of receipts) {
      const receipt = JSON.parse(await fs.readFile(path.join(directory, name), 'utf8'));
      if (receipt.schema_version !== version.name.slice(1) || !Array.isArray(receipt.images) || !receipt.images.length
          || !receipt.fixture?.isolated || receipt.fixture.private_inputs !== false || !receipt.source?.revision
          || !/^[a-f0-9]{40}$/.test(receipt.source.revision) || !receipt.renderer?.kind || !receipt.locale) throw new Error('Screenshot provenance is incomplete.');
      for (const item of receipt.source.files || []) {
        if (typeof item.path !== 'string' || path.isAbsolute(item.path) || item.path.includes('..') || item.path.includes('\\')) throw new Error('Invalid capture source path.');
        if (hash(await fs.readFile(path.join(repositoryRoot, item.path))) !== item.sha256) throw new Error('Screenshot source differs from current implementation.');
      }
      for (const image of receipt.images) {
        const basename = path.posix.basename(image.path || '');
        if (![basename, `docs/screenshots/${version.name}/${basename}`].includes(image.path) || !/^[a-z0-9-]+\.png$/.test(basename) || witnessed.has(basename) || !image.landmark || !image.caption) throw new Error('Screenshot identity or accessible caption is incomplete.');
        const target = path.join(directory, basename);
        if (!(await fs.lstat(target)).isFile()) throw new Error('Screenshot must be a regular file.');
        const bytes = await fs.readFile(target);
        if (hash(bytes) !== image.sha256 || !bytes.subarray(0, 8).equals(Buffer.from([137,80,78,71,13,10,26,10]))) throw new Error('Screenshot bytes differ from capture receipt.');
        if (bytes.length < 24 || bytes.readUInt32BE(16) !== image.width || bytes.readUInt32BE(20) !== image.height) throw new Error('Screenshot dimensions differ from capture receipt.');
        witnessed.add(basename);
      }
    }
    for (const name of names) {
      if (!receipts.includes(name) && !witnessed.has(name)) throw new Error('Screenshot asset lacks verified capture provenance.');
      const destination = path.join(publicRoot, 'screenshots', version.name, name);
      await fs.mkdir(path.dirname(destination), { recursive: true });
      await fs.copyFile(path.join(directory, name), destination);
      copied.push(path.posix.join('screenshots', version.name, name));
    }
  }
  return copied;
}
