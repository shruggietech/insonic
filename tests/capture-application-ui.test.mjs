// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import path from 'node:path';
import {fixtureEnvironment,parseOptions,allowedAsset,assertReceipt,VERSION,PLAYWRIGHT_VERSION} from '../scripts/capture-application-ui.mjs';

test('capture environment excludes owner profiles and all provider secrets',()=>{
  const env=fixtureEnvironment({SystemRoot:'C:/Windows',USERPROFILE:'C:/Users/Owner',PATH:'private/tool',AWS_SECRET_ACCESS_KEY:'secret',HF_TOKEN:'secret',INSONIC_CAPTURE_NATIVE_PATH:'native/library'},'isolated-fixture','package/insonic.exe');
  assert.equal(env.USERPROFILE,path.join('isolated-fixture','profile'));assert.equal(env.HOME,env.USERPROFILE);
  assert.equal(env.AWS_SECRET_ACCESS_KEY,undefined);assert.equal(env.HF_TOKEN,undefined);assert.ok(!env.PATH.includes('private/tool'));
  for(const key of ['APPDATA','LOCALAPPDATA','XDG_CONFIG_HOME','TEMP','TMP'])assert.ok(env[key].startsWith('isolated-fixture'));
});
test('capture options require explicit current binaries and reject arbitrary commands',()=>{
  assert.throws(()=>parseOptions(['--command','arbitrary']));assert.throws(()=>parseOptions(['--cli','only.exe']));assert.throws(()=>parseOptions(['--cli','one','--cli','two','--bridge','bridge']));
  assert.equal(parseOptions(['--cli','one','--bridge','bridge']).cli,path.resolve('one'));
});
test('assets are an exact inventory rather than arbitrary file paths',()=>{
  const files=new Map([['/assets/bundle/app.js',Buffer.from('app')]]);assert.equal(allowedAsset('/assets/bundle/app.js',files),true);
  for(const url of ['/assets/../../profile.json','/assets/bundle/app.js?secret=1','file:///owner/profile','/bridge'])assert.equal(allowedAsset(url,files),false);
});
test('receipt rejects false renderer, private fixtures, missing landmarks and image hashes',()=>{
  const receipt={kind:'application-ui-capture',schema_version:VERSION,renderer:{kind:'chromium-headless',playwright_version:PLAYWRIGHT_VERSION},fixture:{isolated:true,private_inputs:false},source:{revision:'a'.repeat(40),tree_sha256:'b'.repeat(64)},images:['setup-location.png','setup-backends.png','setup-processing.png','library-ready.png'].map(name=>({path:name,sha256:'c'.repeat(64),width:1440,height:1100,landmark:'Setup',caption:'Figure. Actual controls.'}))};
  assert.equal(assertReceipt(receipt),receipt);
  for(const mutate of [r=>r.renderer.kind='native-webview',r=>r.fixture.private_inputs=true,r=>r.images[0].sha256='wrong',r=>r.images[0].landmark='',r=>r.source.tree_sha256='',r=>r.images[0].path='../owner.png',r=>r.images[0].height=6001]){const changed=structuredClone(receipt);mutate(changed);assert.throws(()=>assertReceipt(changed))}
});
