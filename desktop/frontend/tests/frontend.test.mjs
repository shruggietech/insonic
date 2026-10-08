// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import { JSDOM } from 'jsdom';
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
const { pipelinePayload, speakerPayload, termPayload } =
  await import('../.test-build/forms.js');
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
  await click('Play original');
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
  await click('Play original');
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
