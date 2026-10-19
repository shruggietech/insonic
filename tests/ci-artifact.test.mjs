import test from 'node:test';
import assert from 'node:assert/strict';
import { waitForArtifact, waitForDesktopProofs, waitForDesktopInput } from '../scripts/wait-ci-artifact.mjs';

const options = { repository: 'shruggietech/insonic', run: '123', attempt: '2', os: 'windows-2022', token: 'fixture' };
const artifact = { id: 42, name: 'media-source-windows-2022-2', expired: false, digest: 'sha256:' + 'a'.repeat(64), workflow_run: { id: 123 } };
const response = artifacts => ({ total_count: artifacts.length, artifacts });
const gates = { total_count: 2, jobs: ['foundation', 'docs-build'].map(name => ({ name, status: 'completed', conclusion: 'success' })) };
const api = artifacts => async url => url.includes('/jobs?') ? gates : artifacts(url);

test('source handoff waits for this exact run and attempt without downloading bytes', async () => {
  let calls = 0, elapsed = 0;
  const value = await waitForArtifact({ ...options, now: () => elapsed, sleep: async ms => { elapsed += ms; },
    request: api(async url => { assert.match(url, /^https:\/\/api.github.com\/repos\/shruggietech\/insonic\/actions\/runs\/123\/artifacts\?/); return response(++calls === 1 ? [{ ...artifact, name: 'media-source-windows-2022-1' }] : [artifact]); }) });
  assert.equal(value, 42); assert.equal(calls, 2); assert.equal(elapsed, 5000);
});

test('ambiguous, expired, foreign and unbounded handoffs fail closed', async () => {
  for (const data of [response([artifact, artifact]), response([{ ...artifact, expired: true }]), response([{ ...artifact, workflow_run: { id: 999 } }]), { ...response([]), total_count: 101 }]) {
    await assert.rejects(waitForArtifact({ ...options, request: api(async () => data) }), /artifact/);
  }
});

test('missing producer has one aggregate deadline, no stale fallback', async () => {
  let elapsed = 0;
  await assert.rejects(waitForArtifact({ ...options, deadlineMs: 10000, now: () => elapsed, sleep: async ms => { elapsed += ms; }, request: api(async () => response([])) }), /deadline/);
  assert.equal(elapsed, 10000);
  await assert.rejects(waitForArtifact({ ...options, run: '../other', request: async () => { throw new Error('must not request'); } }), /identity/);
});

test('early setup cannot consume artifacts before exact-attempt integrity and documentation success', async () => {
  for (const outcome of ['failure', 'cancelled', 'skipped']) {
    const failed = structuredClone(gates); failed.jobs[0].conclusion = outcome;
    await assert.rejects(waitForArtifact({ ...options, request: async url => url.includes('/jobs?') ? failed : response([artifact]) }), /qualification gate/);
  }
  let elapsed = 0;
  const pending = structuredClone(gates); pending.jobs[1].status = 'in_progress'; pending.jobs[1].conclusion = null;
  assert.equal(await waitForArtifact({ ...options, now: () => elapsed, sleep: async ms => { elapsed += ms; }, request: async url => {
    if (url.includes('/jobs?')) { assert.match(url, /\/attempts\/2\/jobs\?/); return elapsed ? gates : pending; }
    return response([artifact]);
  } }), 42);
  assert.equal(elapsed, 5000);
});

const proofOptions = { ...options, revision: 'a'.repeat(40) };
const scopes = ['archive', 'interactive', 'maintenance', 'setup'];
const proofArtifacts = scopes.map((scope, i) => ({ ...artifact, id: 51 + i,
  name: `windows-desktop-proof-${scope}-${proofOptions.revision}-2` }));
const proofGates = { total_count: 4, jobs: scopes.map(scope => ({
  name: `package-windows / windows-desktop / Qualify Windows desktop (${scope})`, status: 'completed', conclusion: 'success',
})) };
const proofApi = (artifacts = proofArtifacts, jobs = proofGates) => async url => url.includes('/jobs?') ? jobs : response(artifacts);

