// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import { liveInventory, checkLiveSite, closeOwnedBrowser } from '../scripts/check-live-site-ui.mjs';
import { mkdtemp, rm } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';

test('live browser inventory only admits bounded exact production URLs and SHA256', () => {
  const item = { url: 'https://insonic.io/', sha256: 'a'.repeat(64), path: 'index.html' };
  assert.equal(liveInventory({ paths: [item] }).size, 1);
  for (const changed of [{url:'https://evil.test/'}, {url:'https://insonic.io/?token=secret'}, {sha256:'bad'}]) {
    assert.throws(() => liveInventory({ paths: [{...item, ...changed}] }));
  }
  assert.throws(() => liveInventory({ paths: [item, item] }));
  assert.throws(() => liveInventory({ paths: [] }));
  const latest = {url:'https://insonic.io/docs/latest/index.html',sha256:'b'.repeat(64)};
  const aliases = liveInventory({paths:[item,latest]});
  assert.equal(aliases.get('https://insonic.io/docs/latest/'),latest.sha256);
  assert.equal(aliases.has('https://insonic.io/docs/v0.0.0/'),false);
});

test('a stalled graceful close kills only the retained owned browser server', async () => {
  let killed = 0;
  await closeOwnedBrowser({close:()=>new Promise(()=>{}),kill:async()=>{killed++;}}, 5);
  assert.equal(killed,1);
  await closeOwnedBrowser({close:async()=>{},kill:async()=>{throw new Error('unexpected kill');}},5);
});

test('browser verification rejects stale responses before launching a browser', async () => {
  const output = await mkdtemp(path.join(os.tmpdir(), 'live-ui-test-'));
  try {
    await assert.rejects(checkLiveSite({receipt:{paths:[{url:'https://insonic.io/',sha256:'a'.repeat(64)}]}, output,
      reader:async()=>({body:Buffer.from('stale'),type:'text/html'})}), /digest/);
  } finally { await rm(output, {recursive:true,force:true}); }
});
