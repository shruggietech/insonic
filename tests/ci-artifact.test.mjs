import test from 'node:test';
import assert from 'node:assert/strict';
import http from 'node:http';
import { waitForArtifact, waitForDesktopProofs, waitForDesktopInput, waitForWindowsPackages } from '../scripts/wait-ci-artifact.mjs';

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

test('healthy source and Windows joins can finish beyond their former cutoffs', async () => {
  for (const scenario of [
    { wait: waitForArtifact, options, former: 180000, value: [artifact], ready: gates },
    { wait: waitForDesktopProofs, options: proofOptions, former: 400000, value: proofArtifacts, ready: proofGates },
  ]) {
    let elapsed = 0;
    await scenario.wait({ ...scenario.options, now: () => elapsed, sleep: async ms => { elapsed += ms; },
      request: async url => url.includes('/jobs?') ? scenario.ready : response(elapsed > scenario.former ? scenario.value : []) });
    assert.ok(elapsed > scenario.former);
    assert.ok(elapsed < (scenario.former === 180000 ? 900000 : 3600000));
  }
});

test('expanded artifact waits still expire at their own limit or an earlier original phase deadline', async () => {
  for (const scenario of [
    { wait: waitForArtifact, options, limit: 900000, ready: gates },
    { wait: waitForDesktopProofs, options: proofOptions, limit: 3600000, ready: proofGates },
  ]) for (const phaseDeadlineMs of [undefined, 7500]) {
    let elapsed = 0;
    await assert.rejects(scenario.wait({ ...scenario.options, phaseDeadlineMs, now: () => elapsed,
      sleep: async ms => { elapsed += ms; }, request: async url => url.includes('/jobs?') ? scenario.ready : response([]) }), /deadline/);
    assert.equal(elapsed, phaseDeadlineMs ?? scenario.limit);
  }
});

test('a ready response after the original deadline never authorizes artifact consumption', async () => {
  for (const scenario of [
    { wait: waitForArtifact, options, value: [artifact], ready: gates },
    { wait: waitForDesktopProofs, options: proofOptions, value: proofArtifacts, ready: proofGates },
  ]) {
    let elapsed = 0;
    await assert.rejects(scenario.wait({ ...scenario.options, phaseDeadlineMs: 1000, now: () => elapsed,
      request: async url => { elapsed = 1001; return url.includes('/jobs?') ? scenario.ready : response(scenario.value); } }), /deadline/);
  }
});

test('explicit cancellation remains earlier than the expanded artifact allowance', async () => {
  for (const scenario of [
    { wait: waitForArtifact, options, ready: gates },
    { wait: waitForDesktopProofs, options: proofOptions, ready: proofGates },
  ]) {
    const controller = new AbortController(); let elapsed = 0;
    await assert.rejects(scenario.wait({ ...scenario.options, signal: controller.signal, now: () => elapsed,
      sleep: async ms => { elapsed += ms; controller.abort(new Error('owner cancelled')); },
      request: async url => url.includes('/jobs?') ? scenario.ready : response([]) }), /owner cancelled/);
    assert.equal(elapsed, 5000);
  }
});

test('invalid expanded overrides and shared deadlines fail before network access', async () => {
  for (const scenario of [
    { wait: waitForArtifact, options, limit: 900000 },
    { wait: waitForDesktopProofs, options: proofOptions, limit: 3600000 },
  ]) for (const override of [{ deadlineMs: scenario.limit + 1 }, { deadlineMs: NaN }, { deadlineMs: 1.5 },
    { phaseDeadlineMs: 0 }, { phaseDeadlineMs: NaN }]) {
    await assert.rejects(scenario.wait({ ...scenario.options, ...override,
      request: async () => { throw new Error('must not request'); } }), /identity|deadline/);
  }
});

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
  for (const override of [{ revision: '../other' }, { attempt: '0' }, { deadlineMs: 3600001 }])
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
  for (const override of [{ revision: '../other' }, { attempt: '0' }, { deadlineMs: 3600001 }]) {
    await assert.rejects(waitForDesktopInput({ ...proofOptions, ...override, request: async () => { throw new Error('must not request'); } }), /identity/);
  }
});

