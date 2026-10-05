#!/usr/bin/env node
// SPDX-License-Identifier: Apache-2.0
// The upstream release owns kit bytes. This command only imports and records them.
import { createHash, randomUUID } from 'node:crypto';
import { mkdir, readFile, readdir, rename, rm, writeFile, lstat } from 'node:fs/promises';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { inflateRawSync } from 'node:zlib';

export const upstream = 'shruggietech/shruggie-brand';
const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const legalFiles = ['LICENSE', 'LICENSE-BRAND.md', 'NOTICE'];
const limits = Object.freeze({ archive: 64 * 1024 * 1024, expanded: 128 * 1024 * 1024, file: 16 * 1024 * 1024, entries: 3000 });
export const sha256 = (bytes) => createHash('sha256').update(bytes).digest('hex');
const jsonBytes = (value) => Buffer.from(`${JSON.stringify(value, null, 2)}\n`, 'utf8');

export function safePath(path) {
  if (typeof path !== 'string' || !path || path.includes('\\') || path.startsWith('/') || /[\x00-\x1f\x7f<>:"|?*]/.test(path)) throw new Error(`Unsafe kit path: ${path}`);
  for (const part of path.split('/')) {
    if (!part || part === '.' || part === '..' || /[. ]$/.test(part) || /^(con|prn|aux|nul|com[1-9]|lpt[1-9])(?:\.|$)/i.test(part)) throw new Error(`Unsafe kit path: ${path}`);
  }
  return path;
}

function registerPath(path, spellings) {
  const parts = path.split('/');
  for (let index = 1; index <= parts.length; index += 1) {
    const spelling = parts.slice(0, index).join('/');
    const key = spelling.normalize('NFC').toLowerCase();
    if (spellings.has(key) && spellings.get(key) !== spelling) throw new Error(`Case or Unicode path alias: ${path}`);
    spellings.set(key, spelling);
  }
}

function parseJson(bytes, label) {
  if (!bytes) throw new Error(`Missing ${label}`);
  return JSON.parse(new TextDecoder('utf-8', { fatal: true }).decode(bytes));
}

function versionParts(value) {
  if (!/^\d+\.\d+\.\d+$/.test(value ?? '')) throw new Error(`Unsupported version: ${value}`);
  const parts = value.split('.').map(Number);
  if (parts.some((part) => !Number.isSafeInteger(part))) throw new Error(`Invalid version: ${value}`);
  return parts;
}

export function compareVersions(left, right) {
  const a = versionParts(left); const b = versionParts(right);
  for (let index = 0; index < 3; index += 1) if (a[index] !== b[index]) return a[index] > b[index] ? 1 : -1;
  return 0;
}

function crc32(bytes) {
  let crc = 0xffffffff;
  for (const byte of bytes) {
    crc ^= byte;
    for (let bit = 0; bit < 8; bit += 1) crc = (crc >>> 1) ^ ((crc & 1) ? 0xedb88320 : 0);
  }
  return (crc ^ 0xffffffff) >>> 0;
}

// Read ordinary stored/deflated ZIPs in memory, before writing any extracted path.
// ZIP64, links, encrypted files, duplicate paths, and Windows path aliases are refused.
export function readZip(archive) {
  if (!Buffer.isBuffer(archive) || archive.length > limits.archive || archive.length < 22) throw new Error('Archive size is invalid');
  let end = -1;
  for (let offset = archive.length - 22; offset >= Math.max(0, archive.length - 65557); offset -= 1) {
    if (archive.readUInt32LE(offset) === 0x06054b50 && offset + 22 + archive.readUInt16LE(offset + 20) === archive.length) { end = offset; break; }
  }
  if (end < 0) throw new Error('ZIP end record is missing');
  const count = archive.readUInt16LE(end + 10);
  const centralSize = archive.readUInt32LE(end + 12);
  const centralStart = archive.readUInt32LE(end + 16);
  if (archive.readUInt16LE(end + 4) || archive.readUInt16LE(end + 6) || archive.readUInt16LE(end + 8) !== count || count === 65535 || count > limits.entries || !count || centralStart + centralSize !== end) throw new Error('Unsupported ZIP directory');
  const files = new Map(); const names = new Set(); const spellings = new Map(); const spans = [];
  let offset = centralStart; let expanded = 0;
  for (let index = 0; index < count; index += 1) {
    if (offset + 46 > end || archive.readUInt32LE(offset) !== 0x02014b50) throw new Error('Invalid ZIP directory entry');
    const flags = archive.readUInt16LE(offset + 8); const method = archive.readUInt16LE(offset + 10);
    const crc = archive.readUInt32LE(offset + 16); const compressedSize = archive.readUInt32LE(offset + 20); const size = archive.readUInt32LE(offset + 24);
    const nameLength = archive.readUInt16LE(offset + 28); const extraLength = archive.readUInt16LE(offset + 30); const commentLength = archive.readUInt16LE(offset + 32);
    const attributes = archive.readUInt32LE(offset + 38); const local = archive.readUInt32LE(offset + 42);
    const next = offset + 46 + nameLength + extraLength + commentLength;
    if (next > end || !nameLength || flags & 0x2041 || ![0, 8].includes(method) || archive.readUInt16LE(offset + 34) || [compressedSize, size, local].includes(0xffffffff)) throw new Error('Unsupported ZIP entry');
    const rawName = archive.subarray(offset + 46, offset + 46 + nameLength);
    if (!(flags & 0x800) && rawName.some((byte) => byte > 127)) throw new Error('ZIP filenames must use UTF-8');
    const originalName = new TextDecoder('utf-8', { fatal: true }).decode(rawName);
    const directory = originalName.endsWith('/'); const path = safePath(directory ? originalName.slice(0, -1) : originalName);
    const key = path.toLowerCase();
    if (names.has(key)) throw new Error(`Duplicate kit path: ${path}`);
    registerPath(path, spellings);
    names.add(key);
    const unixType = (attributes >>> 16) & 0xf000;
    if (unixType && unixType !== (directory ? 0x4000 : 0x8000)) throw new Error(`Unsupported file type: ${path}`);
    if (size > limits.file || (expanded += size) > limits.expanded) throw new Error('Expanded archive exceeds size limit');
    if (local + 30 > centralStart || archive.readUInt32LE(local) !== 0x04034b50 || archive.readUInt16LE(local + 6) !== flags || archive.readUInt16LE(local + 8) !== method) throw new Error(`Invalid local ZIP entry: ${path}`);
    const localNameLength = archive.readUInt16LE(local + 26); const localExtraLength = archive.readUInt16LE(local + 28);
    const dataStart = local + 30 + localNameLength + localExtraLength; const dataEnd = dataStart + compressedSize;
    if (dataEnd > centralStart || !archive.subarray(local + 30, local + 30 + localNameLength).equals(rawName)) throw new Error(`ZIP header disagreement: ${path}`);
    spans.push([local, dataEnd]);
    if (!(flags & 8) && (archive.readUInt32LE(local + 14) !== crc || archive.readUInt32LE(local + 18) !== compressedSize || archive.readUInt32LE(local + 22) !== size)) throw new Error(`ZIP size disagreement: ${path}`);
    const compressed = archive.subarray(dataStart, dataEnd);
    const bytes = method === 0 ? compressed : inflateRawSync(compressed, { maxOutputLength: Math.max(1, size) });
    if (bytes.length !== size || crc32(bytes) !== crc || (directory && size)) throw new Error(`ZIP payload mismatch: ${path}`);
    if (!directory) files.set(path, Buffer.from(bytes));
    offset = next;
  }
  if (offset !== end) throw new Error('Unexpected ZIP directory bytes');
  spans.sort((a, b) => a[0] - b[0]);
  for (let index = 1; index < spans.length; index += 1) if (spans[index][0] < spans[index - 1][1]) throw new Error('Overlapping ZIP entries');
  const fileKeys = new Set([...files.keys()].map((path) => path.toLowerCase()));
  for (const path of names) {
    const parts = path.split('/');
    for (let index = 1; index < parts.length; index += 1) if (fileKeys.has(parts.slice(0, index).join('/'))) throw new Error(`File is also a directory: ${path}`);
  }
  return files;
}

export function verifyKit(files, candidate) {
  const manifestBytes = files.get('manifest.json'); const manifest = parseJson(manifestBytes, 'manifest.json');
  if (manifest.name !== 'insonic-brand-kit' || !Array.isArray(manifest.files)) throw new Error('Manifest does not identify an Insonic kit');
  const inventory = new Map(); const names = new Set(); const spellings = new Map();
  for (const entry of manifest.files) {
    safePath(entry.path);
    if (names.has(entry.path.toLowerCase())) throw new Error(`Duplicate manifest path: ${entry.path}`);
    registerPath(entry.path, spellings);
    names.add(entry.path.toLowerCase());
    if (!Number.isSafeInteger(entry.bytes) || entry.bytes < 0 || !/^[a-f0-9]{64}$/.test(entry.sha256)) throw new Error(`Invalid manifest record: ${entry.path}`);
    const bytes = files.get(entry.path);
    if (!bytes || bytes.length !== entry.bytes || sha256(bytes) !== entry.sha256) throw new Error(`Manifest mismatch: ${entry.path}`);
    inventory.set(entry.path, entry.sha256);
  }
  for (const path of ['manifest.json', ...legalFiles]) {
    if (!files.has(path)) throw new Error(`Missing legal or manifest file: ${path}`);
    inventory.set(path, sha256(files.get(path)));
  }
  for (const path of files.keys()) if (!inventory.has(path)) throw new Error(`Unlisted kit file: ${path}`);
  const bundle = parseJson(files.get('enforcement/bundle.json'), 'enforcement/bundle.json');
  const consumer = parseJson(files.get('enforcement/consumer-contract.json'), 'enforcement/consumer-contract.json');
  versionParts(manifest.version); versionParts(bundle.package?.brandbuilder_version);
  if (bundle.package?.brand_slug !== 'insonic' || bundle.package?.brand_version !== manifest.version || bundle.publication?.status !== 'release' || bundle.versions?.brand_version !== manifest.version || manifest.consumer_contract?.versions?.compiler_version !== bundle.package.brandbuilder_version || consumer.versions?.brand_version !== manifest.version || consumer.versions?.compiler_version !== bundle.package.brandbuilder_version) throw new Error('Kit identity or consumer contract disagrees');
  if (candidate && (candidate.brandVersion !== manifest.version || candidate.brandbuilderVersion !== bundle.package.brandbuilder_version || candidate.archiveName !== bundle.package.filename || candidate.releaseTag !== bundle.publication.tag)) throw new Error('Kit does not match the selected release');
  if (!/^[a-f0-9]{40}$/.test(bundle.source_revision)) throw new Error('Kit source revision is missing');
  return { manifest, bundle, manifestSha256: sha256(manifestBytes), legalSha256: Object.fromEntries(legalFiles.map((path) => [path, inventory.get(path)])), fileCount: files.size };
}

async function requestBytes(url, maximum, fetchImplementation = fetch) {
  const response = await fetchImplementation(url, { headers: { Accept: 'application/vnd.github+json', 'User-Agent': 'insonic-brand-maintenance' }, signal: AbortSignal.timeout(60000) });
  if (!response.ok) throw new Error(`Upstream request returned ${response.status}: ${url}`);
  if (Number(response.headers.get('content-length') || 0) > maximum) throw new Error(`Download exceeds size limit: ${url}`);
  const chunks = []; let length = 0;
  for await (const chunk of response.body) {
    length += chunk.length;
    if (length > maximum) { await response.body.cancel().catch(() => {}); throw new Error(`Download exceeds size limit: ${url}`); }
    chunks.push(chunk);
  }
  return Buffer.concat(chunks);
}

export function selectRelease(releases) {
  if (!Array.isArray(releases)) throw new Error('Invalid upstream release list');
  const candidates = [];
  for (const release of releases) {
    if (release.draft || release.prerelease || !Array.isArray(release.assets)) continue;
    const checksums = release.assets.filter((asset) => asset.name === 'SHA256SUMS');
    const kits = release.assets.filter((asset) => /^insonic-brand-\d+\.\d+\.\d+-bb\d+\.\d+\.\d+\.zip$/.test(asset.name));
    if (!kits.length) continue;
    if (checksums.length !== 1 || kits.length !== 1) throw new Error('Upstream release has ambiguous kit or checksum assets');
    const archive = kits[0]; const match = archive.name.match(/^insonic-brand-(\d+\.\d+\.\d+)-bb(\d+\.\d+\.\d+)\.zip$/);
    const baseUrl = `https://github.com/${upstream}/releases/download/${encodeURIComponent(release.tag_name)}/`;
    if (archive.browser_download_url !== baseUrl + archive.name || checksums[0].browser_download_url !== baseUrl + 'SHA256SUMS') throw new Error('Release asset URL is outside the official upstream repository');
    candidates.push({ releaseTag: release.tag_name, releaseUrl: `https://github.com/${upstream}/releases/tag/${encodeURIComponent(release.tag_name)}`, publishedAt: release.published_at, archiveName: archive.name, archiveUrl: archive.browser_download_url, checksumUrl: checksums[0].browser_download_url, assetDigest: archive.digest, brandVersion: match[1], brandbuilderVersion: match[2] });
  }
  candidates.sort((a, b) => compareVersions(b.brandVersion, a.brandVersion) || compareVersions(b.brandbuilderVersion, a.brandbuilderVersion));
  if (!candidates.length) throw new Error('No stable upstream Insonic kit release found');
  return candidates[0];
}

export async function downloadKit(fetchImplementation = fetch) {
  const releases = parseJson(await requestBytes(`https://api.github.com/repos/${upstream}/releases?per_page=100`, 4 * 1024 * 1024, fetchImplementation), 'release list');
  const candidate = selectRelease(releases);
  const checksumBytes = await requestBytes(candidate.checksumUrl, 2 * 1024 * 1024, fetchImplementation);
  const checksumLines = new TextDecoder('utf-8', { fatal: true }).decode(checksumBytes).split(/\r?\n/).filter(Boolean);
  const matches = checksumLines.map((line) => line.match(/^([a-fA-F0-9]{64})\s+\*?(?:\.\/)?(.+)$/)).filter((match) => match?.[2] === candidate.archiveName);
  if (matches.length !== 1) throw new Error('Release checksum is missing or ambiguous');
  const archiveSha256 = matches[0][1].toLowerCase();
  if (candidate.assetDigest && candidate.assetDigest !== `sha256:${archiveSha256}`) throw new Error('GitHub asset digest disagrees with release checksums');
  const archiveBytes = await requestBytes(candidate.archiveUrl, limits.archive, fetchImplementation);
  if (sha256(archiveBytes) !== archiveSha256) throw new Error('Release archive SHA-256 mismatch');
  const files = readZip(archiveBytes); const verified = verifyKit(files, candidate);
  const { assetDigest, ...identity } = candidate;
  const lock = { schemaVersion: 1, sourceRepository: `https://github.com/${upstream}`, ...identity, archiveSha256, manifestSha256: verified.manifestSha256, legalSha256: verified.legalSha256, sourceRevision: verified.bundle.source_revision, canonVersion: verified.manifest.canon, fileCount: verified.fileCount };
  return { files, lock };
}

async function collectFiles(directory, prefix = '', files = new Map()) {
  const stat = await lstat(directory);
  if (!stat.isDirectory() || stat.isSymbolicLink()) throw new Error(`Kit directory is not a real directory: ${directory}`);
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    const path = prefix ? `${prefix}/${entry.name}` : entry.name; safePath(path);
    if (entry.isSymbolicLink()) throw new Error(`Kit symlink is forbidden: ${path}`);
    if (entry.isDirectory()) await collectFiles(join(directory, entry.name), path, files);
    else if (entry.isFile()) {
      const fileStat = await lstat(join(directory, entry.name));
      if (fileStat.size > limits.file || files.size >= limits.entries) throw new Error('Installed kit exceeds size limit');
      files.set(path, await readFile(join(directory, entry.name)));
    }
    else throw new Error(`Unsupported installed file: ${path}`);
  }
  return files;
}

export async function verifyInstalled(brandRoot) {
  let lockBytes;
  try {
    const lockStat = await lstat(join(brandRoot, 'lock.json'));
    if (!lockStat.isFile() || lockStat.isSymbolicLink() || lockStat.size > 64 * 1024) throw new Error('Integration lock must be a small regular file');
    lockBytes = await readFile(join(brandRoot, 'lock.json'));
  } catch (error) { if (error.code === 'ENOENT') { try { await lstat(join(brandRoot, 'kit')); } catch (kitError) { if (kitError.code === 'ENOENT') return null; throw kitError; } throw new Error('Installed kit has no integration lock'); } throw error; }
  const lock = parseJson(lockBytes, 'lock.json');
  if (lock.schemaVersion !== 1 || lock.sourceRepository !== `https://github.com/${upstream}` || !/^[a-f0-9]{64}$/.test(lock.archiveSha256 ?? '')) throw new Error('Invalid brand integration lock');
  const files = await collectFiles(join(brandRoot, 'kit')); const verified = verifyKit(files, lock);
  if (lock.manifestSha256 !== verified.manifestSha256 || lock.fileCount !== verified.fileCount || lock.sourceRevision !== verified.bundle.source_revision || legalFiles.some((path) => lock.legalSha256?.[path] !== verified.legalSha256[path])) throw new Error('Installed kit differs from the integration lock');
  return lock;
}

export function updateStatus(installed, candidate) {
  if (!installed) return 'available';
  const brandOrder = compareVersions(candidate.brandVersion, installed.brandVersion);
  const compilerOrder = compareVersions(candidate.brandbuilderVersion, installed.brandbuilderVersion);
  if (brandOrder < 0 || (brandOrder === 0 && compilerOrder < 0)) throw new Error('Upstream kit is older than the installed kit; refusing a downgrade');
  if (brandOrder === 0 && compilerOrder === 0) {
    if (candidate.archiveSha256 !== installed.archiveSha256 || candidate.manifestSha256 !== installed.manifestSha256 || candidate.releaseTag !== installed.releaseTag) throw new Error('Upstream changed an already installed kit version; review the release manually');
    return 'current';
  }
  return 'available';
}

// Stage all bytes, validate them again, then swap. Roll back both paths on failure.
export async function installKit(brandRoot, kit, renameImplementation = rename) {
  await mkdir(brandRoot, { recursive: true });
  const brandStat = await lstat(brandRoot);
  if (!brandStat.isDirectory() || brandStat.isSymbolicLink()) throw new Error('Brand destination must be a real directory');
  const guard = join(brandRoot, '.update-lock');
  try { await mkdir(guard); } catch (error) { if (error.code === 'EEXIST') throw new Error('Another brand update is active; inspect brand/.update-lock if a previous update was interrupted'); throw error; }
  const staging = join(brandRoot, `.staging-${randomUUID()}`); const backup = join(staging, 'previous-kit'); const backupLock = join(staging, 'previous-lock.json');
  let movedKit = false; let movedLock = false; let placedKit = false; let placedLock = false; let preserveRecovery = false;
  try {
    await mkdir(join(staging, 'kit'), { recursive: true });
    for (const [path, bytes] of kit.files) { safePath(path); const target = join(staging, 'kit', ...path.split('/')); await mkdir(dirname(target), { recursive: true }); await writeFile(target, bytes); }
    await writeFile(join(staging, 'lock.json'), jsonBytes(kit.lock));
    await verifyInstalled(staging);
    const current = await verifyInstalled(brandRoot);
    if (updateStatus(current, kit.lock) === 'current') return false;
    if (current) { await renameImplementation(join(brandRoot, 'kit'), backup); movedKit = true; await renameImplementation(join(brandRoot, 'lock.json'), backupLock); movedLock = true; }
    await renameImplementation(join(staging, 'kit'), join(brandRoot, 'kit')); placedKit = true;
    await renameImplementation(join(staging, 'lock.json'), join(brandRoot, 'lock.json')); placedLock = true;
    await verifyInstalled(brandRoot);
    return true;
  } catch (error) {
    try {
      if (placedLock) await rm(join(brandRoot, 'lock.json'), { force: true });
      if (placedKit) await rm(join(brandRoot, 'kit'), { recursive: true, force: true });
      if (movedKit) await rename(backup, join(brandRoot, 'kit'));
      if (movedLock) await rename(backupLock, join(brandRoot, 'lock.json'));
    } catch (rollbackError) { preserveRecovery = true; throw new AggregateError([error, rollbackError], `Update failed and automatic restoration failed. Recovery files remain at ${staging}`); }
    throw error;
  } finally {
    if (!preserveRecovery) await rm(staging, { recursive: true, force: true });
    await rm(guard, { recursive: true, force: true });
  }
}

export async function main(argv = process.argv.slice(2)) {
  const mode = argv.find((arg) => ['--check', '--update', '--verify'].includes(arg));
  if (argv.includes('--help') || !mode) { console.log('Usage: node scripts/sync-brand-kit.mjs --check|--update|--verify [--json]\n--check checks the newest formal upstream kit without writing files.\n--update imports a newer verified kit.\n--verify checks installed kit bytes offline.'); return; }
  if (argv.filter((arg) => ['--check', '--update', '--verify'].includes(arg)).length !== 1 || argv.some((arg) => ![mode, '--json'].includes(arg))) throw new Error('Use exactly one mode, optionally followed by --json');
  const brandRoot = join(repositoryRoot, 'brand'); const installed = await verifyInstalled(brandRoot);
  let result;
  if (mode === '--verify') { if (!installed) throw new Error('No integrated brand kit'); result = { status: 'verified', installed }; }
  else { const kit = await downloadKit(); const status = updateStatus(installed, kit.lock); const updated = mode === '--update' && status === 'available' ? await installKit(brandRoot, kit) : false; result = { status: updated ? 'updated' : status, installed: updated ? kit.lock : installed, upstream: kit.lock }; }
  if (argv.includes('--json')) console.log(JSON.stringify(result, null, 2));
  else console.log(`Brand kit ${result.status}: Insonic ${(result.upstream ?? result.installed).brandVersion}, BrandBuilder ${(result.upstream ?? result.installed).brandbuilderVersion}.${result.status === 'available' ? ' Run with --update to integrate it.' : ''}`);
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) main().catch((error) => { console.error(error.message); process.exitCode = 1; });
