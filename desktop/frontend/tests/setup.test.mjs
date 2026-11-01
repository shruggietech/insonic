// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import {validateNativeRequest} from './contracts.mjs';
import { JSDOM } from 'jsdom';
const dom = new JSDOM('<!doctype html><html><body><div id="root"></div></body></html>', { url: 'http://wails.localhost' });
for (const key of ['window', 'document', 'HTMLElement', 'HTMLInputElement', 'HTMLSelectElement', 'HTMLTextAreaElement', 'Event', 'MouseEvent'])
    globalThis[key] = dom.window[key];
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
const { act, createElement } = await import('react');
const { createRoot } = await import('react-dom/client');
const { App } = await import('../.test-build/App.js');
const wid = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa', ids = ['workspace', 'backends', 'processing', 'services', 'import'];
const proof = 'a'.repeat(64);
const initialState = () => ({ status: 'active', current_stage: 'workspace', stages: ids.map(id => ({ id, state: 'pending', proof })), choices: { processing_mode: 'none' } });
function fixture(selected = false) {
    const b = { calls: [], draft: { revision: 0, path: 'C:/Users/fixture/Voice archive', plan: null, stage: 'workspace', status: 'active', credential_backend:'native' }, journey: { revision: 0, state: initialState() }, selected, lost: false };
    const current = () => ({ kind: 'first-run-setup', schema_version: '1.1.0', workspace_id: wid, ...b.journey, observations: Object.fromEntries(ids.map(id => [id, proof])), stages: b.journey.state.stages, paths: { control_directory: '.insonic', artifacts: 'artifacts', cache: 'cache', space_available_bytes: null }, actions: ['Open import', 'Open Settings', 'Open Jobs'] });
    b.Show = async () => b.selected ? { workspace_id: wid, result: { workspace_id: wid, display_name: 'Voice archive' } } : { error: { code: 'not_found', message: 'No selected library' } };
    b.SetupDraft = async (action, input, id) => { const data = JSON.parse(input); b.calls.push({ operation: 'draft.' + action, data, request_id: id }); if (action === 'save' && b.acceptedDraft?.id === id) {
        assert.deepEqual(b.acceptedDraft.data, data);
        return { result: structuredClone(b.draft) };
    } if (action === 'location')
        return { result: { path: data.path ?? b.draft.path, writable: true, available_bytes: null, existing_workspace: false, control_directory: '.insonic', artifacts: 'artifacts', cache: 'cache' } }; if (action === 'save') {
        assert.equal(data.expected_revision, b.draft.revision);
        b.draft = { ...data, revision: b.draft.revision + 1, plan: { kind: 'workspace-config', schema_version: '1.1.0', workspace_id: wid, display_name: data.plan.display_name, profiles: data.plan.profiles ?? { storage: { adapter_id: 'filesystem', configuration: { root: 'artifacts' } }, catalog: { adapter_id: 'sqlite', configuration: { path: 'catalog.sqlite' } }, graph: { adapter_id: 'ladybugdb', configuration: { path: 'graph/library.lbug' } } } } };
        b.acceptedDraft = { id, data: structuredClone(data) };
        if (b.draftLost) {
            b.draftLost = false;
            throw new Error('Draft response interrupted');
        }
    } return { result: structuredClone(b.draft) }; };
    b.BackendSetup = async (action, path, plan, input, mode, id) => { b.calls.push({ operation: 'backend.' + action, plan: JSON.parse(plan), protected: JSON.parse(input), request_id: id }); if (action === 'plan')
        return { result: { configuration: JSON.parse(plan) } }; if (action === 'preflight')
        return { result: { roles: [] } }; b.selected = true; return { result: { workspace_id: wid, path, display_name: 'Voice archive', configuration: JSON.parse(plan) } }; };
    b.SelectWorkspace = async (path) => { b.selected = true; return { result: { workspace_id: wid, path, display_name: 'Voice archive' } }; };
    b.Choose = async () => '';
    b.CancelOperation = async () => true;
    b.Credential = async (args, input) => { b.calls.push({ operation: 'protected', args, input }); return { result: { state: 'configured' } }; };
    b.Operate = async (req) => { assert.equal(validateNativeRequest(req).valid,true, req.operation+' '+validateNativeRequest(req).errors); b.calls.push(structuredClone(req)); let result = {}; if (req.operation === 'setup.show')
        result = current(); if (req.operation === 'setup.save') {
        if (b.accepted?.request_id === req.request_id) {
            assert.deepEqual(b.accepted.data, req.data);
            return { workspace_id: wid, result: current() };
        }
        if (req.data.expected_revision !== b.journey.revision)
            return { error: { code: 'conflict', message: 'Setup changed. Read current state and retry.' } };
        b.journey = { state: req.data.state, revision: b.journey.revision + 1 };
        b.accepted = structuredClone(req);
        if (b.lost) {
            b.lost = false;
            throw new Error('Response interrupted');
        }
        result = current();
    } ; if (req.operation === 'setup.readiness')
        result = { tasks: [{ task: 'import', ready: true }, { task: 'transcription', ready: false, correction: 'Prepare an elected worker' }] }; if (req.operation === 'settings.show')
        result = { sections: { appearance: { value: null }, media_tools: { origin: 'package', value: null, revision: 'a'.repeat(64) }, processing_tools: { origin: 'package', value: null, revision: 'b'.repeat(64) } }, profiles: {}, credentials: { backend: 'native' } }; if (/list$/.test(req.operation))
        result = { items: [] }; if (req.operation === 'workers.catalog')
        result = { platform: 'windows_amd64', catalog: { profiles: [] } }; if (req.operation === 'models.defaults')
        result = []; if (req.operation === 'query.assistance-show')
        result = { revision: 0, configuration: null }; return { workspace_id: wid, result }; };
    return b;
}
function button(name) { const b = [...document.querySelectorAll('button')].find(x => x.textContent === name); assert.ok(b, name); return b; }
function field(name) { const l = [...document.querySelectorAll('label')].find(x => x.textContent === name); assert.ok(l, name); return document.getElementById(l.htmlFor); }
async function click(name) { await act(async () => button(name).click()); }
async function fill(name, value) { const c = field(name); await act(async () => { Object.getOwnPropertyDescriptor(c.tagName === 'SELECT' ? HTMLSelectElement.prototype : HTMLInputElement.prototype, 'value').set.call(c, value); c.dispatchEvent(new Event('input', { bubbles: true })); c.dispatchEvent(new Event('change', { bubbles: true })); }); }
async function mount(b) { const root = createRoot(document.getElementById('root')); await act(async () => root.render(createElement(App, { bridge: b }))); return () => act(async () => root.unmount()); }
test('fresh optional setup saves location without provisioning then reuses backends and reaches import', async () => { const b = fixture(), close = await mount(b); try {
    assert.match(document.body.textContent, /Step 1 of 5/);
    assert.equal(document.activeElement.textContent, 'Workspace and location (Step 1 of 5)');
    await fill('Library location', 'C:/Users/fixture/New archive');
    await click('Save location and continue');
    assert.equal(b.calls.some(c => c.operation === 'backend.provision'), false);
    assert.match(document.body.textContent, /Step 2 of 5/);
    await click('Create configured workspace');
    assert.match(document.body.textContent, /Step 3 of 5/);
    await click('Skip this stage');
    assert.match(document.body.textContent, /Step 4 of 5/);
    await click('Skip this stage');
    assert.match(document.body.textContent, /Step 5 of 5/);
    await click('Open import');
    assert.ok(field('Media path or URL'));
    await fill('Media path or URL','C:/fixture.wav');
    await click('Import');
    assert.equal(b.calls.find(c=>c.operation==='media.import').data.items[0].source,'C:/fixture.wav');
    assert.equal(b.calls.some(c => ['workers.setup', 'models.prepare', 'protected'].includes(c.operation)), false);
    assert.ok(b.draft.plan.workspace_id);
}
finally {
    await close();
} });
test('unknown journey save keeps immutable request identity and readback does not duplicate effect', async () => { const b = fixture(true), close = await mount(b); try {
    await click('Set up library');
    b.lost = true;
    await click('Continue');
    assert.match(document.body.textContent, /Response interrupted/);
    const first = b.calls.find(c => c.operation === 'setup.save');
    await click('Read saved setup');
    await click('Retry the same save');
    const saved = b.calls.filter(c => c.operation === 'setup.save');
    assert.equal(saved.length, 2);
    assert.equal(saved[1].request_id, first.request_id);
    assert.deepEqual(saved[1].data, first.data);
    assert.equal(b.journey.revision, 1);
    assert.match(document.body.textContent, /Step 2 of 5/);
}
finally {
    await close();
} });
test('deferred setup reopens accepted stage and competing revision remains intact', async () => { const b = fixture(true), close = await mount(b); try {
    await click('Set up library');
    await click('Continue');
    await click('Set up later');
    assert.equal(b.journey.state.status, 'deferred');
    await click('Set up library');
    assert.match(document.body.textContent, /Step 2 of 5/);
    b.journey = { revision: b.journey.revision + 1, state: { ...b.journey.state, current_stage: 'services' } };
    await click('Continue');
    assert.match(document.body.textContent, /Setup changed/);
    assert.equal(b.journey.state.current_stage, 'services');
    await click('Read saved setup');
    assert.match(document.body.textContent, /Step 4 of 5/);
}
finally {
    await close();
} });
test('bootstrap draft survives remount with protected values absent from saves', async () => { const b = fixture(), close = await mount(b); await click('Save location and continue'); await fill('storage backend', 's3'); await fill('storage endpoint', 'https://storage.example'); await fill('storage bucket', 'fixture'); await fill('storage credential access key', 'fixture-access'); await fill('storage credential secret key', 'private-setup-sentinel'); await click('Prepare backend plan'); assert.equal(JSON.stringify(b.calls.filter(c => c.operation.startsWith('draft.'))).includes('private-setup-sentinel'), false); await close(); const close2 = await mount(b); try {
    assert.match(document.body.textContent, /Step 2 of 5/);
    assert.equal(field('storage backend').value, 's3');
    assert.equal(field('storage bucket').value, 'fixture');
    assert.equal(field('storage credential secret key').value, '');
}
finally {
    await close2();
} });
test('all five headings receive focus, Back preserves data and finish remains reopenable in Settings', async () => { const b = fixture(true), close = await mount(b); try {
    await click('Set up library');
    for (const [step, action] of [[1, 'Continue'], [2, 'Continue'], [3, 'Save stage and continue'], [4, 'Save stage and continue']]) {
        assert.match(document.activeElement.textContent, new RegExp('Step ' + step + ' of 5'));
        await click(action);
    }
    assert.match(document.activeElement.textContent, /Step 5 of 5/);
    assert.match(document.body.textContent, /Prepare an elected worker/);
    await click('Back');
    assert.match(document.activeElement.textContent, /Step 4 of 5/);
    await click('Save stage and continue');
    await click('Finish setup');
    assert.equal(b.journey.state.status, 'completed');
    await click('Settings');
    await click('Reopen setup');
    assert.match(document.activeElement.textContent, /Step 5 of 5/);
    for (const c of document.querySelectorAll('input,select,textarea'))
        assert.ok([...document.querySelectorAll('label')].some(l => l.htmlFor === c.id));
}
finally {
    await close();
} });
test('unknown bootstrap save freezes draft bytes until identical retry and never provisions eagerly', async () => { const b = fixture(), close = await mount(b); try {
    b.draftLost = true;
    await click('Save location and continue');
    assert.match(document.body.textContent, /Draft response interrupted/);
    await click('Read saved setup');
    assert.ok(button('Create configured workspace').matches(':disabled'));
    await click('Retry the same save');
    const saves = b.calls.filter(c => c.operation === 'draft.save');
    assert.equal(saves.length, 2);
    assert.equal(saves[0].request_id, saves[1].request_id);
    assert.deepEqual(saves[0].data, saves[1].data);
    assert.equal(b.calls.some(c => c.operation === 'backend.provision'), false);
    assert.equal(b.draft.revision, 1);
}
finally {
    await close();
} });
test('protected service credential registration stays outside saved journey and authentication is distinct', async () => { const b = fixture(true); b.journey.state.current_stage = 'services'; const close = await mount(b); try {
    await click('Set up library');
    await fill('New assistance bearer token', 'service-secret-sentinel');
    await click('Store assistance credential and use its reference');
    const protectedCall = b.calls.find(c => c.operation === 'protected');
    assert.ok(protectedCall.args[1]);
    assert.equal(JSON.parse(protectedCall.input).value, 'service-secret-sentinel');
    assert.equal(field('New assistance bearer token').value, '');
    await click('Save stage and continue');
    assert.equal(JSON.stringify(b.calls.filter(c => c.operation !== 'protected')).includes('service-secret-sentinel'), false);
    assert.equal(JSON.stringify(b.journey).includes('service-secret-sentinel'), false);
}
finally {
    await close();
} });
test('exact pipeline selection survives reopen without configuration or preparation side effects', async () => { const b = fixture(true), original = b.Operate; const pid = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'; b.journey.state.current_stage = 'processing'; b.Operate = async (req) => req.operation === 'pipelines.list' ? { workspace_id: wid, result: { items: [{ id: pid, name: 'Existing elected route', revision: 4 }] } } : original(req); const close = await mount(b); try {
    await click('Set up library');
    await fill('Processing preference', 'connected');
    await fill('Setup pipeline', pid);
    await click('Save stage and continue');
    assert.deepEqual(b.journey.state.choices, { processing_mode: 'connected', pipeline_id: pid, pipeline_revision: 4 });
    assert.equal(b.journey.state.stages.find(s=>s.id==='processing').proof,proof);
    await click('Back');
    assert.equal(field('Setup pipeline').value, pid);
    assert.equal(b.calls.some(c => ['pipelines.set', 'workers.setup', 'models.prepare'].includes(c.operation)), false);
}
finally {
    await close();
} });
test('late setup response cannot replace a new workspace selection', async () => { const b = fixture(true), original = b.Operate; let release; b.Operate = async (req) => { if (req.operation === 'setup.save') {
    await new Promise(resolve => release = resolve);
} return original(req); }; const close = await mount(b); try {
    await click('Set up library');
    await click('Continue');
    await fill('Workspace path', 'C:/other');
    b.SelectWorkspace = async (path) => ({ result: { workspace_id: 'cccccccc-cccc-4ccc-8ccc-cccccccccccc', display_name: 'Other library', path } });
    await click('Open workspace');
    await act(async () => release());
    assert.match(document.body.textContent, /Selected: Other library/);
    assert.equal(document.body.textContent.includes('Backend profiles (Step 2 of 5)'), false);
}
finally {
    await close();
} });
test('bootstrap conflict readback restores current accepted profiles before a correction', async () => { const b = fixture(), close = await mount(b); try {
    await click('Save location and continue');
    const plan = structuredClone(b.draft.plan);
    plan.profiles.storage.configuration.root = 'accepted-other-client';
    b.draft = { ...b.draft, revision: b.draft.revision + 1, plan };
    await click('Read saved setup');
    assert.equal(field('storage root').value, 'accepted-other-client');
    await click('Prepare backend plan');
    assert.equal(b.calls.filter(c => c.operation === 'draft.save').at(-1).data.plan.profiles.storage.configuration.root, 'accepted-other-client');
}
finally {
    await close();
} });