const packageScopes = ['cli', 'desktop'];
const packageArtifacts = packageScopes.map((scope, index) => ({ ...artifact, id: 71 + index, name: `candidate-package-windows-2022-${scope}-${proofOptions.revision}-2` }));
const packageJobs = [{ id: 31, name: 'package-windows / package (windows-2022, cli)', status: 'completed', conclusion: 'success' },
  { id: 32, name: 'package-windows / windows-desktop / Join exact-byte Windows desktop qualification', status: 'completed', conclusion: 'success' }];
const packageApi = (artifacts = packageArtifacts, jobs = packageJobs) => async url => url.includes('/jobs?') ? { total_count: jobs.length, jobs } : response(artifacts);

test('candidate preparation awaits both successful Windows package producers and exact artifacts', async () => {
  let elapsed = 0;
  const pending = structuredClone(packageJobs); pending[1].status = 'in_progress'; pending[1].conclusion = null;
  const ready = await waitForWindowsPackages({ ...proofOptions, now: () => elapsed, sleep: async ms => { elapsed += ms; },
    request: async url => url.includes('/jobs?') ? { total_count: 2, jobs: elapsed ? packageJobs : pending } : response(packageArtifacts) });
  assert.deepEqual(ready, { cli: 71, desktop: 72 });
  assert.equal(elapsed, 5000);
});

test('candidate preparation cannot consume a failed Windows CLI or desktop join', async () => {
  for (const index of [0, 1]) for (const conclusion of ['failure', 'cancelled', 'skipped']) {
    const jobs = structuredClone(packageJobs); jobs[index].conclusion = conclusion;
    const observations = [];
    await assert.rejects(waitForWindowsPackages({ ...proofOptions, request: packageApi(packageArtifacts, jobs), observeFailure: async value => observations.push(value) }),
      new RegExp('packages failed: ' + packageScopes[index]));
    assert.equal(observations[0].kind, 'windows-packages-producer-failure');
    assert.equal(observations[0].scope, packageScopes[index]);
  }
});

test('candidate package readiness rejects another run, revision, attempt or incomplete tuple', async () => {
  await assert.rejects(waitForWindowsPackages({ ...proofOptions, request: packageApi([{ ...packageArtifacts[0], workflow_run: { id: 999 } }, packageArtifacts[1]]) }), /authority/);
  for (const artifacts of [packageArtifacts.slice(0, 1), packageArtifacts.map(item => ({ ...item, name: item.name.replace('-2', '-1') })),
    packageArtifacts.map(item => ({ ...item, name: item.name.replace(proofOptions.revision, 'b'.repeat(40)) }))]) {
    let elapsed = 0;
    await assert.rejects(waitForWindowsPackages({ ...proofOptions, deadlineMs: 10000, now: () => elapsed, sleep: async ms => { elapsed += ms; }, request: packageApi(artifacts) }), /deadline/);
    assert.equal(elapsed, 10000);
  }
});

const windowsReadinessCases = [
  { wait: waitForDesktopInput, artifacts: [inputArtifact], jobs: [inputGate], expected: 42, scope: 'assembly' },
  { wait: waitForDesktopProofs, artifacts: proofArtifacts, jobs: proofGates.jobs, expected: { archive: 51, interactive: 52, maintenance: 53, setup: 54 }, scope: 'archive' },
  { wait: waitForWindowsPackages, artifacts: packageArtifacts, jobs: packageJobs, expected: { cli: 71, desktop: 72 }, scope: 'cli' },
];

const metadataReadCases = [{ wait: waitForArtifact, artifacts: [artifact], jobs: gates.jobs, expected: 42 },
  ...windowsReadinessCases];

