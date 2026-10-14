// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import crypto from 'node:crypto';
import { prepareScreenshots } from '../lib/screenshots.mjs';
import { copyOfflineFiles } from '../scripts/offline.mjs';
import { renderDocument } from '../lib/docs.mjs';
const digest = data => crypto.createHash('sha256').update(data).digest('hex');

test('verified PNGs and receipts remain byte-identical offline; unrelated host files are excluded', async t => {
  const root = await fs.mkdtemp(path.join(os.tmpdir(),'insonic-screenshot-test-'));
  t.after(() => fs.rm(root, { recursive:true,force:true }));
  const source = path.join(root,'docs/screenshots/v1.0.1'); await fs.mkdir(source,{recursive:true});
  const png = Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+a2uoAAAAASUVORK5CYII=', 'base64');
  const implementation = Buffer.from('fixture current implementation'); await fs.writeFile(path.join(root,'implementation.txt'),implementation);
  await fs.writeFile(path.join(source,'fixture.png'),png);
  const receipt = {kind:'application-ui-capture',schema_version:'1.0.1',source:{revision:'a'.repeat(40),dirty:false,files:[{path:'implementation.txt',sha256:digest(implementation)}]},renderer:{kind:'chromium-headless'},fixture:{isolated:true,private_inputs:false},locale:'en-US',images:[{path:'fixture.png',sha256:digest(png),width:1,height:1,landmark:'Fixture',caption:'Figure 1. Fixture surface.'}]};
  await fs.writeFile(path.join(source,'application-capture.json'),JSON.stringify(receipt));
  const out = path.join(root,'out');
  for (const badSource of [{...receipt.source,dirty:true},{...receipt.source,dirty:undefined},{...receipt.source,files:[]},{...receipt.source,files:undefined}]) {
    await fs.writeFile(path.join(source,'application-capture.json'),JSON.stringify({...receipt,source:badSource}));
    await assert.rejects(prepareScreenshots(root,out), /provenance is incomplete/);
    await assert.rejects(fs.access(path.join(out,'screenshots/v1.0.1/fixture.png')));
  }
  await fs.writeFile(path.join(source,'application-capture.json'),JSON.stringify(receipt));
  await prepareScreenshots(root,out);
  await fs.writeFile(path.join(out,'owner-profile.txt'),'excluded private sentinel');
  const offline=path.join(root,'offline'); await copyOfflineFiles(out,offline);
  assert.deepEqual(await fs.readFile(path.join(offline,'screenshots/v1.0.1/fixture.png')),png);
  assert.deepEqual(await fs.readFile(path.join(offline,'screenshots/v1.0.1/application-capture.json')),await fs.readFile(path.join(source,'application-capture.json')));
  await assert.rejects(fs.access(path.join(offline,'owner-profile.txt')));
  await fs.writeFile(path.join(source,'fixture.png'),Buffer.from('changed'));
  await assert.rejects(prepareScreenshots(root,out), /bytes differ/);
  await fs.writeFile(path.join(source,'fixture.png'),png); await fs.writeFile(path.join(root,'implementation.txt'),'stale');
  await assert.rejects(prepareScreenshots(root,out), /source differs/);
});

test('documentation images use bounded versioned routes and require alternative text', () => {
  const entry={version:'1.0.1',pages:[]};
  const rendered=renderDocument('![Optional setup location](../screenshots/v1.0.1/setup-location.png)',entry);
  assert.match(rendered.html,/src="\/screenshots\/v1.0.1\/setup-location.png"/);
  assert.match(rendered.html,/alt="Optional setup location"/);
  assert.throws(()=>renderDocument('![](../screenshots/v1.0.1/setup-location.png)',entry),/alternative text/);
  assert.throws(()=>renderDocument('![Wrong version](../screenshots/v1.0.0/setup-location.png)',entry),/versioned/);
  assert.doesNotMatch(renderDocument('![Private](file://C:/Users/owner.png)',entry).html, /<img/);
});