test('known rejected location stays editable for keyboard error correction',async()=>{const b=fixture(),original=b.SetupDraft;b.SetupDraft=async(action,input,id)=>action==='save'&&JSON.parse(input).path==='C:/invalid'?{error:{code:'invalid_request',message:'Library location is inside the application installation. Choose another directory.'}}:original(action,input,id);const close=await mount(b);try{await fill('Library location','C:/invalid');await click('Save location and continue');assert.equal(document.activeElement.getAttribute('role'),'alert');assert.equal(field('Library location').disabled,false);assert.equal(document.body.textContent.includes('Retry the same save'),false);await fill('Library location','C:/valid');await click('Save location and continue');assert.match(document.activeElement.textContent,/Step 2 of 5/)}finally{await close()}});

test('nonsecret bootstrap credential backend restores independently of protected values',async()=>{const b=fixture(),close=await mount(b);await click('Save location and continue');await fill('Bootstrap credential storage','vault');await fill('Bootstrap vault passphrase','private-passphrase-sentinel');await click('Prepare backend plan');assert.equal(b.draft.credential_backend,'vault');assert.equal(JSON.stringify(b.draft).includes('private-passphrase-sentinel'),false);await close();const close2=await mount(b);try{assert.equal(field('Bootstrap credential storage').value,'vault');assert.equal(field('Bootstrap vault passphrase').value,'')}finally{await close2()}});