test('transient HTTP metadata failures recover for every source and Windows join within the same deadline', async t => {
  for (const scenario of metadataReadCases) for (const dimension of ['jobs', 'artifacts']) {
    let elapsed = 0;
    const calls = { jobs: 0, artifacts: 0 };
    const mocked = t.mock.method(globalThis, 'fetch', async url => {
      const key = url.includes('/jobs?') ? 'jobs' : 'artifacts';
      if (++calls[key] <= 2 && key === dimension) return new Response('transient provider failure', { status: 502 });
      return Response.json({ total_count: scenario[key].length, [key]: scenario[key] });
    });
    try {
      const ready = await scenario.wait({ ...proofOptions, phaseDeadlineMs: 5000, now: () => elapsed,
        sleep: async ms => { elapsed += ms; } });
      assert.deepEqual(ready, scenario.expected);
      assert.equal(calls[dimension], 3);
      assert.equal(calls[dimension === 'jobs' ? 'artifacts' : 'jobs'], 1);
      assert.equal(elapsed, 3000);
    } finally { mocked.mock.restore(); }
  }
});

test('persistent transient metadata failure stops after three attempts', async t => {
  let elapsed = 0, calls = 0;
  t.mock.method(globalThis, 'fetch', async url => url.includes('/jobs?')
    ? Response.json({ total_count: packageJobs.length, jobs: packageJobs })
    : (++calls, new Response('transient provider failure', { status: 502 })));
  await assert.rejects(waitForWindowsPackages({ ...proofOptions, now: () => elapsed,
    sleep: async ms => { elapsed += ms; } }), /HTTP 502/);
  assert.equal(calls, 3);
  assert.equal(elapsed, 3000);
});

test('authentication, permanent HTTP errors and malformed metadata are never retried', async t => {
  for (const status of [401, 403, 404, 409, 422, 501, 200]) {
    let calls = 0;
    const mocked = t.mock.method(globalThis, 'fetch', async url => url.includes('/jobs?')
      ? Response.json({ total_count: packageJobs.length, jobs: packageJobs })
      : (++calls, new Response('not JSON and not an accepted inventory', { status })));
    try {
      await assert.rejects(waitForWindowsPackages({ ...proofOptions,
        sleep: async () => assert.fail('permanent failure must not retry') }));
      assert.equal(calls, 1);
    } finally { mocked.mock.restore(); }
  }
});

test('retry backoff cannot restart or overrun the earlier qualification deadline', async t => {
  let elapsed = 0, calls = 0;
  t.mock.method(globalThis, 'fetch', async url => url.includes('/jobs?')
    ? Response.json({ total_count: packageJobs.length, jobs: packageJobs })
    : (++calls, new Response('transient provider failure', { status: 503 })));
  await assert.rejects(waitForWindowsPackages({ ...proofOptions, phaseDeadlineMs: 500,
    now: () => elapsed, sleep: async ms => { elapsed += ms; } }), /deadline/);
  assert.equal(calls, 1);
  assert.equal(elapsed, 500);
});

test('a transient fetch transport failure can recover but caller cancellation cannot retry', async t => {
  for (const cancelled of [false, true]) {
    const controller = new AbortController(); let calls = 0, elapsed = 0;
    const mocked = t.mock.method(globalThis, 'fetch', async url => {
      if (url.includes('/jobs?')) return Response.json({ total_count: packageJobs.length, jobs: packageJobs });
      if (++calls === 1) {
        if (cancelled) controller.abort(new Error('owner cancelled'));
        throw new TypeError('fetch failed', { cause: Object.assign(new Error('connection reset'), { code: 'ECONNRESET' }) });
      }
      return Response.json(response(packageArtifacts));
    });
    try {
      const operation = waitForWindowsPackages({ ...proofOptions, signal: controller.signal,
        now: () => elapsed, sleep: async ms => { elapsed += ms; } });
      if (cancelled) {
        await assert.rejects(operation, /owner cancelled/);
        assert.equal(calls, 1); assert.equal(elapsed, 0);
      } else {
        assert.deepEqual(await operation, { cli: 71, desktop: 72 });
        assert.equal(calls, 2); assert.equal(elapsed, 1000);
      }
    } finally { mocked.mock.restore(); }
  }
});

