// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import { cp, mkdtemp, readFile, writeFile, rm, rename, symlink, mkdir, chmod } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { verifyWebviewSource, verifyWailsSource } from '../scripts/webview-source.mjs';

const root = fileURLToPath(new URL('../', import.meta.url));
async function fixture(run, module = 'go-webview2') {
  const directory = await mkdtemp(join(tmpdir(), 'insonic-webview-source-'));
  try {
    await mkdir(join(directory, 'third_party'));
    await cp(join(root, 'third_party', module), join(directory, 'third_party', module), { recursive: true });
    await cp(join(root, 'third_party', module + '.source.json'), join(directory, 'third_party', module + '.source.json'));
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

test('maintained Wails source keeps the complete pinned baseline and three behavioral files', async () => {
  const authority = await verifyWailsSource(root);
  assert.equal(Object.keys(authority.upstream_files).length, 956);
  assert.equal(authority.upstream_sum, 'h1:hLYrSHUUq6hRXl3rWT9ZXiyFIgM94nVN9Ns6iEJzxyw=');
  assert.deepEqual(authority.modified, ['internal/app/app_dev.go', 'internal/app/app_production.go', 'internal/frontend/desktop/windows/frontend.go']);
  assert.deepEqual(authority.added, ['LICENSE.insonic', 'NOTICE.insonic', 'internal/app/app_lifecycle_test.go']);
  assert.notEqual(authority.upstream_sum, authority.patched_sum);
});

test('Wails source authority rejects unchanged-file tampering and redirected replacements', async () => fixture(async directory => {
  const file = join(directory, 'third_party/wails/internal/app/app.go');
  const original = await readFile(file);
  await chmod(file, 0o600);
  await writeFile(file, Buffer.concat([original, Buffer.from('\nchanged\n')]));
  await assert.rejects(verifyWailsSource(directory), /bytes differ/);
  await writeFile(file, original);
  const mod = join(directory, 'go.mod');
  await writeFile(mod, (await readFile(mod, 'utf8')).replace('./third_party/wails', '../unapproved-framework'));
  await assert.rejects(verifyWailsSource(directory), /replacement/);
}, 'wails'));

for (const [directory,module,version,verify] of [
  ['go-webview2','github.com/wailsapp/go-webview2','v1.0.23',verifyWebviewSource],
  ['wails','github.com/wailsapp/wails/v2','v2.16.0',verifyWailsSource],
]) {
  test(directory+' rejects missing, changed or conflicting archive/module sums',async()=>fixture(async repository=>{
    const sumFile=join(repository,'go.sum'),original=await readFile(sumFile,'utf8');
    const authority=await verify(repository);
    for(const suffix of ['/go.mod','']){
      const prefix=module+' '+version+suffix+' ',matches=original.split('\n').filter(value=>value.startsWith(prefix));
      assert.equal(matches.length,1);
      const line=matches[0],bad=prefix+'h1:'+'A'.repeat(43)+'=';
      for(const changed of [original.split('\n').filter(value=>value!==line).join('\n'),original.replace(line,bad),original+bad+'\n']){
        await writeFile(sumFile,changed);
        await assert.rejects(verify(repository),/bound to go.sum/);
      }
      await writeFile(sumFile,original);
      assert.equal((await verify(repository)).upstream_sum,authority.upstream_sum);
    }
  },directory));
}
