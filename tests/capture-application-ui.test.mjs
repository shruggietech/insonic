// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import path from 'node:path';
import os from 'node:os';
import {mkdtemp,mkdir,readFile,writeFile,rm} from 'node:fs/promises';
import {spawn} from 'node:child_process';
import {createHash} from 'node:crypto';
import {fixtureEnvironment,parseOptions,allowedAsset,assertReceipt,boundedOperation,sourceSnapshot,embeddedSourceIdentity,assertBinarySource,VERSION,PLAYWRIGHT_VERSION} from '../scripts/capture-application-ui.mjs';

test('capture renderer authority follows the exact root package and lock pins',async()=>{
  const manifest=JSON.parse(await readFile(new URL('../package.json',import.meta.url),'utf8'));
  const lock=JSON.parse(await readFile(new URL('../package-lock.json',import.meta.url),'utf8'));
  assert.match(PLAYWRIGHT_VERSION,/^\d+\.\d+\.\d+$/);
  assert.equal(PLAYWRIGHT_VERSION,manifest.devDependencies['playwright-core']);
  assert.equal(PLAYWRIGHT_VERSION,lock.packages[''].devDependencies['playwright-core']);
  assert.equal(PLAYWRIGHT_VERSION,lock.packages['node_modules/playwright-core'].version);
});

test('real Git source inventory marks untracked compiled inputs dirty while ignoring build output',async t=>{
  const temporary=path.resolve(os.tmpdir());const fixture=await mkdtemp(path.join(temporary,'insonic-capture-git-'));
  t.after(async()=>{assert.equal(path.dirname(fixture),temporary);assert.ok(path.basename(fixture).startsWith('insonic-capture-git-'));await rm(fixture,{recursive:true,force:true})});
  const git=async args=>new Promise((resolve,reject)=>{const child=spawn('git',args,{cwd:fixture,windowsHide:true,stdio:['pipe','pipe','pipe'],env:{...process.env,GIT_CONFIG_NOSYSTEM:'1',GIT_CONFIG_GLOBAL:path.join(fixture,'empty-config'),GIT_TERMINAL_PROMPT:'0'}});let message='';child.stdout.on('data',chunk=>message+=chunk);child.stderr.on('data',chunk=>message+=chunk);child.once('error',reject);child.once('exit',code=>code===0?resolve():reject(Error(message)));child.stdin.end()});
  await mkdir(path.join(fixture,'cmd','insonic'),{recursive:true});
  await writeFile(path.join(fixture,'cmd','insonic','main.go'),'package main\nfunc main() {}\n');
  await writeFile(path.join(fixture,'.gitignore'),'build/\nempty-config\n');await writeFile(path.join(fixture,'empty-config'),'');
  await git(['init','--quiet']);await git(['config','core.excludesFile',path.join(fixture,'empty-config')]);await git(['config','core.hooksPath',path.join(fixture,'no-hooks')]);
  await git(['add','.gitignore','cmd/insonic/main.go']);await git(['-c','user.name=Capture fixture','-c','user.email=capture@example.invalid','commit','--quiet','--no-gpg-sign','-m','Capture fixture']);
  const clean=await sourceSnapshot(fixture);assert.equal(clean.dirty,false);
  await mkdir(path.join(fixture,'build'));await writeFile(path.join(fixture,'build','capture.png'),'ignored generated capture');
  const ignored=await sourceSnapshot(fixture);assert.equal(ignored.dirty,false);assert.equal(ignored.tree_sha256,clean.tree_sha256);
  const added='package main\nvar captureFixtureExtra = 1\n';await writeFile(path.join(fixture,'cmd','insonic','extra.go'),added);
  const untracked=await sourceSnapshot(fixture);assert.equal(untracked.dirty,true);assert.notEqual(untracked.tree_sha256,clean.tree_sha256);
  assert.deepEqual(untracked.files.find(file=>file.path==='cmd/insonic/extra.go'),{path:'cmd/insonic/extra.go',sha256:createHash('sha256').update(added).digest('hex')});
});

test('capture requires an exact clean Go binary source identity',()=>{
  const revision='a'.repeat(40),build=modified=>Buffer.from('build\tvcs.revision='+revision+'\nbuild\tvcs.modified='+modified+'\n');
  assert.deepEqual(embeddedSourceIdentity(build('false')),{revision,modified:false});assert.equal(assertBinarySource(build('false'),revision).modified,false);
  assert.throws(()=>assertBinarySource(build('true'),revision),/clean source/);assert.throws(()=>assertBinarySource(build('false'),'b'.repeat(40)),/does not match/);
  assert.throws(()=>embeddedSourceIdentity(Buffer.from('build\tvcs.revision='+revision+'\n')),/missing or malformed/);assert.throws(()=>embeddedSourceIdentity(build('falsehood')),/missing or malformed/);
});

test('capture environment excludes owner profiles and all provider secrets',()=>{
  const env=fixtureEnvironment({SystemRoot:'C:/Windows',USERPROFILE:'C:/Users/Owner',PATH:'private/tool',AWS_SECRET_ACCESS_KEY:'secret',HF_TOKEN:'secret',INSONIC_CAPTURE_NATIVE_PATH:'native/library'},'isolated-fixture','package/insonic.exe');
  assert.equal(env.USERPROFILE,path.join('isolated-fixture','profile'));assert.equal(env.HOME,env.USERPROFILE);
  assert.equal(env.AWS_SECRET_ACCESS_KEY,undefined);assert.equal(env.HF_TOKEN,undefined);assert.ok(!env.PATH.includes('private/tool'));
  for(const key of ['APPDATA','LOCALAPPDATA','XDG_CONFIG_HOME','TEMP','TMP'])assert.ok(env[key].startsWith('isolated-fixture'));
  assert.equal(env.LOCALAPPDATA,path.join(env.USERPROFILE,'AppData','Local'));assert.equal(env.APPDATA,path.join(env.USERPROFILE,'AppData','Roaming'));
});
test('an unresponsive browser operation has a classified phase deadline',async()=>{
  await assert.rejects(boundedOperation('browser.test',()=>new Promise(()=>{}),10),error=>error.code==='capture_timeout'&&error.phase==='browser.test');
  assert.equal(await boundedOperation('browser.completed',()=>Promise.resolve('result'),10),'result');
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
  const receipt={kind:'application-ui-capture',schema_version:VERSION,renderer:{kind:'chromium-headless',playwright_version:PLAYWRIGHT_VERSION},fixture:{isolated:true,browser_profile_owned:true,private_inputs:false},source:{revision:'a'.repeat(40),tree_sha256:'b'.repeat(64),dirty:false},images:['setup-location.png','setup-backends.png','setup-processing.png','library-ready.png'].map(name=>({path:name,sha256:'c'.repeat(64),width:1440,height:1100,landmark:'Setup',caption:'Figure. Actual controls.'}))};
  assert.equal(assertReceipt(receipt),receipt);
  for(const mutate of [r=>r.renderer.kind='native-webview',r=>r.renderer.playwright_version='0.0.0',r=>r.fixture.private_inputs=true,r=>r.fixture.browser_profile_owned=false,r=>r.images[0].sha256='wrong',r=>r.images[0].landmark='',r=>r.source.tree_sha256='',r=>r.source.dirty=true,r=>delete r.source.dirty,r=>r.images[0].path='../owner.png',r=>r.images[0].height=6001]){const changed=structuredClone(receipt);mutate(changed);assert.throws(()=>assertReceipt(changed))}
});