test('early Windows preparation waits for all four successful exact-attempt producers', async () => {
  let elapsed = 0;
  const pending = structuredClone(proofGates); pending.jobs[2].status = 'in_progress'; pending.jobs[2].conclusion = null;
  const value = await waitForDesktopProofs({ ...proofOptions, now: () => elapsed, sleep: async ms => { elapsed += ms; }, request: async url => {
    if (url.includes('/jobs?')) { assert.match(url, /\/attempts\/2\/jobs\?/); return elapsed ? proofGates : pending; }
    assert.match(url, /\/runs\/123\/artifacts\?/); return response(proofArtifacts);
  } });
  assert.deepEqual(value, { archive: 51, interactive: 52, maintenance: 53, setup: 54 });
  assert.equal(elapsed, 5000);
});

test('proof availability cannot override failed, cancelled or skipped native qualification', async () => {
  for (const scope of scopes) for (const conclusion of ['failure', 'cancelled', 'skipped']) {
    const jobs = structuredClone(proofGates); jobs.jobs[scopes.indexOf(scope)].conclusion = conclusion;
    await assert.rejects(waitForDesktopProofs({ ...proofOptions, request: proofApi(proofArtifacts, jobs) }), /proof failed/);
  }
});

test('Windows proof readiness rejects ambiguous, foreign, expired and incomplete inventories', async () => {
  for (const change of [item => ({ ...item, workflow_run: { id: 999 } }), item => ({ ...item, expired: true }),
    item => ({ ...item, digest: null }), item => ({ ...item, id: -1 })]) {
    await assert.rejects(waitForDesktopProofs({ ...proofOptions, request: proofApi([change(proofArtifacts[0]), ...proofArtifacts.slice(1)]) }), /authority/);
  }
  await assert.rejects(waitForDesktopProofs({ ...proofOptions, request: proofApi([...proofArtifacts, proofArtifacts[0]]) }), /Ambiguous/);
  await assert.rejects(waitForDesktopProofs({ ...proofOptions, request: proofApi(proofArtifacts, { total_count: 5, jobs: [...proofGates.jobs, proofGates.jobs[0]] }) }), /Ambiguous/);
  await assert.rejects(waitForDesktopProofs({ ...proofOptions, request: async () => ({ total_count: 101, artifacts: [], jobs: [] }) }), /Incomplete/);
});

test('Windows proof wait has one deadline and cannot reuse another revision or attempt', async () => {
  for (const artifacts of [[], proofArtifacts.slice(1), proofArtifacts.map(item => ({ ...item, name: item.name.replace('-2', '-1') })),
    proofArtifacts.map(item => ({ ...item, name: item.name.replace(proofOptions.revision, 'b'.repeat(40)) }))]) {
    let elapsed = 0;
    await assert.rejects(waitForDesktopProofs({ ...proofOptions, deadlineMs: 10000, now: () => elapsed,
      sleep: async ms => { elapsed += ms; }, request: proofApi(artifacts) }), /deadline/);
    assert.equal(elapsed, 10000);
  }
  for (const override of [{ revision: '../other' }, { attempt: '0' }, { deadlineMs: 400001 }])
    await assert.rejects(waitForDesktopProofs({ ...proofOptions, ...override, request: async () => { throw new Error('must not request'); } }), /identity/);
});


test('failed Windows producer retains exact bounded identity before refusing the join',async()=>{
 const jobs=structuredClone(proofGates);jobs.jobs[0]={...jobs.jobs[0],id:1234,conclusion:'failure'};
 const observations=[];
 await assert.rejects(waitForDesktopProofs({...proofOptions,request:proofApi(proofArtifacts,jobs),observeFailure:async value=>observations.push(value)}),/proof failed: archive/);
 assert.deepEqual(observations,[{kind:'windows-proof-producer-failure',repository:options.repository,run:'123',attempt:'2',revision:proofOptions.revision,scope:'archive',job_id:1234,status:'completed',conclusion:'failure'}]);
 assert.equal(JSON.stringify(observations).includes(options.token),false);
});


test('a diagnostic sink error cannot replace the failed Windows producer',async()=>{
 const jobs=structuredClone(proofGates);jobs.jobs[0].conclusion='failure';
 await assert.rejects(waitForDesktopProofs({...proofOptions,request:proofApi(proofArtifacts,jobs),observeFailure:async()=>{throw new Error('diagnostic write denied');}}),/Required Windows proof failed: archive/);
});

