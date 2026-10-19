// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import { cp, mkdtemp, readFile, writeFile, rm, rename, symlink } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { verifyWebviewSource } from '../scripts/webview-source.mjs';

const root = fileURLToPath(new URL('../', import.meta.url));
async function fixture(run) {
  const directory = await mkdtemp(join(tmpdir(), 'insonic-webview-source-'));
  try {
    await cp(join(root, 'third_party'), join(directory, 'third_party'), { recursive: true });
    for (const file of ['go.mod', 'go.sum']) await cp(join(root, file), join(directory, file));
    await run(directory);
  } finally { await rm(directory, { recursive: true, force: true }); }
}

test('maintained WebView source preserves the exact upstream and explicit patch scope', async () => {
  const authority = await verifyWebviewSource(root);
  assert.equal(Object.keys(authority.upstream_files).length, 397);
  assert.deepEqual(authority.modified, ['pkg/combridge/iunknown.go', 'pkg/edge/ICoreWebView2Controller.go', 'pkg/edge/ICoreWebView2Deferral.go', 'pkg/edge/ICoreWebView2WebResourceRequestedEventArgs.go', 'pkg/edge/ICoreWebView2WebResourceResponse.go', 'pkg/edge/chromium.go', 'pkg/edge/corewebview2.go', 'pkg/edge/create_env_go.go']);
  assert.notEqual(authority.upstream_sum, authority.patched_sum);
});

for (const file of ['README.md', 'pkg/edge/window_lifetime_windows.go']) {
  test(`source authority rejects changed ${file} bytes`, async () => fixture(async directory => {
    const path = join(directory, 'third_party/go-webview2', file);
    await writeFile(path, `${await readFile(path, 'utf8')}\nchanged\n`);
    await assert.rejects(verifyWebviewSource(directory), /bytes differ/);
  }));
}

test('source authority rejects unlisted files and rewritten upstream hashes', async () => fixture(async directory => {
  const file = join(directory, 'third_party/go-webview2/unlisted.go');
  await writeFile(file, 'package unlisted\n');
  await assert.rejects(verifyWebviewSource(directory), /bytes differ|membership/);
  await rm(file);
  const manifest = join(directory, 'third_party/go-webview2.source.json');
  const authority = JSON.parse(await readFile(manifest, 'utf8'));
  authority.upstream_files['LICENSE'] = '0'.repeat(64);
  await writeFile(manifest, JSON.stringify(authority));
  await assert.rejects(verifyWebviewSource(directory), /checksum/);
}));

test('source authority rejects a redirected local replacement', async () => fixture(async directory => {
  const file = join(directory, 'go.mod');
  await writeFile(file, (await readFile(file, 'utf8')).replace('./third_party/go-webview2', '../unapproved'));
  await assert.rejects(verifyWebviewSource(directory), /replacement/);
}));

test('source authority rejects a linked module directory', async () => fixture(async directory => {
  const original = join(directory, 'third_party/go-webview2');
  const moved = join(directory, 'third_party/saved');
  await rename(original, moved);
  await symlink(moved, original, 'junction');
  await assert.rejects(verifyWebviewSource(directory), /linked/);
}));