test('skipping first import finishes the journey without submitting work',async()=>{const b=fixture(true);b.journey.state.current_stage='import';b.journey.state.stages=b.journey.state.stages.map(s=>({...s,state:s.id==='import'?'pending':'skipped'}));const close=await mount(b);try{await click('Set up library');await click('Skip this stage');assert.equal(b.journey.state.status,'completed');assert.equal(b.journey.state.stages.find(s=>s.id==='import').state,'skipped');assert.equal(b.calls.some(c=>c.operation==='media.import'),false);assert.ok(field('Media path or URL'))}finally{await close()}});

test('host restart opens saved provisioned library and resumes authoritative journey without reprovisioning',async()=>{const b=fixture(),close=await mount(b);await click('Save location and continue');await click('Create configured workspace');await click('Skip this stage');await close();b.selected=false;b.draft.location={existing_workspace:true,path:b.draft.path};const provisionCount=b.calls.filter(c=>c.operation==='backend.provision').length;const close2=await mount(b);try{assert.match(document.body.textContent,/Step 2 of 5/);assert.equal([...document.querySelectorAll('button')].some(b=>b.textContent==='Create configured workspace'),false);await click('Open saved library and resume');assert.match(document.activeElement.textContent,/Step 4 of 5/);assert.match(document.body.textContent,/Selected: Voice archive/);assert.equal(b.calls.filter(c=>c.operation==='backend.provision').length,provisionCount);assert.equal(b.calls.some(c=>c.operation==='setup.show'),true)}finally{await close2()}});
test('choosing a different new library drops normalized workspace and profile identities',async()=>{const b=fixture(),close=await mount(b);try{await click('Save location and continue');b.draft.plan.credential_namespace_id='eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee';for(const p of Object.values(b.draft.plan.profiles)){p.profile_id='ffffffff-ffff-4fff-8fff-ffffffffffff';p.profile_revision=1}await click('Read saved setup');await click('Back');await fill('Library location','C:/Users/fixture/Second archive');await click('Save location and continue');const saved=b.calls.filter(c=>c.operation==='draft.save').at(-1);assert.equal(saved.data.plan.workspace_id,undefined);assert.equal(saved.data.plan.credential_namespace_id,undefined);for(const profile of Object.values(saved.data.plan.profiles)){assert.equal(profile.profile_id,undefined);assert.equal(profile.profile_revision,undefined)}assert.equal(saved.data.plan.profiles.storage.adapter_id,'filesystem');assert.equal(b.calls.some(c=>c.operation==='backend.provision'),false)}finally{await close()}});