test('a permanent metadata failure cancels a sibling read before another retry', async t => {
  let siblingSignal, releaseBackoff;
  const controller = new AbortController();
  t.mock.method(globalThis, 'fetch', async (url, init) => {
    if (!url.includes('/jobs?')) return new Response('not authorized', { status: 401 });
    siblingSignal = init.signal;
    throw new TypeError('fetch failed', { cause: Object.assign(new Error('connection reset'), { code: 'ECONNRESET' }) });
  });
  try {
    await assert.rejects(waitForWindowsPackages({ ...proofOptions, signal: controller.signal,
      sleep: async (ms, unused, { signal }) => new Promise((resolve, reject) => {
        releaseBackoff = resolve;
        if (signal.aborted) reject(signal.reason);
        else signal.addEventListener('abort', () => reject(signal.reason), { once: true });
      }) }), /HTTP 401/);
    assert.equal(siblingSignal.aborted, true);
  } finally { controller.abort(new Error('fixture cleanup')); releaseBackoff?.(); }
});

test('TLS, refused redirects and unclassified fetch errors are not transient retries', async t => {
  for (const cause of [{ code: 'CERT_HAS_EXPIRED' }, { code: 'ERR_TLS_CERT_ALTNAME_INVALID' },
    new Error('unexpected redirect'), undefined]) {
    let calls = 0;
    const mocked = t.mock.method(globalThis, 'fetch', async url => {
      if (url.includes('/jobs?')) return Response.json({ total_count: packageJobs.length, jobs: packageJobs });
      calls++;
      throw new TypeError('fetch failed', { cause });
    });
    try {
      await assert.rejects(waitForWindowsPackages({ ...proofOptions,
        sleep: async () => assert.fail('unclassified failure must not retry') }), /fetch failed/);
      assert.equal(calls, 1);
    } finally { mocked.mock.restore(); }
  }
});

test('a real socket drop after successful headers retries the discarded partial inventory body', async t => {
  let artifactRequests = 0, partialResponse, headersObserved = false, elapsed = 0;
  const server = http.createServer((request, reply) => {
    if (request.url === '/jobs') {
      reply.setHeader('Content-Type', 'application/json'); reply.end(JSON.stringify(gates));
    } else if (++artifactRequests === 1) {
      partialResponse = reply;
      reply.writeHead(200, { 'Content-Type': 'application/json' });
      reply.flushHeaders(); reply.write('{"total_count":');
    } else {
      reply.setHeader('Content-Type', 'application/json'); reply.end(JSON.stringify(response([artifact])));
    }
  });
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
  t.after(async () => { server.closeAllConnections(); await new Promise(resolve => server.close(resolve)); });
  const nativeFetch = globalThis.fetch;
  t.mock.method(globalThis, 'fetch', async (url, init) => {
    const route = url.includes('/jobs?') ? '/jobs' : '/artifacts';
    const received = await nativeFetch(`http://127.0.0.1:${server.address().port}${route}`, init);
    if (route === '/artifacts' && artifactRequests === 1) {
      assert.equal(received.status, 200); headersObserved = true; partialResponse.destroy();
    }
    return received;
  });
  assert.equal(await waitForArtifact({ ...options, phaseDeadlineMs: 60000, now: () => elapsed,
    sleep: async ms => { elapsed += ms; } }), 42);
  assert.equal(headersObserved, true);
  assert.equal(artifactRequests, 2);
  assert.equal(elapsed, 1000);
});

test('oversized inventory bodies and cancellation during body reads cannot retry', async t => {
  for (const cancelled of [false, true]) {
    let calls = 0;
    const controller = new AbortController();
    const mocked = t.mock.method(globalThis, 'fetch', async url => {
      if (url.includes('/jobs?')) return Response.json(gates);
      calls++;
      if (!cancelled) return new Response('x'.repeat((1 << 20) + 1));
      return new Response(new ReadableStream({ pull(stream) {
        controller.abort(new Error('owner cancelled during body'));
        stream.error(new TypeError('terminated', { cause: { code: 'UND_ERR_SOCKET' } }));
      } }));
    });
    try {
      await assert.rejects(waitForArtifact({ ...options, signal: controller.signal,
        sleep: async () => assert.fail('body refusal cannot retry') }), cancelled ? /owner cancelled/ : /bound/);
      assert.equal(calls, 1);
    } finally { mocked.mock.restore(); }
  }
});