const inputArtifact = { ...artifact, name: `windows-desktop-input-${proofOptions.revision}-2` };
const inputGate = { id: 24, name: 'package-windows / windows-desktop / Assemble Windows desktop delivery', status: 'completed', conclusion: 'success' };
const inputApi = (artifacts = [inputArtifact], jobs = [inputGate]) => async url => url.includes('/jobs?') ? { total_count: jobs.length, jobs } : response(artifacts);

test('early desktop input setup waits for successful assembly and exact artifact authority', async () => {
  let elapsed = 0;
  const result = await waitForDesktopInput({ ...proofOptions, now: () => elapsed, sleep: async ms => { elapsed += ms; }, request: async url => {
    if (url.includes('/jobs?')) {
      assert.match(url, /attempts\/2\/jobs/);
      return { total_count: elapsed ? 1 : 0, jobs: elapsed ? [{ ...inputGate, status: elapsed >= 10000 ? 'completed' : 'in_progress', conclusion: elapsed >= 10000 ? 'success' : null }] : [] };
    }
    return response([inputArtifact]);
  } });
  assert.equal(result, 42);
  assert.equal(elapsed, 10000);
});

test('an uploaded input cannot override failed or cancelled assembly and retains its primary cause', async () => {
  for (const conclusion of ['failure', 'cancelled', 'skipped', 'timed_out']) {
    const observations = [];
    await assert.rejects(waitForDesktopInput({ ...proofOptions, request: inputApi([inputArtifact], [{ ...inputGate, conclusion }]),
      observeFailure: async value => observations.push(value) }), /input failed: assembly/);
    assert.equal(observations.length, 1);
    assert.equal(observations[0].kind, 'windows-input-producer-failure');
    assert.equal(observations[0].job_id, 24);
    assert.equal(observations[0].conclusion, conclusion);
    assert.ok(!JSON.stringify(observations).includes(options.token));
  }
  await assert.rejects(waitForDesktopInput({ ...proofOptions, request: inputApi([], [{ ...inputGate, conclusion: 'failure' }]),
    observeFailure: async () => { throw new Error('diagnostic failure'); } }), /input failed: assembly/);
});

test('desktop input rejects ambiguous producers, expired or foreign artifacts and partial inventories', async () => {
  for (const change of [item => ({ ...item, expired: true }), item => ({ ...item, workflow_run: { id: 999 } }),
    item => ({ ...item, digest: null }), item => ({ ...item, id: -1 })]) {
    await assert.rejects(waitForDesktopInput({ ...proofOptions, request: inputApi([change(inputArtifact)]) }), /authority/);
  }
  await assert.rejects(waitForDesktopInput({ ...proofOptions, request: inputApi([inputArtifact, inputArtifact]) }), /Ambiguous/);
  await assert.rejects(waitForDesktopInput({ ...proofOptions, request: inputApi([inputArtifact], [inputGate, inputGate]) }), /Ambiguous/);
  await assert.rejects(waitForDesktopInput({ ...proofOptions, request: async () => ({ total_count: 101, artifacts: [], jobs: [] }) }), /Incomplete/);
});

test('desktop input cannot reuse another revision or attempt and has one bounded deadline', async () => {
  for (const artifacts of [[], [{ ...inputArtifact, name: inputArtifact.name.replace('-2', '-1') }],
    [{ ...inputArtifact, name: inputArtifact.name.replace(proofOptions.revision, 'b'.repeat(40)) }]]) {
    let elapsed = 0;
    await assert.rejects(waitForDesktopInput({ ...proofOptions, deadlineMs: 10000, now: () => elapsed,
      sleep: async ms => { elapsed += ms; }, request: inputApi(artifacts) }), /deadline/);
    assert.equal(elapsed, 10000);
  }
  for (const override of [{ revision: '../other' }, { attempt: '0' }, { deadlineMs: 400001 }]) {
    await assert.rejects(waitForDesktopInput({ ...proofOptions, ...override, request: async () => { throw new Error('must not request'); } }), /identity/);
  }
});
