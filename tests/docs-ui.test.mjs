// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import path from 'node:path';
import vm from 'node:vm';
import os from 'node:os';
import { readFile, writeFile, mkdir, mkdtemp, rm, symlink } from 'node:fs/promises';
import * as fixtureFS from 'node:fs/promises';
import * as docsTool from '../scripts/check-docs-ui.mjs';
import { boundedPath, parseOptions, readExportAsset, serve, configureBrowserNetwork, sanitizedRequestURL, assertLandingGeometry, captureLandingActivation } from '../scripts/check-docs-ui.mjs';

test('documentation server confines public assets to its selected export', () => {
  const directory = path.resolve('site/out');
  assert.equal(boundedPath(directory, 'docs/latest/index.html'), path.join(directory, 'docs/latest/index.html'));
  for (const attempt of ['../README.md', '../../private.json', path.resolve('site/outside/file')]) assert.throws(() => boundedPath(directory, attempt), /leaves/);
});

test('browser verification rejects missing, repeated and unsupported arguments', () => {
  assert.equal(parseOptions(['--browser', './chromium', '--output', './build/ui']).browser, path.resolve('chromium'));
  for (const args of [['--browser'], ['--output', '--out'], ['--browser', 'a', '--browser', 'b'], ['--visible', 'true']]) assert.throws(() => parseOptions(args), /Use/);
});

test('documentation assets reject real directory links and junctions before reading or caching outside files', async () => {
  const fixture = await mkdtemp(path.join(os.tmpdir(), 'insonic-docs-assets-'));
  const directory = path.join(fixture, 'export');
  const outside = path.join(fixture, 'outside');
  let served;
  try {
    await mkdir(directory); await mkdir(outside);
    await writeFile(path.join(directory, 'index.html'), 'owned documentation');
    await writeFile(path.join(outside, 'secret.txt'), 'outside bytes must not be read');
    await writeFile(path.join(outside, 'index.html'), 'outside index must not be read');
    await symlink(outside, path.join(directory, 'linked'), process.platform === 'win32' ? 'junction' : 'dir');
    const cache = new Map();
    for (const target of ['linked/secret.txt', 'linked', 'linked/index.html']) {
      await assert.rejects(readExportAsset(directory, target, cache), /links|junctions|export/);
      assert.equal(cache.size, 0, 'rejected outside assets must never enter the cache');
    }
    assert.equal((await readExportAsset(directory, 'index.html', cache)).bytes.toString(), 'owned documentation');
    assert.equal(cache.size, 1);
    served = await serve({ public: directory });
    for (const target of ['linked/secret.txt', 'linked/', 'linked/index.html']) {
      const response = await fetch(`${served.origin}/public/${target}`);
      assert.equal(response.status, 404);
      assert.equal(await response.text(), '');
      assert.equal(served.cache.size, 0, 'HTTP rejection must precede bytes and caching');
    }
    const owned = await fetch(`${served.origin}/public/`);
    assert.equal(owned.status, 200);
    assert.equal(await owned.text(), 'owned documentation');
  } finally {
    if (served) await new Promise(resolve => { served.server.closeAllConnections(); served.server.close(resolve); });
    assert.equal(path.dirname(fixture), os.tmpdir());
    assert.ok(path.basename(fixture).startsWith('insonic-docs-assets-'));
    await rm(fixture, { recursive: true, force: true });
  }
});

test('documentation assets reject a final symbolic link before reading outside bytes', async () => {
  const fixture = await mkdtemp(path.join(os.tmpdir(), 'insonic-docs-filelink-'));
  try {
    const directory = path.join(fixture, 'export'); await mkdir(directory);
    const outside = path.join(fixture, 'outside.txt'); await writeFile(outside, 'outside bytes');
    await symlink(outside, path.join(directory, 'linked.txt'));
    const cache = new Map();
    await assert.rejects(readExportAsset(directory, 'linked.txt', cache), /links|junctions|export/);
    assert.equal(cache.size, 0);
  } finally {
    assert.equal(path.dirname(fixture), os.tmpdir());
    assert.ok(path.basename(fixture).startsWith('insonic-docs-filelink-'));
    await rm(fixture, { recursive: true, force: true });
  }
});

