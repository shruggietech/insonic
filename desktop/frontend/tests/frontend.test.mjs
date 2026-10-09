// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import { JSDOM } from 'jsdom';
import { readFileSync } from 'node:fs';
import { validateNativeRequest } from './contracts.mjs';
const dom = new JSDOM(
  '<!doctype html><html><body><div id="root"></div></body></html>',
  { url: 'http://wails.localhost' },
);
for (const key of [
  'window',
  'document',
  'HTMLElement',
  'HTMLInputElement',
  'HTMLTextAreaElement',
  'HTMLSelectElement',
  'Event',
  'MouseEvent',
])
  globalThis[key] = dom.window[key];
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
window.matchMedia = () => ({
  matches: false,
  addEventListener() {},
  removeEventListener() {},
});
const { act, createElement } = await import('react');
const { createRoot } = await import('react-dom/client');
const { App } = await import('../.test-build/App.js');
const { appendCapture, Client, StaleResponse, rationalSeconds } =
  await import('../.test-build/client.js');
const { pipelinePayload, speakerPayload, termPayload, uuid } =
  await import('../.test-build/forms.js');
const { runQualificationOnce, mediaDiagnostics } =
  await import('../.test-build/qualification.js');
const wid = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',
  mid = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb';
const tick = async () => {
  for (let i = 0; i < 3; i++)
    await act(() => new Promise((resolve) => setTimeout(resolve, 0)));
};
let root;
function mock() {
  const calls = [],
    terms = [],
    pipelines = [],
    speakers = [];
  const entry = {
    id: mid,
    revision: 1,
    title: 'Committed speech',
    mode:'reference',
    class: 'audio',
    digest: 'digest',
    dates: { precision: 'date-only', date: '1961-01-20' },
    availability: 'available',
  };
  const bridge = {
    calls,
    Show: async () => ({
      workspace_id: wid,
      result: { display_name: 'Fixture library' },
    }),
    SelectWorkspace: async (path, name, create) => {
      calls.push({ operation: 'select', path, name, create });
      return { result: { workspace_id: wid, display_name: name } };
    },
    Choose: async () => '/fixture.wav',
    Playback: async (...args) => {
      calls.push({ operation: 'Playback', args });
      return { result: { url: '/media/opaque', mime_type: 'audio/wav' } };
    },
    ClosePlayback: async () => ({ result: {} }),
    VerifyPlayback: async () => ({ result: { valid: true } }),
    Credential: async (args, input) => {
      calls.push({ operation: 'Credential', args, input });
      return { result: { state: 'configured' } };
    },
    CompleteSmoke: async () => {},
    Operate: async (request) => {
      const validation = validateNativeRequest(request);
      assert.equal(
        validation.valid,
        true,
        `${request.operation}: ${validation.errors}`,
      );
      calls.push(request);
      let result = { items: [], next_id: '' };
      if (request.operation === 'media.list') result.items = [entry];
      if (request.operation === 'media.show') result = entry;
      if(request.operation==='recordings.roster.show')result={recording_id:mid,declared:false,revision:0,members:[]};
      if (request.operation === 'media.metadata')
        result = {
          observations: [
            { family: 'container', tag: 'date', raw: '1961-01-20' },
          ],
        };
      if (request.operation === 'recordings.cues')
        result = {
          recording_id: mid,
          revision: 3,
          document_digest: '5'.repeat(64),
          voices: [{ id: mid, name: 'Voice 1' }],
          cues: [
            {
              ordinal: 0,
              id: 'cue1',
              text: 'Historical speech',
              timing: { start_milliseconds: '1250', end_milliseconds: '2500' },
              seek_seconds: 1.25,
              speaker_attributions: [],
            },
          ],
          next_ordinal: null,
        };
      if (request.operation === 'pipelines.list') result.items = pipelines;
      if (request.operation === 'pipelines.set') {
        const p = { ...request.data.pipeline, revision: 1 };
        pipelines.push(p);
        result = p;
      }
      if (request.operation === 'pipelines.show')
        result = pipelines.find((p) => p.id === request.item_id);
      if (request.operation === 'pipelines.inspect')
        result = { reachability: 'not-probed' };
      if (request.operation === 'speakers.list')
        result.items = speakers.map((speaker) => ({
          speaker,
          aliases:
            calls.find(
              (c) => c.operation === 'speakers.set' && c.item_id === speaker.id,
            )?.data.aliases ?? [],
        }));
      if (request.operation === 'terms.list') result.items = terms;
      if (request.operation === 'terms.set') {
        const term = { ...request.data.term, revision: 1 };
        terms.push(term);
        result = term;
      }
      if (request.operation === 'terms.show')
        result = terms.find((t) => t.id === request.item_id);
      if (request.operation === 'speakers.set') {
        const speaker = { ...request.data.speaker, revision: 1 };
        speakers.push(speaker);
        result = { speaker, aliases: request.data.aliases };
      }
      if (request.operation === 'speakers.show') {
        const s = calls.find(
          (c) =>
            c.operation === 'speakers.set' && c.item_id === request.item_id,
        );
        result = {
          speaker: { ...s.data.speaker, revision: 1 },
          aliases: s.data.aliases,
        };
      }
      if (request.operation === 'settings.show')
        result = {
          sections: {
            media_tools: {
              revision: '1'.repeat(64),
              value: {},
              origin: 'package',
            },
            processing_tools: {
              revision: '2'.repeat(64),
              value: {},
              origin: 'package',
            },
            appearance: {
              revision: '3'.repeat(64),
              value: { theme: 'system', reduced_motion: false },
              origin: 'default',
            },
          },
          profiles: {
            storage: { adapter: 'filesystem' },
            catalog: { adapter: 'sqlite' },
            graph: { adapter: 'ladybug' },
          },
          credentials: { backend: 'native' },
        };
      if (request.operation === 'work.list')
        result.items = [
          {
            id: mid,
            kind: 'media.import',
            state: 'failed',
            phase: 'metadata',
            error: 'unavailable',
          },
        ];
      if (request.operation === 'work.show')
        result = { id: mid, state: 'failed', phase: 'metadata', generation: 1 };
      return { workspace_id: wid, result };
    },
  };
  return bridge;
}
async function mount(bridge) {
  document.body.innerHTML = '<div id="root"></div>';
  root = createRoot(document.getElementById('root'));
  await act(async () => root.render(createElement(App, { bridge })));
  await tick();
}
async function unmount() {
  await act(async () => root.unmount());
}
function button(text) {
  const found = [...document.querySelectorAll('button')].find(
    (b) => b.textContent === text,
  );
  assert.ok(found, `Missing button ${text}`);
  return found;
}
async function click(text) {
  await act(async () => button(text).click());
  await tick();
}
async function fill(label, value) {
  const l = [...document.querySelectorAll('label')].find(
    (l) => l.textContent === label,
  );
  assert.ok(l, `Missing field ${label}`);
  const control = document.getElementById(l.htmlFor);
  await act(async () => {
    const proto =
      control.tagName === 'SELECT'
        ? HTMLSelectElement.prototype
        : control.tagName === 'TEXTAREA'
          ? HTMLTextAreaElement.prototype
          : HTMLInputElement.prototype;
    Object.getOwnPropertyDescriptor(proto, 'value').set.call(control, value);
    control.dispatchEvent(new Event('input', { bubbles: true }));
    control.dispatchEvent(new Event('change', { bubbles: true }));
  });
}
function field(label) {
  const found = [...document.querySelectorAll('label')].find(
    (l) => l.textContent === label,
  );
  assert.ok(found, `Missing field ${label}`);
  return document.getElementById(found.htmlFor);
}
const mediaToolFixture = (directory) => ({
  kind: 'media-tools',
  schema_version: '0.0.0',
  ffprobe: {
    path: `${directory}/ffprobe.exe`,
    sha256: '7'.repeat(64),
    version: '1',
  },
  exiftool: {
    path: `${directory}/exiftool.exe`,
    sha256: '8'.repeat(64),
    version: '1',
  },
});
test('native HTML loads the standalone IIFE as a deferred classic script', () => {
  const html = readFileSync(
    new URL('../../index.html', import.meta.url),
    'utf8',
  );
  const page = new JSDOM(html);
  const script = page.window.document.querySelector(
    'script[src="/assets/bundle/app.js"]',
  );
  assert.ok(script);
  assert.equal(script.type, '');
  assert.equal(script.defer, true);
  const bundle = readFileSync(
    new URL('../../assets/bundle/app.js', import.meta.url),
    'utf8',
  );
  assert.match(bundle, /^"use strict";\(\(\)=>\{/);
});
test('fixture readiness and DOM event start exactly one qualification journey', async () => {
  let runs = 0,
    finish;
  const start = runQualificationOnce(async () => {
    runs++;
    await new Promise((resolve) => {
      finish = resolve;
    });
  });
  const fixturesReady = start(),
    domReady = start();
  assert.equal(fixturesReady, domReady);
  await Promise.resolve();
  assert.equal(runs, 1);
  finish();
  await Promise.all([fixturesReady, domReady]);
  await start();
  assert.equal(runs, 1);
});
test('qualification media diagnostics distinguish transport and decode states without URLs or browser error text', () => {
  const diagnostic = mediaDiagnostics({
    readyState: 0,
    networkState: 3,
    duration: NaN,
    buffered: { length: 0 },
    error: { code: 4, message: 'Forbidden private source URL /do-not-log' },
    currentSrc: 'wails://private-source/do-not-log',
  });
  assert.equal(
    diagnostic,
    'readyState=0,networkState=3,error=4(unsupported-source),buffered=0,duration=unavailable',
  );
  assert.equal(diagnostic.includes('do-not-log'), false);
  assert.match(
    mediaDiagnostics({
      readyState: 1,
      networkState: 2,
      duration: 3,
      buffered: { length: 1 },
      error: { code: 3 },
    }),
    /error=3\(decode\).*duration=positive/,
  );
  assert.equal(mediaDiagnostics(null), 'media-element=missing');
});
test('opaque native contexts create RFC 4122 v4 identities from cryptographic random bytes', () => {
  const descriptors = Object.getOwnPropertyDescriptors(crypto);
  let calls = 0;
  Object.defineProperty(crypto, 'randomUUID', {
    configurable: true,
    value: undefined,
  });
  Object.defineProperty(crypto, 'getRandomValues', {
    configurable: true,
    value: (bytes) => {
      calls++;
      assert.ok(bytes instanceof Uint8Array);
      assert.equal(bytes.length, 16);
      return bytes.fill(0xff);
    },
  });
  try {
    assert.equal(uuid(), 'ffffffff-ffff-4fff-bfff-ffffffffffff');
    assert.equal(calls, 1);
  } finally {
    for (const key of ['randomUUID', 'getRandomValues']) {
      if (descriptors[key])
        Object.defineProperty(crypto, key, descriptors[key]);
      else delete crypto[key];
    }
  }
});
test('workspace and selection generations reject delayed responses', async () => {
  let resolve;
  const client = new Client({
    Operate: () =>
      new Promise((r) => {
        resolve = r;
      }),
  });
  client.select(wid);
  const pending = client.call('media.show', mid);
  client.select(mid);
  resolve({ workspace_id: wid, result: {} });
  await assert.rejects(pending, StaleResponse);
  client.select(wid);
  const p = client.call('media.show', mid);
  client.invalidate();
  resolve({ workspace_id: wid, result: {} });
  await assert.rejects(p, StaleResponse);
});
test('shared failure envelope never appears as successful state', async () => {
  const client = new Client({
    Operate: async () => ({
      workspace_id: wid,
      error: { code: 'conflict', message: 'Refresh revision' },
    }),
  });
  client.select(wid);
  await assert.rejects(
    client.call('terms.set', mid, {}),
    (e) => e.code === 'conflict',
  );
});
test('payloads preserve CAS and discard fields incompatible with changed route', () => {
  const previous = {
    id: mid,
    revision: 7,
    configuration: {
      recognition: {
        mode: 'hosted',
        endpoint: 'https://old.example',
        credential_id: wid,
      },
      quality: {},
    },
  };
  const values = {
    name: 'Local',
    preset: 'local',
    recognition_mode: 'local',
    diarization_mode: 'local',
    recognition_model: mid,
    diarization_model: mid,
    recognition_device: 'cpu',
    diarization_device: 'cpu',
    language: 'en',
    hints: '',
    quality: false,
  };
  const payload = pipelinePayload(values, previous);
  assert.equal(payload.expected_revision, 7);
  assert.equal(payload.pipeline.configuration.recognition.endpoint, undefined);
  assert.equal(payload.pipeline.configuration.recognition.model_id, mid);
  const term = termPayload(
    {
      canonical: 'Test',
      variants: '',
      language: '',
      context: '',
      state: 'active',
      speaker_id: '',
    },
    { id: mid, revision: 2 },
  );
  assert.equal(term.expected_revision, 2);
  assert.deepEqual(term.term.variants, []);
});
test('rational seek conversion never manufactures timing for untimed or invalid evidence', () => {
  assert.equal(rationalSeconds('1250/1000'), 1.25);
  assert.equal(rationalSeconds('1/0'), null);
  assert.equal(rationalSeconds('-1'), null);
  assert.equal(rationalSeconds('9007199254740992'), null);
});
test('raw capture chunks retain exact nanoseconds and split UTF-8 without JSON number parsing', () => {
  const text = '{"name":"café","unix_ns":1791405973347928400}';
  const bytes = new TextEncoder().encode(text);
  const split = bytes.indexOf(0xc3) + 1;
  const page = (start, end) => ({
    encoding: 'base64',
    data: Buffer.from(bytes.slice(start, end)).toString('base64'),
    offset: start,
    next_offset: end,
    sha256: 'exactdigest',
  });
  const first = appendCapture(page(0, split));
  const final = appendCapture(page(split, bytes.length), first);
  assert.equal(final.text, text);
  assert.throws(
    () =>
      appendCapture(
        { ...page(split, bytes.length), sha256: 'replacement' },
        first,
      ),
    /capture changed/,
  );
});
test('speaker name edits preserve heterogeneous alias metadata and provenance', () => {
  const prior = {
    speaker: { id: mid, revision: 3 },
    aliases: [
      {
        id: wid,
        text: 'Alias',
        speaker_id: mid,
        language: 'fr',
        scope: 'archive',
        state: 'inactive',
        provenance: { basis: 'captured' },
      },
    ],
  };
  const payload = speakerPayload(
    {
      name: 'Renamed',
      aliases: 'Alias',
      language: '',
      scope: '',
      state: 'active',
    },
    prior,
  );
  assert.deepEqual(payload.aliases, prior.aliases);
});
test('rendered library detail/date/relocation/seek actions use original current authority', async () => {
  const bridge = mock();
  await mount(bridge);
  await click('Open Committed speech');
  assert.match(document.body.textContent, /Historical speech/);
  await click('Seek 1250 ms');
  const call = bridge.calls.find((c) => c.operation === 'Playback');
  assert.deepEqual(call.args, [mid, 1, 3, '5'.repeat(64)]);
  const audio = document.querySelector('audio');
  assert.ok(audio);
  await act(async () =>
    audio.dispatchEvent(new Event('loadedmetadata', { bubbles: true })),
  );
  assert.equal(audio.currentTime, 1.25);
  await fill('Origination date', '1961-01-20');
  await click('Save date correction');
  assert.deepEqual(
    bridge.calls.find((c) => c.operation === 'media.set-origin').data,
    { revision: 1, options: { originated_on: '1961-01-20' } },
  );
  await fill('New original path', '/moved.wav');
  await click('Verify identity and reconnect');
  assert.deepEqual(
    bridge.calls.find((c) => c.operation === 'media.relocate').data,
    { revision: 1, path: '/moved.wav' },
  );
  await unmount();
});
test('original playback for a fresh native entry passes no current-recording fence', async () => {
  const bridge = mock(),
    original = bridge.Operate;
  bridge.Operate = async (request) => {
    const response = await original(request);
    if (request.operation === 'media.show')
      response.result = {
        id: mid,
        asset_id: wid,
        title: 'Committed speech',
    mode:'reference',
        class: 'audio',
        mode: 'reference',
        source_locator: '/fixture.wav',
        digest: '7'.repeat(64),
        size: 12500,
        original_publication_id: null,
        subtitle_publication_id: wid,
        duration_us: 1000000,
        facts: { streams: [] },
        metadata: { reports: [] },
        dates: { precision: 'date-only', date: '1961-01-20' },
        report_publication_ids: [],
        revision: 7,
        availability: 'available',
      };
    if (request.operation === 'recordings.cues')
      return {
        workspace_id: wid,
        error: { code: 'not_found', message: 'Current recording unavailable.' },
      };
    return response;
  };
  await mount(bridge);
  await click('Open Committed speech');
  assert.equal(document.querySelector('[role=alert]'), null);
  await click('Play recording audio');
  assert.deepEqual(bridge.calls.find((c) => c.operation === 'Playback').args, [
    mid,
    7,
    0,
    '',
  ]);
  assert.ok(document.querySelector('audio'));
  await unmount();
});
test('rendered import includes the published manifest discriminator and schema version', async () => {
  const bridge = mock();
  await mount(bridge);
  await fill('Media path or URL', '/fixture.wav');
  await fill('Title', 'Contract fixture');
  await click('Import');
  const request = bridge.calls.find((c) => c.operation === 'media.import');
  assert.equal(request.data.kind, 'import-manifest');
  assert.equal(request.data.schema_version, '0.0.0');
  assert.equal(validateNativeRequest(request).valid, true);
  const { kind, ...missingKind } = request.data;
  assert.equal(
    validateNativeRequest({ ...request, data: missingKind }).valid,
    false,
  );
  const { schema_version, ...missingVersion } = request.data;
  assert.equal(
    validateNativeRequest({ ...request, data: missingVersion }).valid,
    false,
  );
  await unmount();
});
test('rendered assembly and export requests satisfy the published shared envelope', async () => {
  const bridge = mock();
  await mount(bridge);
  await click('Open Committed speech');
  await fill('Turn start milliseconds', '2500');
  await fill('Turn end milliseconds', '2600');
  await click('Add turn');
  await click('Assemble current recording');
  const assembly = bridge.calls.find(
    (c) => c.operation === 'recordings.assemble',
  );
  assert.deepEqual(assembly.data.turns, [
    { label: 'Voice 1', start_us: 2500000, end_us: 2600000 },
  ]);
  assert.equal(validateNativeRequest(assembly).valid, true);
  await fill('Export destination', '/fixture.srt');
  await click('Export subtitles');
  assert.equal(
    validateNativeRequest(
      bridge.calls.find((c) => c.operation === 'recordings.export'),
    ).valid,
    true,
  );
  await unmount();
});
test('rendered direct processing carries named model choices and only published elections', async () => {
  const bridge = mock(),
    original = bridge.Operate,
    recognition = 'cccccccc-cccc-4ccc-8ccc-cccccccccccc',
    diarization = 'eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee';
  bridge.Operate = async (request) => {
    if (request.operation === 'models.list') {
      const validation = validateNativeRequest(request);
      assert.equal(validation.valid, true, validation.errors);
      return {
        workspace_id: wid,
        result: {
          items: [
            {
              id: recognition,
              name: 'Local speech model',
              model_version: '1',
              state: 'available',
            },
            {
              id: diarization,
              name: 'Local speaker model',
              model_version: '2',
              state: 'available',
            },
          ],
          next_id: '',
        },
      };
    }
    return original(request);
  };
  await mount(bridge);
  await click('Open Committed speech');
  const control = (label) => {
    const field = [...document.querySelectorAll('label')].find(
      (l) => l.textContent === label,
    );
    return document.getElementById(field.htmlFor);
  };
  assert.equal(
    control('Saved pipeline').options[0].textContent,
    'Explicit local model selection',
  );
  assert.match(
    control('Local recognition model').textContent,
    /Local speech model/,
  );
  assert.match(
    control('Local diarization model').textContent,
    /Local speaker model/,
  );
  await fill('Local recognition model', recognition);
  await fill('Local diarization model', diarization);
  const options = (label) => [...control(label).options].map((o) => o.value);
  const request = (data) => ({
    operation: 'recordings.process',
    workspace_id: wid,
    item_id: mid,
    data,
  });
  for (const transcription of options('Transcription input')) {
    const validation = validateNativeRequest(
      request({
        ...(transcription ? { transcription } : {}),
        diarization: 'run',
        recognition_model_id: recognition,
        diarization_model_id: diarization,
      }),
    );
    assert.equal(validation.valid, true, validation.errors);
  }
  for (const diarizationChoice of options('Speaker attribution')) {
    const validation = validateNativeRequest(
      request({
        transcription: 'supplied',
        diarization: diarizationChoice,
        diarization_model_id: diarization,
      }),
    );
    assert.equal(validation.valid, true, validation.errors);
  }
  assert.equal(options('Speaker attribution').includes('off'), false);
  assert.equal(
    validateNativeRequest(
      request({
        transcription: 'supplied',
        diarization: 'off',
        diarization_model_id: diarization,
      }),
    ).valid,
    false,
  );
  await fill('Transcription input', 'generate');
  await click('Rerun processing');
  const generated = bridge.calls.find(
    (c) => c.operation === 'recordings.process',
  );
  assert.deepEqual(generated.data, {
    recognition_model_id: recognition,
    diarization_model_id: diarization,
    transcription: 'generate',
    diarization: 'run',
  });
  assert.equal(validateNativeRequest(generated).valid, true);
  await fill('Transcription input', 'supplied');
  await fill('Speaker attribution', 'reuse');
  await click('Rerun processing');
  assert.deepEqual(
    bridge.calls.filter((c) => c.operation === 'recordings.process').at(-1)
      .data,
    {
      transcription: 'supplied',
      diarization: 'reuse',
    },
  );
  await unmount();
});
test('rendered pinned model registration matches the published manifest contract', async () => {
  const bridge = mock();
  await mount(bridge);
  await click('Settings');
  for (const [label, value] of [
    ['Model name', 'Contract model'],
    ['Model version', '1'],
    ['Upstream revision', 'immutable-revision'],
    ['Model license', 'MIT'],
    ['Model file source URL', 'https://example.invalid/model'],
    ['Model file SHA-256', '6'.repeat(64)],
    ['Model file byte size', '100'],
  ])
    await fill(label, value);
  await click('Add pinned file');
  await click('Register model manifest');
  const request = bridge.calls.find((c) => c.operation === 'models.register');
  assert.equal(validateNativeRequest(request).valid, true);
  assert.deepEqual(Object.keys(request.data), ['manifest']);
  assert.equal(
    bridge.calls.some((c) => c.operation === 'models.acquire'),
    false,
  );
  await unmount();
});

test('model aliases and configured source discovery use revisioned shared requests', async () => {
  const bridge=mock(), original=bridge.Operate;
  let aliases=[], sources=[];
  bridge.Operate=async request=>{
    if(request.operation.startsWith('models.alias.')||request.operation.startsWith('models.source.')||request.operation==='models.discover'||request.operation==='models.resolve'){
      const validation=validateNativeRequest(request);assert.equal(validation.valid,true,validation.errors);bridge.calls.push(request);
      const aliasOperation=request.operation.startsWith('models.alias.');
      const items=aliasOperation?aliases:sources;
      if(request.operation.endsWith('.list'))return {result:{items,next_id:''}};
      if(request.operation.endsWith('.set')){
        const value={...(aliasOperation?request.data.alias:request.data.source),revision:items.length?8:7};
        assert.equal(request.data.expected_revision,items[0]?.revision??0);
        if(aliasOperation)aliases=[value];else sources=[value];
        return {result:value};
      }
      if(request.operation.endsWith('.show'))return {result:items[0]};
      if(request.operation.endsWith('.remove')){assert.equal(request.data.expected_revision,items[0].revision);if(aliasOperation)aliases=[];else sources=[];return {result:{state:'deleted'}};}
      if(request.operation==='models.discover')return {result:{items:[{selector:'tiny',reference:'source:local/tiny',manifest_digest:'a'.repeat(64),state:'discovered',compatible:true,diagnostics:[]}]}};
      return {result:{reference:request.data.reference,target:{kind:'base',id:mid,operation:request.data.operation},state:'registered',compatible:true,manifest_digest:'a'.repeat(64),upstream_revision:'exact',diagnostics:[]}};
    }
    return original(request);
  };
  await mount(bridge);await click('Settings');
  await fill('Model alias name','speech');await fill('Exact model target JSON',JSON.stringify({kind:'base',id:mid,operation:'transcription'}));await click('Save model alias');
  assert.match(document.body.textContent,/speech/);assert.match(document.body.textContent,/Expected alias revision: 7/);
  await click('Edit alias speech');await fill('Exact model target JSON',JSON.stringify({kind:'base',id:wid,operation:'transcription'}));await click('Save model alias');
  assert.equal(bridge.calls.filter(c=>c.operation==='models.alias.set').at(-1).data.expected_revision,7);
  await fill('Model source name','local');await fill('Model source configuration JSON',JSON.stringify({url:'http://127.0.0.1:8123/catalog',local_http:true}));await click('Save model source');
  await fill('Discovery source',sources[0].id);await click('Discover source models');
  assert.match(document.body.textContent,/discovered/);assert.match(document.body.textContent,/not an installed or verified model/);
  await fill('Model reference','source:local/tiny');await click('Resolve model reference');
  assert.equal(bridge.calls.find(c=>c.operation==='models.resolve').data.reference,'source:local/tiny');
  assert.equal(bridge.calls.some(c=>c.operation==='models.acquire'),false);
  await click('Remove alias speech');await click('Remove source local');await unmount();
});

test('shared selectors accept missing aliases and show compatibility without hiding exact identity',async()=>{
  const bridge=mock(),original=bridge.Operate;
  bridge.Operate=async request=>{
    if(request.operation==='models.list')return {result:{items:[{id:mid,name:'declared-speech',model_version:'v1',capabilities:['transcription'],state:'registered',operation_compatibility:{transcription:false,diarization:false}}],next_id:''}};
    if(request.operation==='models.alias.list')return {result:{items:[{id:wid,name:'queued-speech',revision:4,state:'active',target:{kind:'base',id:mid,operation:'transcription'}}],next_id:''}};
    if(request.operation==='models.resolve'){assert.equal(validateNativeRequest(request).valid,true);bridge.calls.push(request);return {result:{reference:request.data.reference,target:{kind:'base',id:mid,operation:'transcription'},state:'registered',compatible:false,diagnostics:['unsupported-runtime'],manifest_digest:'a'.repeat(64),upstream_revision:'immutable'}};}
    return original(request);
  };
  await mount(bridge);await click('Open Committed speech');await fill('Transcription input','generate');
  assert.match(field('Local recognition model').textContent,/queued-speech/);
  assert.match(field('Local recognition model').textContent,/queued-speech.*incompatible with default adapter/);
  assert.match(field('Local recognition model').textContent,/declared-speech.*v1, transcription, registered.*incompatible with default adapter/);
  await fill('Local recognition model','queued-speech');await click('Inspect local recognition model reference');
  assert.match(document.body.textContent,/Incompatible reference: registered/);assert.match(document.body.textContent,/unsupported-runtime/);
  await fill('Local diarization model reference','base:'+mid);await click('Rerun processing');
  const request=bridge.calls.find(c=>c.operation==='recordings.process');assert.equal(request.data.recognition_model_id,'queued-speech');assert.equal(request.data.diarization_model_id,'base:'+mid);assert.equal(validateNativeRequest(request).valid,true);
  await unmount();
});

test('local import, processing and pipeline choices offer base aliases while retaining other kinds in global inspection',async()=>{
  const bridge=mock(),original=bridge.Operate;
  const aliases=['transcription','diarization'].flatMap((operation,operationIndex)=>['base','speaker','hosted'].map((kind,index)=>({id:`${operationIndex*3+index+1}1111111-1111-4111-8111-111111111111`,name:`${kind}-${operation}`,revision:1,state:'active',target:{kind,operation,...(kind==='hosted'?{adapter:'insonic-http',contract_version:'1',endpoint:'https://example.test/worker',remote_model:'remote',upstream_revision:'1'}:{id:mid})}})));
  bridge.Operate=async request=>{
    if(request.operation==='models.alias.list')return {result:{items:aliases,next_id:''}};
    if(request.operation==='models.resolve')return {result:{reference:request.data.reference,target:aliases.find(alias=>alias.name===request.data.reference).target,state:'hosted-only',compatible:true,diagnostics:[]}};
    return original(request);
  };
  const choices=(label,operation)=>{
    const options=[...field(label).options].map(option=>option.value);
    assert.equal(options.includes(`base-${operation}`),true,label);
    for(const kind of ['speaker','hosted'])assert.equal(options.includes(`${kind}-${operation}`),false,`${label} cannot offer ${kind} for local execution`);
  };
  await mount(bridge);
  await fill('Import speaker attribution','diarize');choices('Import diarization model','diarization');
  await click('Open Committed speech');await fill('Transcription input','generate');
  choices('Local recognition model','transcription');choices('Local diarization model','diarization');
  await fill('Local recognition model reference','hosted-transcription');
  assert.equal([...field('Local recognition model').options].find(option=>option.value==='hosted-transcription').disabled,true);
  await click('Inspect local recognition model reference');assert.match(document.body.textContent,/Unavailable for local processing: hosted-only/);
  await click('Pipelines');choices('recognition model','transcription');choices('diarization model','diarization');
  await click('Settings');
  const table=[...document.querySelectorAll('table')].find(table=>table.querySelector('caption')?.textContent==='Workspace model aliases');
  assert.match(table.textContent,/speaker-transcription/);assert.match(table.textContent,/hosted-transcription/);
  await unmount();
});

test('job details distinguish frozen models and acquisition work from processing completion',async()=>{
  const bridge=mock(),original=bridge.Operate;
  bridge.Operate=async request=>{
    if(request.operation==='work.list')return {result:{items:[{id:mid,kind:'recordings.process',state:'pending',phase:'acquiring-models'}],next_id:''}};
    if(request.operation==='work.show')return {result:{id:mid,state:'pending',phase:'acquiring-models',generation:0,model_selections:[{reference:'speech',manifest_digest:'a'.repeat(64)}],acquisition_ids:[wid]}};
    return original(request);
  };
  await mount(bridge);await click('Jobs');await click('Inspect job');
  assert.match(document.body.textContent,/acquiring-models/);assert.match(document.body.textContent,/selected models/);assert.match(document.body.textContent,/acquisition jobs/);assert.match(document.body.textContent,/speech/);await unmount();
});
test('rendered terminology form persists and rereads the shared contract', async () => {
  const bridge = mock();
  await mount(bridge);
  await click('Terms');
  await fill('Canonical term', 'Archive term');
  await fill('Term variants', 'archive spelling');
  await click('Save term');
  const saved = bridge.calls.find((c) => c.operation === 'terms.set');
  assert.equal(saved.data.term.canonical, 'Archive term');
  assert.deepEqual(saved.data.term.variants, ['archive spelling']);
  assert.equal(saved.data.expected_revision, 0);
  assert.ok(
    bridge.calls.find(
      (c) => c.operation === 'terms.show' && c.item_id === saved.item_id,
    ),
  );
  assert.match(document.querySelector('table').textContent, /Archive term/);
  await unmount();
});
test('buffered playback is paused and current evidence removed after authority replacement', async () => {
  const bridge = mock();
  bridge.VerifyPlayback = async () => ({
    error: { code: 'conflict', message: 'Current recording replaced.' },
  });
  await mount(bridge);
  await click('Open Committed speech');
  await click('Play recording audio');
  const audio = document.querySelector('audio');
  let paused = false;
  audio.pause = () => {
    paused = true;
  };
  await act(() => new Promise((resolve) => setTimeout(resolve, 5100)));
  await tick();
  assert.equal(paused, true);
  assert.equal(document.querySelector('audio'), null);
  assert.equal(document.body.textContent.includes('Historical speech'), false);
  assert.match(
    document.querySelector('[role=alert]').textContent,
    /Refresh current results and reacquire/,
  );
  await unmount();
});
test('connected pipeline dedicated fields save elected routes and inspect without executing them', async () => {
  const bridge = mock();
  await mount(bridge);
  await click('Pipelines');
  await fill('Pipeline name', 'Hosted route');
  await fill('Preset', 'connected');
  for (const stage of ['recognition', 'diarization']) {
    await fill(`${stage} endpoint`, 'http://127.0.0.1:9/fixture');
    await fill(`${stage} remote model`, 'no-inference');
  }
  await click('Save pipeline');
  const saved = bridge.calls.find((c) => c.operation === 'pipelines.set');
  assert.equal(saved.data.pipeline.preset, 'connected');
  assert.equal(saved.data.pipeline.configuration.recognition.mode, 'hosted');
  assert.deepEqual(saved.data.pipeline.configuration.recognition.capabilities, [
    'transcription',
  ]);
  assert.equal(
    saved.data.pipeline.configuration.diarization.model_id,
    undefined,
  );
  await click('Inspect elected stages and context');
  assert.match(document.body.textContent, /not-probed/);
  assert.equal(
    bridge.calls.some((c) => c.operation === 'recordings.process'),
    false,
  );
  await unmount();
});
test('Library selectors page beyond the first 100 entities and map the chosen later person', async () => {
  const bridge = mock(),
    original = bridge.Operate;
  const late = 'cccccccc-cccc-4ccc-8ccc-cccccccccccc';
  bridge.Operate = async (request) => {
    if (request.operation === 'pipelines.show')
      return {
        workspace_id: wid,
        result: {
          id: request.item_id,
          name: 'Later catalog item',
          revision: 2,
        },
      };
    if (
      request.operation === 'speakers.list' ||
      request.operation === 'pipelines.list'
    ) {
      const item = request.data?.after_id
        ? { id: late, name: 'Later catalog item', revision: 2 }
        : { id: wid, name: 'First catalog item', revision: 1 };
      return {
        workspace_id: wid,
        result: {
          items: [
            request.operation === 'speakers.list'
              ? { speaker: item, aliases: [] }
              : item,
          ],
          next_id: request.data?.after_id ? '' : wid,
        },
      };
    }
    return original(request);
  };
  await mount(bridge);
  await click('Open Committed speech');
  await click('Load more saved pipelines');
  await click('Load more known speakers');
  await fill('Saved pipeline', late);
  await fill('Local voice', mid);
  await fill('Known person', late);
  await click('Save current mapping');
  assert.equal(
    bridge.calls.find((c) => c.operation === 'recordings.map-speaker').data
      .speaker_id,
    late,
  );
  await unmount();
});
test('rendered speaker form keeps aliases separately associated with person identity', async () => {
  const bridge = mock();
  await mount(bridge);
  await click('Speakers');
  await fill('Speaker name', 'Archive speaker');
  await fill('Aliases', 'Historical name');
  await click('Save speaker and aliases');
  const saved = bridge.calls.find((c) => c.operation === 'speakers.set');
  assert.equal(saved.data.speaker.name, 'Archive speaker');
  assert.equal(saved.data.aliases[0].speaker_id, saved.item_id);
  assert.equal(saved.data.aliases[0].text, 'Historical name');
  assert.match(document.querySelector('table').textContent, /Archive speaker/);
  assert.ok(button('Edit Archive speaker'));
  await click('Terms');
  const speakerField = [...document.querySelectorAll('label')].find(
    (l) => l.textContent === 'Associated speaker',
  );
  const speakerChoice = document.getElementById(speakerField.htmlFor);
  assert.equal(
    [...speakerChoice.options].find((option) => option.value === saved.item_id)
      .textContent,
    'Archive speaker',
  );
  await unmount();
});
test('mapping pagination retains previously observed revisions for selected voices', async () => {
  const bridge = mock(),
    original = bridge.Operate;
  bridge.Operate = async (request) => {
    const response = await original(request);
    if (request.operation === 'recordings.mappings')
      response.result = request.data?.after_id
        ? {
            items: [{ local_speaker_id: wid, speaker_id: mid, revision: 2 }],
            next_id: '',
          }
        : {
            items: [{ local_speaker_id: mid, speaker_id: wid, revision: 9 }],
            next_id: wid,
          };
    if (request.operation === 'speakers.list')
      response.result = {
        items: [
          {
            speaker: {
              id: wid,
              name: 'Known person',
              state: 'active',
              revision: 1,
            },
            aliases: [],
          },
        ],
        next_id: '',
      };
    return response;
  };
  await mount(bridge);
  await click('Open Committed speech');
  await fill('Local voice', mid);
  await fill('Known person', wid);
  await click('Next page');
  await click('Save current mapping');
  const saved = bridge.calls.find(
    (c) => c.operation === 'recordings.map-speaker',
  );
  assert.equal(saved.data.mapping_revision, 9);
  assert.equal(saved.data.local_speaker_id, mid);
  await unmount();
});
test('mixed untimed speaker participation retains playable source intervals', async () => {
  const bridge = mock(),
    original = bridge.Operate;
  bridge.Operate = async (request) => {
    const response = await original(request);
    if (request.operation === 'speakers.select')
      response.result = {
        selection: {
          diagnostics: [],
          duplicates: [],
          partial: false,
          items: [
            {
              reference: {
                recording_id: mid,
                recording_revision: 3,
                document_digest: '5'.repeat(64),
                cue_id: 'mixed-cue',
              },
              untimed: true,
              source_intervals: [{ start_seconds: '1/1', end_seconds: '2/1' }],
            },
          ],
        },
        next_cursor: '',
      };
    return response;
  };
  await mount(bridge);
  await click('Speakers');
  await fill('Speaker name', 'Mixed participation');
  await click('Save speaker and aliases');
  await click('Inspect current speaker evidence');
  assert.match(
    document.querySelector('table caption').parentElement.textContent,
    /Mixed participation/,
  );
  const corpus = [...document.querySelectorAll('table')].find(
    (table) =>
      table.querySelector('caption')?.textContent === 'Current speaker corpus',
  );
  assert.match(corpus.textContent, /Untimed participation/);
  await click('Play 1/1–2/1 seconds');
  assert.deepEqual(bridge.calls.find((c) => c.operation === 'Playback').args, [
    mid,
    1,
    3,
    '5'.repeat(64),
  ]);
  const audio = document.querySelector('audio');
  await act(async () =>
    audio.dispatchEvent(new Event('loadedmetadata', { bubbles: true })),
  );
  assert.equal(audio.currentTime, 1);
  await unmount();
});
test('rendered Jobs recovery acts on the selected real work ID', async () => {
  const bridge = mock();
  await mount(bridge);
  await click('Jobs');
  await click('Retry selected job');
  assert.equal(
    bridge.calls.find((c) => c.operation === 'work.retry').item_id,
    mid,
  );
  assert.equal(
    bridge.calls.some((c) => c.operation === 'jobs.start'),
    false,
  );
  await unmount();
});
test('failed supplied assembly asks for current inputs rather than queuing an unrecoverable retry', async () => {
  const bridge = mock(),
    original = bridge.Operate;
  bridge.Operate = async (request) => {
    const response = await original(request);
    if (request.operation === 'work.list')
      response.result.items = [
        {
          id: mid,
          kind: 'recordings.assemble',
          state: 'failed',
          phase: 'assembly',
        },
      ];
    return response;
  };
  await mount(bridge);
  await click('Jobs');
  assert.match(
    document.querySelector('table').textContent,
    /Open this recording in Library, supply the intended turns/,
  );
  assert.equal(
    [...document.querySelectorAll('button')].some(
      (b) => b.textContent === 'Retry selected job',
    ),
    false,
  );
  assert.equal(
    bridge.calls.some((c) => c.operation === 'work.retry'),
    false,
  );
  assert.ok(button('Inspect job'));
  await unmount();
});
test('settings appearance CAS and credential protected entry keep secrets out of Operate', async () => {
  const bridge = mock();
  await mount(bridge);
  await click('Settings');
  await fill('Theme preference', 'light');
  await click('Save appearance');
  const saved = bridge.calls.find((c) => c.operation === 'settings.set');
  assert.deepEqual(saved.data, {
    section: 'appearance',
    revision: '3'.repeat(64),
    value: { theme: 'light', reduced_motion: false },
  });
  await fill('Credential reference ID', mid);
  await fill('New credential value', 'synthetic-test-only');
  await click('Add credential');
  assert.equal(
    bridge.calls.find((c) => c.operation === 'Credential').args[0],
    'add',
  );
  assert.equal(document.querySelector('input[type=password]').value, '');
  assert.equal(
    bridge.calls
      .filter((c) => c.operation !== 'Credential')
      .some((c) => JSON.stringify(c).includes('synthetic-test-only')),
    false,
  );
  await unmount();
});
test('reopened vault and session selections remain authoritative until an explicit backend change', async () => {
  for (const selected of ['vault', 'session']) {
    const bridge = mock(),
      original = bridge.Operate;
    let persisted = selected;
    bridge.Operate = async (request) => {
      const response = await original(request);
      if (request.operation === 'settings.show')
        response.result.credentials = { backend: persisted };
      return response;
    };
    bridge.Credential = async (args, input) => {
      bridge.calls.push({ operation: 'Credential', args, input });
      if (args[0] === 'select') persisted = args[1];
      return { result: { state: 'configured' } };
    };
    await mount(bridge);
    await click('Settings');
    assert.equal(field('Credential backend').value, selected);
    assert.equal(button('Select credential backend').disabled, true);
    await click('Select credential backend');
    await click('Library');
    await click('Settings');
    assert.equal(field('Credential backend').value, selected);
    assert.equal(
      bridge.calls.some((c) => c.operation === 'Credential'),
      false,
    );
    await fill('Credential backend', 'native');
    assert.equal(button('Select credential backend').disabled, false);
    await click('Select credential backend');
    assert.deepEqual(
      bridge.calls.find((c) => c.operation === 'Credential').args,
      ['select', 'native'],
    );
    assert.equal(field('Credential backend').value, persisted);
    assert.equal(button('Select credential backend').disabled, true);
    await unmount();
  }
});
test('failed authoritative settings reads never guess or submit a credential backend', async () => {
  const bridge = mock(),
    original = bridge.Operate;
  bridge.Operate = (request) =>
    request.operation === 'settings.show'
      ? Promise.resolve({
          workspace_id: wid,
          error: { code: 'unavailable', message: 'Selection unavailable.' },
        })
      : original(request);
  await mount(bridge);
  await click('Settings');
  assert.equal(field('Credential backend').value, '');
  assert.equal(button('Select credential backend').disabled, true);
  await click('Select credential backend');
  assert.equal(
    bridge.calls.some((c) => c.operation === 'Credential'),
    false,
  );
  await unmount();
});
test('effective package tools remain read-only and unchanged saves never persist package paths', async () => {
  const bridge = mock(),
    original = bridge.Operate;
  bridge.Operate = async (request) => {
    const response = await original(request);
    if (request.operation === 'settings.show') {
      response.result.sections.media_tools.value =
        mediaToolFixture('/package-A');
      response.result.sections.processing_tools.value = {
        kind: 'processing-tools',
        schema_version: '0.0.0',
        cueson: {
          executable: '/package-A/cueson.exe',
          executable_sha256: '9'.repeat(64),
        },
        processing: {},
      };
    }
    return response;
  };
  await mount(bridge);
  await click('Settings');
  assert.match(document.body.textContent, /\/package-A\/ffprobe.exe/);
  assert.match(document.body.textContent, /\/package-A\/cueson.exe/);
  assert.equal(field('ffprobe executable path').value, '');
  assert.equal(field('Cueson executable path').value, '');
  assert.equal(
    field('Advanced media tool contract').value.includes('/package-A'),
    false,
  );
  assert.equal(
    field('Advanced processing tool contract').value.includes('/package-A'),
    false,
  );
  for (const text of [
    'Save media tools',
    'Save processing tools',
    'Validate and save advanced media tools',
    'Validate and save advanced processing tools',
  ]) {
    assert.equal(button(text).disabled, true);
    await click(text);
  }
  assert.equal(
    bridge.calls.some((c) => c.operation === 'settings.set'),
    false,
  );
  await unmount();
});
test('explicit media overrides are intentional and resetting rereads relocated package defaults with CAS', async () => {
  const bridge = mock(),
    original = bridge.Operate;
  let override,
    relocated = false;
  bridge.Operate = async (request) => {
    const response = await original(request);
    if (
      request.operation === 'settings.set' &&
      request.data.section === 'media_tools'
    ) {
      override = request.data.value;
      if (override === null) relocated = true;
    }
    if (request.operation === 'settings.show')
      response.result.sections.media_tools = {
        origin: override ? 'workspace' : 'package',
        revision: override ? 'a'.repeat(64) : '1'.repeat(64),
        value:
          override ?? mediaToolFixture(relocated ? '/package-B' : '/package-A'),
      };
    return response;
  };
  await mount(bridge);
  await click('Settings');
  await act(async () => field('Edit explicit workspace media tools').click());
  await tick();
  assert.equal(field('ffprobe executable path').value, '');
  for (const [name, hash] of [
    ['ffprobe', '7'],
    ['exiftool', '8'],
  ]) {
    await fill(`${name} executable path`, `/explicit/${name}.exe`);
    await fill(`${name} SHA-256`, hash.repeat(64));
    await fill(`${name} version`, '1');
  }
  await click('Save media tools');
  const saved = bridge.calls.find((c) => c.operation === 'settings.set');
  assert.deepEqual(saved.data.value, mediaToolFixture('/explicit'));
  assert.equal(saved.data.revision, '1'.repeat(64));
  assert.equal(
    button('Reset media tools to installed defaults').disabled,
    false,
  );
  await click('Reset media tools to installed defaults');
  const reset = bridge.calls
    .filter((c) => c.operation === 'settings.set')
    .at(-1);
  assert.deepEqual(reset.data, {
    section: 'media_tools',
    revision: 'a'.repeat(64),
    value: null,
  });
  assert.equal(field('Edit explicit workspace media tools').checked, false);
  assert.equal(field('ffprobe executable path').value, '');
  assert.equal(button('Save media tools').disabled, true);
  assert.match(document.body.textContent, /\/package-B\/ffprobe.exe/);
  assert.equal(document.body.textContent.includes('/package-A'), false);
  await unmount();
});
test('processing tool reset uses the observed workspace revision and clears explicit editor paths', async () => {
  const bridge = mock(),
    original = bridge.Operate;
  let explicit = true;
  bridge.Operate = async (request) => {
    const response = await original(request);
    if (
      request.operation === 'settings.set' &&
      request.data.section === 'processing_tools'
    )
      explicit = false;
    if (request.operation === 'settings.show')
      response.result.sections.processing_tools = {
        origin: explicit ? 'workspace' : 'package',
        revision: explicit ? 'b'.repeat(64) : '2'.repeat(64),
        value: {
          kind: 'processing-tools',
          schema_version: '0.0.0',
          cueson: {
            executable: explicit
              ? '/explicit/cueson.exe'
              : '/package-B/cueson.exe',
            executable_sha256: '9'.repeat(64),
          },
          processing: {},
        },
      };
    return response;
  };
  await mount(bridge);
  await click('Settings');
  assert.equal(field('Cueson executable path').value, '/explicit/cueson.exe');
  await click('Reset processing tools to installed defaults');
  assert.deepEqual(
    bridge.calls.find((c) => c.operation === 'settings.set').data,
    { section: 'processing_tools', revision: 'b'.repeat(64), value: null },
  );
  assert.equal(field('Cueson executable path').value, '');
  assert.equal(
    field('Edit explicit workspace processing tools').checked,
    false,
  );
  assert.equal(button('Save processing tools').disabled, true);
  assert.match(document.body.textContent, /\/package-B\/cueson.exe/);
  await unmount();
});
test('core fields have associated labels, navigation is native keyboard focusable, help is offline', async () => {
  const bridge = mock();
  await mount(bridge);
  for (const control of document.querySelectorAll('input,select,textarea'))
    assert.ok(
      [...document.querySelectorAll('label')].some(
        (l) => l.htmlFor === control.id,
      ),
    );
  assert.equal(
    document.querySelector('[data-bb-host]').dataset.bbHost,
    'wails',
  );
  assert.ok(document.querySelector('a[href="/assets/help/index.html"]'));
  button('Jobs').focus();
  assert.equal(document.activeElement, button('Jobs'));
  assert.equal(
    document.querySelector('table caption').textContent,
    'Current audio and video',
  );
  await unmount();
});
test('revision conflict displays actionable failure without optimistic term row', async () => {
  const bridge = mock(),
    original = bridge.Operate;
  bridge.Operate = (req) =>
    req.operation === 'terms.set'
      ? Promise.resolve({
          workspace_id: wid,
          error: { code: 'conflict', message: 'Another revision is current.' },
        })
      : original(req);
  await mount(bridge);
  await click('Terms');
  await fill('Canonical term', 'Not persisted');
  await click('Save term');
  assert.match(
    document.querySelector('[role=alert]').textContent,
    /Another revision/,
  );
  assert.equal(
    document.querySelector('table').textContent.includes('Not persisted'),
    false,
  );
  await unmount();
});
test('workspace-open failure remains visible after request ownership changes', async () => {
  const bridge = mock();
  bridge.SelectWorkspace = async () => ({
    error: { code: 'not_found', message: 'Workspace unavailable.' },
  });
  await mount(bridge);
  await fill('Workspace path', '/missing');
  await click('Open workspace');
  assert.match(
    document.querySelector('[role=alert]').textContent,
    /Workspace unavailable/,
  );
  await unmount();
});
test('malformed saved appearance stays repairable with the authoritative hash revision', async () => {
  const bridge = mock(),
    original = bridge.Operate;
  let repaired = false;
  bridge.Operate = async (request) => {
    const response = await original(request);
    if (request.operation === 'settings.show' && !repaired)
      response.result.sections.appearance = {
        revision: '4'.repeat(64),
        value: null,
        origin: 'workspace',
        error: { code: 'invalid_request', message: 'Invalid appearance.' },
      };
    if (
      request.operation === 'settings.set' &&
      request.data.section === 'appearance'
    )
      repaired = true;
    return response;
  };
  await mount(bridge);
  await click('Settings');
  assert.match(
    document.querySelector('[role=alert]').textContent,
    /appearance is invalid/,
  );
  await fill('Theme preference', 'light');
  await click('Save appearance');
  const saved = bridge.calls.find((c) => c.operation === 'settings.set');
  assert.equal(saved.data.revision, '4'.repeat(64));
  assert.equal(saved.data.value.theme, 'light');
  assert.equal(document.querySelector('[role=alert]'), null);
  await unmount();
});
test('late startup Show cannot replace a newly opened workspace', async () => {
  const bridge = mock();
  let complete;
  bridge.Show = () =>
    new Promise((resolve) => {
      complete = resolve;
    });
  bridge.SelectWorkspace = async () => ({
    result: { workspace_id: mid, display_name: 'New selection' },
  });
  await mount(bridge);
  await fill('Workspace path', '/new');
  await click('Open workspace');
  await act(async () => {
    complete({ workspace_id: wid, result: { display_name: 'Old startup' } });
  });
  await tick();
  assert.match(document.body.textContent, /Selected: New selection/);
  assert.equal(
    document.body.textContent.includes('Selected: Old startup'),
    false,
  );
  await unmount();
});


test('native qualification bounds a webview play promise that never settles', async () => {
  const {qualificationPlay}=await import('../.test-build/qualification.js');
  await assert.rejects(qualificationPlay({play:()=>new Promise(()=>{})},5),error=>error.name==='TimeoutError');
  await qualificationPlay({play:async()=>{}},5);
});

test('library playback tickets reset media readiness before applying a new source seek', async () => {
  const bridge = mock();
  let ticket = 0;
  bridge.Playback = async () => ({result:{url:`/media/ticket-${++ticket}`,mime_type:'audio/wav'}});
  await mount(bridge);
  await click('Open Committed speech');
  await click('Seek 1250 ms');
  const old = document.querySelector('audio');
  Object.defineProperty(old, 'readyState', {value:1});
  await act(async () => old.dispatchEvent(new Event('loadedmetadata', {bubbles:true})));
  assert.equal(old.currentTime, 1.25);
  await click('Seek 1250 ms');
  const fresh = document.querySelector('audio');
  try {
  assert.notEqual(fresh, old, 'a new ticket must not retain the old resource readiness');
  assert.equal(fresh.readyState, 0);
  assert.equal(fresh.currentTime, 0);
  await act(async () => fresh.dispatchEvent(new Event('loadedmetadata', {bubbles:true})));
  assert.equal(fresh.currentTime, 1.25);
  } finally { await unmount(); }
});

test('library source seek survives an early metadata seek lost before playable data', async () => {
  const bridge = mock();
  await mount(bridge);
  await click('Open Committed speech');
  await click('Seek 1250 ms');
  const audio = document.querySelector('audio');
  let ready = 1, time = 0;
  Object.defineProperty(audio, 'readyState', {get: () => ready});
  Object.defineProperty(audio, 'currentTime', {get: () => time,set: value => {time=value;}});
  try {
    await act(async()=>audio.dispatchEvent(new Event('loadedmetadata',{bubbles:true})));
    await act(async()=>audio.dispatchEvent(new Event('seeked',{bubbles:true})));
    time=0;
    assert.equal(audio.currentTime,0,'native decoder can lose a seek before media data arrives');
    ready=2;
    await act(async()=>audio.dispatchEvent(new Event('loadeddata',{bubbles:true})));
    assert.equal(audio.currentTime,1.25,'pending source seek must recover when data becomes playable');
    await act(async()=>audio.dispatchEvent(new Event('seeked',{bubbles:true})));
    audio.currentTime=2;
    ready=4;
    await act(async()=>audio.dispatchEvent(new Event('canplay',{bubbles:true})));
    assert.equal(audio.currentTime,2,'later buffering must preserve user playback after confirmed seek');
  } finally {await unmount();}
});

test('native mount readiness waits for delayed workspace controls and still rejects missing screens', async () => {
  const {qualificationMounted} = await import('../.test-build/qualification.js');
  const host = document.createElement('div');
  host.setAttribute('data-bb-host', 'wails');
  host.innerHTML = '<form></form>';
  document.body.append(host);
  try {
    await assert.rejects(qualificationMounted(1), /screens did not mount/);
    const waiting = qualificationMounted(1000);
    const timer = setTimeout(() => {
      const controls = document.createElement('div');
      controls.setAttribute('aria-label', 'Library controls');
      host.append(controls);
    }, 30);
    try { await waiting; } finally { clearTimeout(timer); }
  } finally { host.remove(); }
});


test('rendered standalone transcript keeps independent limits and replacement election',async()=>{
 const bridge=mock();await mount(bridge);
 await act(async()=>field('Import transcript into an existing recording').click());await tick();
 await fill('Existing recording ID or exact title',mid);await fill('Transcript path or URL','https://example.test/captions.cueson.json');
 await fill('Import speaker attribution','off');await fill('Transcript format','cueson');await fill('Transcript byte limit','4096');await fill('Transcript timeout in milliseconds','30000');
 await act(async()=>field('Replace an existing current transcript').click());await tick();await click('Import');
 const request=bridge.calls.find(c=>c.operation==='media.import');assert.equal(request.data.items[0].record,mid);assert.equal(request.data.items[0].source,undefined);assert.equal(request.data.defaults.replace_transcript,true);assert.equal(request.data.defaults.attribution,'off');assert.equal(request.data.defaults.transcript_max_bytes,4096);assert.equal(request.data.defaults.transcript_timeout_ms,30000);assert.equal(validateNativeRequest(request).valid,true);await unmount();
});

test('invalid numeric admission controls fail before native submission',async()=>{
 const bridge=mock();await mount(bridge);await fill('Media path or URL','/fixture.wav');await fill('Transcript byte limit','NaN');await click('Import');assert.equal(bridge.calls.some(c=>c.operation==='media.import'),false);assert.match(document.body.textContent,/Enter valid transcript limits/);await unmount();
});

test('rendered replacement submits explicit stable-target elections',async()=>{
 const bridge=mock();await mount(bridge);
 await act(async()=>field('Target existing recording audio').click());await tick();
 await fill('Existing recording ID or exact title',mid);await fill('Media path or URL','/replacement.wav');
 await act(async()=>field('Replace current audio').click());await tick();
 await fill('Existing transcript policy','keep');await fill('Existing roster policy','retain');
 await act(async()=>field('Current transcript applies to replacement audio').click());await tick();await click('Import');
 const req=bridge.calls.find(c=>c.operation==='media.import');assert.equal(req.data.items[0].record,mid);assert.equal(req.data.items[0].kind,'media');assert.equal(req.data.defaults.replace_audio,true);assert.equal(req.data.defaults.existing_transcript,'keep');assert.equal(req.data.defaults.transcript_applies,true);assert.equal(req.data.defaults.existing_roster,'retain');assert.equal(validateNativeRequest(req).valid,true);await unmount();
});
test('audio-only roster distinguishes absent and empty and rereads conflicts',async()=>{
 const bridge=mock(),original=bridge.Operate;let roster={recording_id:mid,declared:false,revision:0,members:[]}, conflict=false;
 bridge.Operate=async req=>{
  if(req.operation.startsWith('recordings.roster.')){
   assert.equal(validateNativeRequest(req).valid,true);bridge.calls.push(req);
   if(req.operation==='recordings.roster.show')return {result:roster};
   if(conflict){roster={...roster,revision:8};return {error:{code:'conflict',message:'Roster changed. Refresh and retry.'}};}
   assert.equal(req.data.expected_revision,roster.revision);roster={...roster,declared:true,revision:7,members:[]};return {result:roster};
  }
  if(req.operation==='recordings.cues')return {error:{code:'not_found',message:'No document'}};
  return original(req);
 };
 await mount(bridge);await click('Open Committed speech');assert.match(document.body.textContent,/Roster not declared/);
 await click('Apply roster edit');assert.match(document.body.textContent,/Enter at least one speaker reference/);assert.equal(bridge.calls.filter(c=>c.operation==='recordings.roster.add'||c.operation==='recordings.roster.remove').length,0);
 await click('Clear roster');assert.match(document.body.textContent,/Explicitly empty roster/);assert.match(document.body.textContent,/Revision 7/);
 conflict=true;await fill('Roster speaker references',mid);await click('Apply roster edit');assert.match(document.body.textContent,/Roster changed/);assert.match(document.body.textContent,/Revision 8/);await unmount();
});
test('initial desktop roster preserves explicit empty membership',async()=>{
 const bridge=mock();await mount(bridge);await fill('Media path or URL','/fixture.wav');await act(async()=>field('Declare known speakers at import').click());await tick();await click('Import');const req=bridge.calls.find(c=>c.operation==='media.import');assert.deepEqual(req.data.defaults.known_speakers,[]);assert.equal(validateNativeRequest(req).valid,true);await unmount();
});
