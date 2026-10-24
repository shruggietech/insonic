// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import crypto from 'node:crypto';
import { execFileSync } from 'node:child_process';
import { milliseconds } from '../../scripts/qualification-timing.mjs';
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

test('frozen captures verify exact ancestral source while later implementation changes', async t => {
  const root = await fs.mkdtemp(path.join(os.tmpdir(),'insonic-capture-history-'));
  t.after(() => fs.rm(root,{recursive:true,force:true}));
  const git = args => execFileSync('git',args,{cwd:root,encoding:'utf8',windowsHide:true,timeout:milliseconds('git_seconds'),stdio:['ignore','pipe','pipe']}).trim();
  git(['init','--quiet']);
  await fs.writeFile(path.join(root,'VERSION'),'1.0.1\n');
  const implementation=Buffer.from('captured implementation');
  await fs.writeFile(path.join(root,'implementation.txt'),implementation);
  git(['add','VERSION','implementation.txt']);
  git(['-c','user.name=Fixture','-c','user.email=fixture@example.test','-c','commit.gpgsign=false','commit','--quiet','-m','captured source']);
  const revision=git(['rev-parse','HEAD']);
  const source=path.join(root,'docs/screenshots/v1.0.1');await fs.mkdir(source,{recursive:true});
  const png=Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+a2uoAAAAASUVORK5CYII=','base64');
  await fs.writeFile(path.join(source,'fixture.png'),png);
  const receipt={schema_version:'1.0.1',source:{revision,dirty:false,files:[{path:'implementation.txt',sha256:digest(implementation)}]},renderer:{kind:'chromium-headless'},fixture:{isolated:true,private_inputs:false},locale:'en-US',images:[{path:'fixture.png',sha256:digest(png),width:1,height:1,landmark:'Fixture',caption:'Captured release surface.'}]};
  const save=()=>fs.writeFile(path.join(source,'application-capture.json'),JSON.stringify(receipt));await save();
  await fs.writeFile(path.join(root,'implementation.txt'),'later backend implementation');
  const out=path.join(root,'out');await prepareScreenshots(root,out);
  assert.deepEqual(await fs.readFile(path.join(out,'screenshots/v1.0.1/fixture.png')),png);
  receipt.source.files[0].sha256='f'.repeat(64);await save();
  await assert.rejects(prepareScreenshots(root,out),/source differs/);
  receipt.source.files[0].sha256=digest(implementation);receipt.source.revision='f'.repeat(40);await save();
  await prepareScreenshots(root,out);
  assert.equal(JSON.parse(await fs.readFile(path.join(out,'screenshots/v1.0.1/application-capture.json'),'utf8')).source.revision,'f'.repeat(40));
  await fs.writeFile(path.join(root,'VERSION'),'1.0.2\n');git(['add','VERSION']);
  git(['-c','user.name=Fixture','-c','user.email=fixture@example.test','-c','commit.gpgsign=false','commit','--quiet','-m','another version']);
  receipt.source.revision=git(['rev-parse','HEAD']);await save();
  await assert.rejects(prepareScreenshots(root,out),/captured version/);
});

test('squashed CI capture sources require one coherent historical version and unchanged receipts', async t => {
  const root=await fs.mkdtemp(path.join(os.tmpdir(),'insonic-capture-squash-'));
  assert.equal(path.dirname(root),path.resolve(os.tmpdir()));
  t.after(()=>fs.rm(root,{recursive:true,force:true}));
  const git=args=>execFileSync('git',args,{cwd:root,encoding:'utf8',windowsHide:true,timeout:milliseconds('git_seconds'),stdio:['ignore','pipe','pipe']}).trim();
  const commit=message=>git(['-c','user.name=Fixture','-c','user.email=fixture@example.test','-c','commit.gpgsign=false','commit','--quiet','-am',message]);
  git(['init','--quiet']);
  await fs.writeFile(path.join(root,'VERSION'),'1.0.1\n');
  await fs.writeFile(path.join(root,'implementation.txt'),'before');
  await fs.writeFile(path.join(root,'secondary.txt'),'before secondary');
  git(['add','VERSION','implementation.txt','secondary.txt']);commit('initial');
  const main=git(['branch','--show-current']);git(['checkout','--quiet','-b','ci-capture']);
  const implementation=Buffer.from('captured implementation'),secondary=Buffer.from('captured secondary');
  await fs.writeFile(path.join(root,'implementation.txt'),implementation);await fs.writeFile(path.join(root,'secondary.txt'),secondary);commit('captured');
  const revision=git(['rev-parse','HEAD']);git(['checkout','--quiet',main]);git(['merge','--squash','ci-capture']);commit('owner squash');
  assert.throws(()=>git(['merge-base','--is-ancestor',revision,'HEAD']));
  await fs.writeFile(path.join(root,'implementation.txt'),'later implementation');await fs.writeFile(path.join(root,'secondary.txt'),'later secondary');commit('later');
  const directory=path.join(root,'docs/screenshots/v1.0.1');await fs.mkdir(directory,{recursive:true});
  const png=Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+a2uoAAAAASUVORK5CYII=','base64');await fs.writeFile(path.join(directory,'fixture.png'),png);
  const receipt={schema_version:'1.0.1',source:{revision,dirty:false,files:[{path:'implementation.txt',sha256:digest(implementation)},{path:'secondary.txt',sha256:digest(secondary)}]},renderer:{kind:'native-installer'},fixture:{isolated:true,private_inputs:false},locale:'en-US',images:[{path:'fixture.png',sha256:digest(png),width:1,height:1,landmark:'Fixture',caption:'Actual capture.'}]};
  const filename=path.join(directory,'installer-capture.json'),save=()=>fs.writeFile(filename,JSON.stringify(receipt));await save();
  const out=path.join(root,'out');await prepareScreenshots(root,out);
  assert.deepEqual(await fs.readFile(path.join(out,'screenshots/v1.0.1/installer-capture.json')),await fs.readFile(filename));
  receipt.source.files[1].sha256=digest(Buffer.from('before secondary'));await save();
  await assert.rejects(prepareScreenshots(root,out),/matching complete snapshot/);
});


