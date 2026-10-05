// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict';
import { mkdtemp, readFile, readdir, rename, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';
import { downloadKit, installKit, readZip, safePath, selectRelease, sha256, updateStatus, verifyInstalled, verifyKit } from '../scripts/sync-brand-kit.mjs';

const encode = (value) => Buffer.from(`${JSON.stringify(value)}\n`);
function crc32(bytes) {
  let crc = 0xffffffff;
  for (const byte of bytes) { crc ^= byte; for (let bit = 0; bit < 8; bit += 1) crc = (crc >>> 1) ^ ((crc & 1) ? 0xedb88320 : 0); }
  return (crc ^ 0xffffffff) >>> 0;
}

function zip(entries) {
  const localRecords = []; const centralRecords = []; let offset = 0;
  for (const [path, bytes, attributes = 0] of entries) {
    const name = Buffer.from(path); const checksum = crc32(bytes);
    const local = Buffer.alloc(30); local.writeUInt32LE(0x04034b50); local.writeUInt16LE(20, 4); local.writeUInt32LE(checksum, 14); local.writeUInt32LE(bytes.length, 18); local.writeUInt32LE(bytes.length, 22); local.writeUInt16LE(name.length, 26);
    localRecords.push(local, name, bytes);
    const central = Buffer.alloc(46); central.writeUInt32LE(0x02014b50); central.writeUInt16LE(20, 4); central.writeUInt16LE(20, 6); central.writeUInt32LE(checksum, 16); central.writeUInt32LE(bytes.length, 20); central.writeUInt32LE(bytes.length, 24); central.writeUInt16LE(name.length, 28); central.writeUInt32LE(attributes, 38); central.writeUInt32LE(offset, 42);
    centralRecords.push(central, name); offset += local.length + name.length + bytes.length;
  }
  const central = Buffer.concat(centralRecords); const end = Buffer.alloc(22); end.writeUInt32LE(0x06054b50); end.writeUInt16LE(entries.length, 8); end.writeUInt16LE(entries.length, 10); end.writeUInt32LE(central.length, 12); end.writeUInt32LE(offset, 16);
  return Buffer.concat([...localRecords, central, end]);
}

function fixture(version = '1.0.0') {
  const archiveName = `insonic-brand-${version}-bb3.0.1.zip`;
  const bundle = { package: { id: archiveName.slice(0, -4), filename: archiveName, brand_slug: 'insonic', brand_version: version, brandbuilder_version: '3.0.1' }, versions: { brand_version: version }, source_revision: 'a'.repeat(40), publication: { status: 'release', tag: 'v3.0.1' } };
  const files = new Map([['enforcement/bundle.json', encode(bundle)], ['enforcement/consumer-contract.json', encode({ versions: { brand_version: version, compiler_version: '3.0.1' } })], ['logos/example.svg', Buffer.from('<svg/>')]]);
  const manifest = { name: 'insonic-brand-kit', version, canon: '2.0.0', consumer_contract: { versions: { compiler_version: '3.0.1' } }, files: [...files].map(([path, bytes]) => ({ path, bytes: bytes.length, sha256: sha256(bytes) })) };
  files.set('manifest.json', encode(manifest));
  for (const path of ['LICENSE', 'LICENSE-BRAND.md', 'NOTICE']) files.set(path, Buffer.from(`Legal ${path}\n`));
  const verified = verifyKit(files);
  const archive = zip([...files]);
  const lock = { schemaVersion: 1, sourceRepository: 'https://github.com/shruggietech/shruggie-brand', archiveName, archiveSha256: sha256(archive), brandVersion: version, brandbuilderVersion: '3.0.1', releaseTag: 'v3.0.1', sourceRevision: bundle.source_revision, manifestSha256: verified.manifestSha256, legalSha256: verified.legalSha256, fileCount: files.size };
  return { files, archive, lock };
}

function release(kit, overrides = {}) {
  const base = 'https://github.com/shruggietech/shruggie-brand/releases/download/v3.0.1/';
  return { tag_name: 'v3.0.1', draft: false, prerelease: false, published_at: '2026-10-03T23:54:23Z', assets: [{ name: kit.lock.archiveName, browser_download_url: base + kit.lock.archiveName, digest: `sha256:${kit.lock.archiveSha256}` }, { name: 'SHA256SUMS', browser_download_url: base + 'SHA256SUMS' }], ...overrides };
}

test('archive reader rejects traversal, Windows aliases, links, and case duplicates', () => {
  for (const path of ['../escape', '/absolute', 'a\\b', 'C:/file', 'a/../b', 'a//b', 'CON.txt', 'a/trailing.', 'a/trailing ']) assert.throws(() => safePath(path), /Unsafe/);
  assert.throws(() => readZip(zip([['../escape', Buffer.from('x')]])), /Unsafe/);
  assert.throws(() => readZip(zip([['a', Buffer.from('x')], ['A', Buffer.from('y')]])), /Duplicate/);
  assert.throws(() => readZip(zip([['A/x', Buffer.from('x')], ['a/y', Buffer.from('y')]])), /path alias/);
  assert.throws(() => readZip(zip([['link', Buffer.from('x'), (0xa000 << 16) >>> 0]])), /file type/);
  assert.throws(() => readZip(zip([['a', Buffer.from('x')], ['a/b', Buffer.from('y')]])), /also a directory/);
});

test('archive and manifest validation detect payload changes and unlisted files', () => {
  const kit = fixture(); const files = readZip(kit.archive);
  assert.equal(verifyKit(files, kit.lock).fileCount, kit.files.size);
  const damaged = Buffer.from(kit.archive); damaged[30 + Buffer.byteLength('enforcement/bundle.json')] ^= 1;
  assert.throws(() => readZip(damaged), /payload mismatch/);
  files.set('logos/example.svg', Buffer.from('<changed/>'));
  assert.throws(() => verifyKit(files), /Manifest mismatch/);
  const extra = new Map(kit.files); extra.set('unexpected.txt', Buffer.from('extra'));
  assert.throws(() => verifyKit(extra), /Unlisted/);
});

test('newest stable Insonic kit wins; unrelated and prerelease releases are ignored', () => {
  const old = fixture(); const newer = fixture('1.0.1');
  assert.equal(selectRelease([release(old), release(newer)]).brandVersion, '1.0.1');
  assert.equal(selectRelease([release(newer, { prerelease: true }), release(old), { draft: false, assets: [] }]).brandVersion, '1.0.0');
  const bad = release(old); bad.assets[0].browser_download_url = 'https://example.com/kit.zip';
  assert.throws(() => selectRelease([bad]), /outside the official/);
});

test('download checks the release digest and full kit without writing files', async () => {
  const kit = fixture(); const seen = [];
  const fakeFetch = async (url) => { seen.push(url); if (url.includes('/releases?')) return new Response(JSON.stringify([release(kit)])); if (url.endsWith('SHA256SUMS')) return new Response(`${kit.lock.archiveSha256}  ./${kit.lock.archiveName}\n`); return new Response(kit.archive); };
  const result = await downloadKit(fakeFetch);
  assert.equal(result.lock.archiveSha256, kit.lock.archiveSha256);
  assert.equal(result.lock.manifestSha256, kit.lock.manifestSha256);
  assert.equal(seen.length, 3);
  await assert.rejects(downloadKit(async (url) => url.endsWith('.zip') ? new Response(Buffer.from('wrong')) : fakeFetch(url)), /SHA-256 mismatch/);
});

test('same versions cannot be silently replaced or downgraded', () => {
  const current = fixture().lock;
  assert.equal(updateStatus(current, current), 'current');
  assert.equal(updateStatus(current, fixture('1.0.1').lock), 'available');
  assert.throws(() => updateStatus(fixture('1.0.1').lock, current), /downgrade/);
  assert.throws(() => updateStatus(current, { ...current, archiveSha256: 'b'.repeat(64) }), /changed an already installed/);
});

test('failed staged replacement restores previous kit and lock', async () => {
  const directory = await mkdtemp(join(tmpdir(), 'insonic-brand-test-'));
  try {
    const original = fixture(); await installKit(directory, original);
    const previousLock = await readFile(join(directory, 'lock.json'));
    let moves = 0;
    await assert.rejects(installKit(directory, fixture('1.0.1'), async (...args) => { moves += 1; if (moves === 4) throw new Error('Simulated lock replacement failure'); return rename(...args); }), /Simulated/);
    assert.deepEqual(await readFile(join(directory, 'lock.json')), previousLock);
    assert.equal((await verifyInstalled(directory)).brandVersion, '1.0.0');
    assert.deepEqual((await readdir(directory)).sort(), ['kit', 'lock.json']);
    await writeFile(join(directory, 'kit', 'logos', 'example.svg'), '<changed/>');
    await assert.rejects(verifyInstalled(directory), /Manifest mismatch/);
  } finally { await rm(directory, { recursive: true, force: true }); }
});