test('WebSocket routing is configured independently and closes every endpoint without forwarding', async () => {
  const calls = [];
  const handlers = {};
  const context = {
    async routeWebSocket(pattern, handler) { calls.push(['websocket', pattern]); handlers.websocket = handler; },
    async route(pattern, handler) { calls.push(['http', pattern]); handlers.http = handler; },
  };
  const attempts = [];
  await configureBrowserNetwork(context, { origin: 'http://127.0.0.1:1234', out: path.resolve('site/out'), offline: path.resolve('site/offline'), externalRequests: [], webSocketAttempts: attempts });
  assert.deepEqual(calls, [['websocket', '**/*'], ['http', '**/*']]);
  for (const endpoint of ['ws://127.0.0.1:1234/endpoint?token=private', 'wss://user:private@example.invalid/endpoint?token=private#private']) {
    let closure;
    await handlers.websocket({ url: () => endpoint, async close(options) { closure = options; }, connectToServer() { assert.fail('the guard must never open a server connection'); } });
    assert.equal(closure.code, 1008);
    assert.equal(attempts.at(-1), sanitizedRequestURL(endpoint));
    assert.doesNotMatch(attempts.at(-1), /private|user:|\?|#/);
  }
  assert.equal(attempts.length, 2);
});

test('responsive native navigation preserves reader choices and focus through width changes', async () => {
  const source = await readFile(new URL('../site/scripts/site-runtime.js', import.meta.url), 'utf8');
  const scoped = source.slice(source.indexOf('for (const disclosure of document.querySelectorAll'));
  const callbacks = {};
  const listeners = {};
  const summary = { focused: false, focus() { this.focused = true; } };
  const child = {};
  const disclosure = { dataset: { responsiveDisclosure: '960' }, open: false, addEventListener(name, callback) { listeners[name] = callback; }, contains(element) { return element === child; }, querySelector() { return summary; } };
  const media = { matches: true, addEventListener(name, callback) { callbacks[name] = callback; } };
  const document = { activeElement: null, querySelectorAll() { return [disclosure]; } };
  vm.runInNewContext(scoped, { document, window: { matchMedia() { return media; } } });
  assert.equal(disclosure.open, true);
  media.matches = false; callbacks.change(); assert.equal(disclosure.open, false);
  disclosure.open = true; listeners.toggle(); // Reader expands the narrow layout.
  media.matches = true; callbacks.change(); assert.equal(disclosure.open, true);
  disclosure.open = false; listeners.toggle(); // Independent wide-layout choice.
  media.matches = false; callbacks.change(); assert.equal(disclosure.open, true);
  document.activeElement = child;
  media.matches = true; callbacks.change();
  assert.equal(disclosure.open, false);
  assert.equal(summary.focused, true, 'collapsing must not strand focus in hidden navigation');
});


test('landing geometry rejects touching controls, clipped labels and undersized targets', () => {
  const control = (label, x, y, width = 100, height = 44) => ({ label, x, y, width, height, text_clipped: false, center_blocked: false });
  const valid = { primary_actions: 1, controls: [control('First',0,0),control('Second',0,56)], groups: [[control('First',0,0),control('Second',0,56)]] };
  assert.doesNotThrow(() => assertLandingGeometry(valid, 'fixture'));
  for (const broken of [
    {...valid, primary_actions: 2},
    {...valid, controls:[control('Small',0,0,20)]},
    {...valid, controls:[{...control('Clipped',0,0),text_clipped:true}]},
    {...valid, controls:[{...control('Obscured',0,0),center_blocked:true}]},
    {...valid, groups:[[control('First',0,0),control('Touching',0,44)]]},
    {...valid, controls:[control('First',0,0),control('Overlapping',20,20)]},
  ]) assert.throws(() => assertLandingGeometry(broken, 'fixture'), /fixture/);
});

test('activation probe cancels only native link navigation and records exact trusted input', () => {
  let handler;
  const recordedWindow = {};
  const document = {addEventListener(name, callback, capture) {assert.equal(name,'click'); assert.equal(capture,true); handler=callback;}};
  vm.runInNewContext('(' + captureLandingActivation.toString() + ')()', {document,window:recordedWindow});
  const anchor = {getAttribute:()=> 'https://github.com/example/package.zip'};
  let prevented=0,stopped=0;
  handler({target:{closest:()=>anchor},isTrusted:true,preventDefault(){prevented++;},stopImmediatePropagation(){stopped++;}});
  assert.equal(prevented,1); assert.equal(stopped,1);
  assert.equal(recordedWindow.__landingActivations[0].href,'https://github.com/example/package.zip');
  assert.equal(recordedWindow.__landingActivations[0].trusted,true);
  handler({target:{closest:()=>null},preventDefault(){assert.fail('non-link blocked');}});
});


test('documentation fixture confines actual profiles and handles marker/profile initialization failure',async t=>{
  const sandbox=await mkdtemp(path.join(os.tmpdir(),'insonic-docs-fixture-'));
  t.after(()=>rm(sandbox,{recursive:true,force:true}));
  const parent=path.join(sandbox,'nested spaces café');await mkdir(parent);
  const owned=await docsTool.createDocumentationFixture({parent});
  assert.equal(path.dirname(owned.directory),await fixtureFS.realpath(parent));
  assert.ok((await fixtureFS.stat(path.join(owned.directory,'home','cache'))).isDirectory());
  await owned.cleanup();await assert.rejects(fixtureFS.lstat(owned.directory),{code:'ENOENT'});assert.equal(owned.evidence.removed,true);
  for(const stage of ['marker','profile']){
    let allocated;
    const filesystem={...fixtureFS,mkdtemp:async prefix=>allocated=await mkdtemp(prefix),
      writeFile:async(...args)=>{if(stage==='marker')throw Object.assign(Error('marker denied'),{code:'EACCES'});return writeFile(...args)},
      mkdir:async(...args)=>{if(stage==='profile')throw Object.assign(Error('profile denied'),{code:'EACCES'});return mkdir(...args)}};
    await assert.rejects(docsTool.createDocumentationFixture({parent,filesystem}),/denied/);
    await assert.rejects(fixtureFS.lstat(allocated),{code:'ENOENT'});
  }
});

test('temporary parent policy rejects Windows roots and aliases without allocating there',async t=>{
  const {validateTemporaryParent}=await import('../scripts/owned-ui-fixture.mjs');
  for(const value of ['C:\\','C:/','C:\\Temp\\..','\\\\server\\share\\','\\\\?\\C:\\','\\\\?\\UNC\\server\\share\\','C:temp','\\temp',''])assert.throws(()=>validateTemporaryParent(value,path.win32),/temporary parent/);
  for(const value of ['C:\\Temp\\nested spaces café','\\\\server\\share\\temp café'])assert.equal(validateTemporaryParent(value,path.win32),path.win32.normalize(value));
  const parent=await mkdtemp(path.join(os.tmpdir(),'insonic-docs-policy-'));t.after(()=>rm(parent,{recursive:true,force:true}));
  const file=path.join(parent,'not-directory');await writeFile(file,'keep');
  let allocations=0;
  for(const candidate of [path.parse(parent).root,file,path.join(parent,'missing')])await assert.rejects(docsTool.createDocumentationFixture({parent:candidate,filesystem:{...fixtureFS,mkdtemp:async()=>{allocations++;throw Error('must not allocate')}}}));
  assert.equal(allocations,0);
  const alias=path.join(parent,'root-alias');await symlink(path.parse(parent).root,alias,process.platform==='win32'?'junction':'dir');
  await assert.rejects(docsTool.createDocumentationFixture({parent:alias}),/temporary parent/);
  const owned=await docsTool.createDocumentationFixture({parent});
  await writeFile(path.join(owned.directory,'docs-ui-owner.json'),'{}');
  await assert.rejects(owned.cleanup(),error=>error.message.includes(owned.directory)&&error.message.includes('ownership'));
});

test('documentation handled export failure removes its fixture and retains cleanup evidence',async t=>{
  const parent=await mkdtemp(path.join(os.tmpdir(),'insonic-docs-handled-'));t.after(()=>rm(parent,{recursive:true,force:true}));
  const output=path.join(parent,'report');
  await assert.rejects(docsTool.checkDocumentationUI({out:path.join(parent,'missing-export'),offline:path.join(parent,'missing-offline'),output,fixtureParent:parent}));
  const report=JSON.parse(await readFile(path.join(output,'report.json'),'utf8'));
  assert.equal(report.cleanup.owned_fixture_removed,true);assert.equal(path.dirname(report.fixture.path),await fixtureFS.realpath(parent));
  await assert.rejects(fixtureFS.lstat(report.fixture.path),{code:'ENOENT'});assert.equal(report.passed,false);
});


test('documentation browser launch failure closes its server and removes the owned profile',async t=>{
  const parent=await mkdtemp(path.join(os.tmpdir(),'insonic-docs-launch-'));t.after(()=>rm(parent,{recursive:true,force:true}));
  const exported=path.join(parent,'export');await mkdir(exported);
  const output=path.join(parent,'report');
  await assert.rejects(docsTool.checkDocumentationUI({out:exported,offline:exported,output,browser:path.join(parent,'missing-browser'),fixtureParent:parent}),/executable|launch/i);
  const report=JSON.parse(await readFile(path.join(output,'report.json'),'utf8'));
  assert.equal(report.cleanup.server_closed,true);assert.equal(report.cleanup.owned_fixture_removed,true);
  await assert.rejects(fixtureFS.lstat(report.fixture.path),{code:'ENOENT'});assert.equal(report.passed,false);
});