test('squashed captures find coherent ancestors whose anchor was unchanged, beyond four matching anchors', async t => {
  const root=await fs.mkdtemp(path.join(os.tmpdir(),'insonic-capture-stable-anchor-'));
  t.after(()=>fs.rm(root,{recursive:true,force:true}));
  const git=args=>execFileSync('git',args,{cwd:root,encoding:'utf8',windowsHide:true,timeout:milliseconds('git_seconds'),stdio:['ignore','pipe','pipe']}).trim();
  const commit=message=>git(['-c','user.name=Fixture','-c','user.email=fixture@example.test','-c','commit.gpgsign=false','commit','--quiet','-am',message]);
  git(['init','--quiet']);
  const implementation=Buffer.from('captured implementation'),secondary=Buffer.from('captured secondary');
  await fs.writeFile(path.join(root,'VERSION'),'1.0.1\n');
  await fs.writeFile(path.join(root,'implementation.txt'),implementation);
  await fs.writeFile(path.join(root,'secondary.txt'),'before secondary');
  git(['add','VERSION','implementation.txt','secondary.txt']);commit('initial anchor');
  const main=git(['branch','--show-current']);git(['checkout','--quiet','-b','ci-capture']);
  // The later complete snapshot changes only the secondary file. A log limited
  // to implementation.txt would return the older, incomplete initial snapshot.
  await fs.writeFile(path.join(root,'secondary.txt'),secondary);commit('captured secondary');
  const revision=git(['rev-parse','HEAD']);
  git(['checkout','--quiet',main]);git(['merge','--squash','ci-capture']);commit('owner squash');
  assert.throws(()=>git(['merge-base','--is-ancestor',revision,'HEAD']));
  // All five newer ancestors retain the anchor, but none match the complete
  // inventory. Candidate truncation would discard the valid owner squash.
  for(let i=0;i<5;i++){
    await fs.writeFile(path.join(root,'secondary.txt'),'later secondary '+i);commit('secondary update '+i);
  }
  await fs.writeFile(path.join(root,'implementation.txt'),'later implementation');commit('later anchor');
  const directory=path.join(root,'docs/screenshots/v1.0.1');await fs.mkdir(directory,{recursive:true});
  const png=Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+a2uoAAAAASUVORK5CYII=','base64');
  await fs.writeFile(path.join(directory,'fixture.png'),png);
  const receipt={schema_version:'1.0.1',source:{revision,dirty:false,files:[{path:'implementation.txt',sha256:digest(implementation)},{path:'secondary.txt',sha256:digest(secondary)}]},renderer:{kind:'chromium-headless'},fixture:{isolated:true,private_inputs:false},locale:'en-US',images:[{path:'fixture.png',sha256:digest(png),width:1,height:1,landmark:'Fixture',caption:'Original captured surface.'}]};
  const filename=path.join(directory,'application-capture.json');
  const original=Buffer.from(JSON.stringify(receipt));await fs.writeFile(filename,original);
  const out=path.join(root,'out');await prepareScreenshots(root,out);
  assert.deepEqual(await fs.readFile(path.join(out,'screenshots/v1.0.1/application-capture.json')),original);
  assert.deepEqual(await fs.readFile(path.join(out,'screenshots/v1.0.1/fixture.png')),png);
  // Mixed historical bytes never become an acceptable coherent snapshot.
  receipt.source.files[1].sha256=digest(Buffer.from('never committed secondary'));
  await fs.writeFile(filename,JSON.stringify(receipt));
  await assert.rejects(prepareScreenshots(root,out),/matching complete snapshot/);
});