test('late workspace-change load cannot regress accepted setup state after provisioning',async()=>{const b=fixture(),original=b.Operate;const reads=[];b.Operate=async req=>{if(req.operation==='setup.show'){const snapshot=await original(req);return new Promise(resolve=>reads.push(()=>resolve(snapshot)))}return original(req)};const close=await mount(b);try{await click('Save location and continue');await click('Create configured workspace');assert.equal(reads.length,2,'configured readback and workspace-change effect should both observe pre-save state');await act(async()=>reads[0]());assert.equal(b.journey.revision,1);assert.match(document.body.textContent,/Step 3 of 5/);await act(async()=>reads[1]());assert.match(document.body.textContent,/Step 3 of 5/);b.Operate=original;await click('Save stage and continue');assert.equal(b.journey.revision,2);assert.match(document.body.textContent,/Step 4 of 5/);assert.equal(document.body.textContent.includes('Setup changed.'),false)}finally{await close()}});
test('Open import retains ordinary import choices; setup pipeline only elects readiness',async()=>{const b=fixture(true);const pid='bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb';b.journey.state.current_stage='import';b.journey.state.choices={processing_mode:'connected',pipeline_id:pid,pipeline_revision:4};const close=await mount(b);try{await click('Set up library');assert.ok(b.calls.some(c=>c.operation==='setup.readiness'));await click('Open import');assert.equal(field('Saved import preset').value,'');await fill('Media path or URL','C:/fixture.wav');await click('Import');const req=b.calls.find(c=>c.operation==='media.import');assert.equal(req.data.defaults.preset,undefined);assert.equal(JSON.stringify(req.data).includes(pid),false);assert.equal(b.calls.some(c=>c.operation==='import-presets.set'),false)}finally{await close()}});