test('Windows readiness waits for complete consistent inventories after scheduling and upload races', async () => {
  for (const scenario of windowsReadinessCases) for (const dimension of ['jobs', 'artifacts']) for (const difference of [-1, 1]) {
    let elapsed = 0;
    const ready = await scenario.wait({ ...proofOptions, now: () => elapsed, sleep: async ms => { elapsed += ms; }, request: async url => {
      const key = url.includes('/jobs?') ? 'jobs' : 'artifacts';
      const items = scenario[key];
      return { total_count: items.length + (!elapsed && key === dimension ? difference : 0), [key]: items };
    } });
    assert.deepEqual(ready, scenario.expected);
    assert.equal(elapsed, 5000, dimension + ' inconsistency must not authorize consumption');
  }
});

test('persistent partial inventories cannot authorize Windows consumption and retain the aggregate deadline', async () => {
  for (const scenario of windowsReadinessCases) for (const dimension of ['jobs', 'artifacts']) {
    let elapsed = 0;
    await assert.rejects(scenario.wait({ ...proofOptions, deadlineMs: 10000, now: () => elapsed, sleep: async ms => { elapsed += ms; }, request: async url => {
      const key = url.includes('/jobs?') ? 'jobs' : 'artifacts';
      return { total_count: scenario[key].length + (key === dimension ? 1 : 0), [key]: scenario[key] };
    } }), /deadline exceeded.*incomplete/i);
    assert.equal(elapsed, 10000);
  }
});

test('malformed or oversized Windows inventories fail immediately without a retry', async () => {
  for (const scenario of windowsReadinessCases) for (const dimension of ['jobs', 'artifacts']) for (const count of [-1, 0.5, '1', null, 101]) {
    await assert.rejects(scenario.wait({ ...proofOptions, sleep: async () => { assert.fail('malformed inventory must fail immediately'); }, request: async url => {
      const key = url.includes('/jobs?') ? 'jobs' : 'artifacts';
      return { total_count: key === dimension ? count : scenario[key].length, [key]: scenario[key] };
    } }), /Incomplete Windows/);
  }
});

test('a known failed Windows producer stays primary even in a partial inventory', async () => {
  for (const scenario of windowsReadinessCases) {
    const observations = [];
    await assert.rejects(scenario.wait({ ...proofOptions, sleep: async () => { assert.fail('known producer failure must not wait'); },
      observeFailure: async observation => observations.push(observation), request: async url => {
        const key = url.includes('/jobs?') ? 'jobs' : 'artifacts';
        const items = structuredClone(scenario[key]);
        if (key === 'jobs') items[0].conclusion = 'failure';
        return { total_count: items.length + 1, [key]: items };
      } }), new RegExp('failed: ' + scenario.scope));
    assert.equal(observations.length, 1);
    assert.equal(observations[0].scope, scenario.scope);
    assert.equal(observations[0].conclusion, 'failure');
  }
});

test('Windows producer joins permit proofs beyond the advisory target under the original hard deadline', async () => {
  for (const scenario of [
    { wait: waitForDesktopProofs, artifacts: proofArtifacts, jobs: proofGates.jobs, expected: { archive: 51, interactive: 52, maintenance: 53, setup: 54 } },
    { wait: waitForDesktopInput, artifacts: [inputArtifact], jobs: [inputGate], expected: 42 },
    { wait: waitForWindowsPackages, artifacts: packageArtifacts, jobs: packageJobs, expected: { cli: 71, desktop: 72 } }
  ]) {
    let elapsed = 0;
    const ready = await scenario.wait({ ...proofOptions, phaseDeadlineMs: 3600000, now: () => elapsed,
      sleep: async ms => { elapsed += ms; }, request: async url => url.includes('/jobs?')
        ? { total_count: scenario.jobs.length, jobs: scenario.jobs }
        : response(elapsed >= 3005000 ? scenario.artifacts : []) });
    assert.deepEqual(ready, scenario.expected);
    assert.equal(elapsed, 3005000);
    assert.ok(elapsed + 60000 < 3600000, 'Original finalization reserve remains available');
  }
});
