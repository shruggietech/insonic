// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { parse } from 'yaml';
import { waitForArtifact } from '../scripts/wait-ci-artifact.mjs';

const workflow = name => parse(readFileSync(new URL(`../.github/workflows/${name}.yml`, import.meta.url), 'utf8'));
const deps = job => [job.needs ?? []].flat();

// Follow the production DAG, rather than a second scheduling implementation.
// A consumer inherits every successful prerequisite's completion timestamp.
function earliest(jobs, id, durations, failed = new Set(), seen = new Set()) {
  assert.ok(jobs[id], `unknown production job ${id}`);
  assert.ok(!seen.has(id), `cycle at ${id}`);
  if (failed.has(id)) return Infinity;
  const next = new Set([...seen, id]);
  return Math.max(0, ...deps(jobs[id]).map(parent => earliest(jobs, parent, durations, failed, next))) + (durations[id] ?? 0);
}

test('actual short artifact helper still rejects a valid cold producer at exactly 180 seconds', async () => {
  let clock = 0;
  await assert.rejects(waitForArtifact({ repository: 'shruggietech/insonic', run: '42', attempt: '1',
    os: 'windows-2022', token: 'test-only', now: () => clock, sleep: async ms => { clock += ms; },
    request: async url => url.includes('/jobs?')
      ? { total_count: 2, jobs: ['foundation', 'docs-build'].map(name => ({ name, status: 'completed', conclusion: 'success' })) }
      : { total_count: 0, artifacts: [] },
  }), /Same-run source artifact deadline exceeded/);
  assert.equal(clock, 180000);
});

for (const preparationMs of [5000, 1342000]) {
  test(`actual workflow waits for verified inputs with ${preparationMs}ms preparation`, async () => {
    const jobs = workflow('ci').jobs;
    const durations = { 'media-windows': preparationMs, 'media-linux': 3000, foundation: 22000, 'docs-build': 185000 };
    const prepared = Math.max(earliest(jobs, 'media-windows', durations), earliest(jobs, 'media-linux', durations));
    for (const id of ['foundation', 'docs-build', 'core', 'core-windows', 'qualify-linux', 'qualify-windows', 'package-linux', 'package-windows', 'adapters', 'candidate', 'live-site']) {
      const start = earliest(jobs, id, durations) - (durations[id] ?? 0);
      assert.ok(start >= prepared, `${id} began before verified preparation`);
    }
    for (const id of ['package-linux']) {
      assert.ok(earliest(jobs, id, durations) >= earliest(jobs, 'docs-build', durations), `${id} outran documentation`);
      assert.ok(earliest(jobs, id, durations) >= earliest(jobs, 'foundation', durations), `${id} outran foundation`);
    }
    assert.equal(jobs['package-windows'].with['overlap-source-setup'], true);
    assert.deepEqual(jobs['package-windows'].needs, ['scope', 'readiness']);
    const start = earliest(jobs, 'package-windows', durations);
    const received = await waitForArtifact({ repository: 'shruggietech/insonic', run: '42', attempt: '1',
      os: 'windows-2022', token: 'test-only', now: () => start, sleep: async () => assert.fail('ready transfer must not poll'),
      request: async url => url.includes('/jobs?')
        ? { total_count: 2, jobs: ['foundation', 'docs-build'].map(name => ({ name, status: 'completed', conclusion: 'success' })) }
        : { total_count: 1, artifacts: [{ id: 7, name: 'media-source-windows-2022-1', expired: false,
          workflow_run: { id: 42 }, digest: 'sha256:' + 'd'.repeat(64) }] },
    });
    assert.equal(received, 7, 'the actual helper consumes the ready current-run transfer');
  });
}

for (const result of ['failure', 'cancelled']) {
  test(`failed/cancelled source readiness prevents consumers (${result})`, () => {
    const jobs = workflow('ci').jobs;
    for (const source of ['media-windows', 'media-linux']) {
      assert.equal(earliest(jobs, 'package-windows', {}, new Set([source])), Infinity);
      assert.equal(earliest(jobs, 'foundation', {}, new Set([source])), Infinity);
    }
    assert.match(jobs.readiness.if, /always\(\)/);
    assert.match(jobs.readiness.if, /needs\.scope\.result == 'success'/);
  });
}

test('source workflow publishes readiness only after the verified warm or cold producer', () => {
  const media = workflow('media-source');
  assert.deepEqual(deps(media.jobs.ready), ['probe', 'groups', 'formats', 'codec', 'ffmpeg']);
  assert.equal(media.jobs.ready.if, 'always()');
  const step = media.jobs.ready.steps.find(step => step.run === 'node scripts/ci-acceptance.mjs source');
  assert.ok(step, 'terminal readiness must evaluate both producer outcomes');
  assert.equal(media.on.workflow_call.outputs.ready.value, '${{ jobs.ready.outputs.ready }}');
  const probe = media.jobs.probe.steps;
  const verify = probe.findIndex(step => step.id === 'ready');
  const transfer = probe.findIndex(step => step.run?.includes('media-transfer.py pack'));
  const upload = probe.findIndex(step => step.with?.name === 'media-source-${{ inputs.os }}-${{ github.run_attempt }}');
  assert.ok(verify >= 0 && transfer > verify && upload > transfer);
  const completeRestore = probe.find(step => step.name === 'Restore exact complete source cache before scheduling cold stages');
  assert.equal(completeRestore.with['restore-keys'], undefined);
  for (const id of ['probe', 'groups', 'formats', 'codec', 'ffmpeg']) {
    const steps = media.jobs[id].steps;
    assert.equal(steps.find(step => step.with?.path === 'build/automation').with.ref, '${{ job.workflow_sha }}');
    assert.ok(steps.find(step => step.run?.includes('source-toolchain.py')).run.includes("inputs.revision == job.workflow_sha"));
  }
  assert.equal(media.jobs.ready.steps[0].with.ref, '${{ job.workflow_sha }}');
});

test('cold Windows dependency preparation gates qualification and restores a verified same-attempt handoff', () => {
  const jobs = workflow('ci').jobs;
  const durations = { 'dependencies-windows': 400000, 'media-linux': 5000, 'media-windows': 5000 };
  for (const id of ['foundation', 'docs-build', 'package-windows']) {
    assert.ok(earliest(jobs, id, durations) >= 400000);
    assert.equal(earliest(jobs, id, durations, new Set(['dependencies-windows'])), Infinity);
  }
  const steps = jobs['dependencies-windows'].steps;
  assert.equal(steps.find(s => s.uses === './build/actions/setup-go').with.cache, false);
  assert.ok(steps.some(s => s.run === 'python scripts/go-dependency-transfer.py prepare'));
  assert.ok(!steps.some(s => /go test|package-desktop.py all|windows_package_pipeline.py assemble|qualify.py (native|desktop|cli)/.test(s.run ?? '')));
  assert.equal(jobs['package-windows'].with['reuse-ci-go-dependencies'], true);
  for (const [file, id] of [['native-qualification','packages'], ['windows-desktop-qualification','assemble']]) {
    const document = workflow(file);
    assert.equal(document.on.workflow_call.inputs['reuse-ci-go-dependencies'].default, false);
    const consumer = document.jobs[id].steps;
    const download = consumer.findIndex(s => s.with?.name?.startsWith('go-dependencies-windows-'));
    const restore = consumer.findIndex(s => s.run === 'python scripts/ci-package-inputs.py');
    const build = consumer.findIndex(s => /package-desktop.py all|windows_package_pipeline.py assemble/.test(s.run ?? ''));
    const gate = consumer.findIndex(s => s.run === 'node scripts/ci-acceptance.mjs wait-inputs');
    const sourceGate = consumer.findIndex(s => s.run?.includes('wait-ci-artifact.mjs'));
    assert.ok(gate >= 0 && sourceGate > gate && download > sourceGate && restore > download && build > restore);
    assert.match(consumer.find(s => s.uses === './build/actions/setup-go').with.cache, /!.*reuse-ci-go-dependencies/);
    assert.equal(consumer[download].with['digest-mismatch'], 'error');
    assert.ok(consumer[download].with.name.endsWith('-${{ github.run_attempt }}'));
  }
});

test('cold media campaign isolates every compiled cache and keeps the namespace across attempts', () => {
  const ci = workflow('ci'), media = workflow('media-source');
  assert.equal(ci.on.workflow_dispatch.inputs.cold_media_source.default, false);
  assert.equal(media.on.workflow_call.inputs['cold-cache'].default, false);
  for (const os of ['linux', 'windows']) {
    assert.equal(ci.jobs[`media-${os}`].with['cold-cache'], '${{ github.event_name == \'workflow_dispatch\' && inputs.cold_media_source }}');
  }
  const compiled = [], originals = [];
  for (const job of Object.values(media.jobs)) for (const step of job.steps ?? []) {
    if (!step.with?.key?.startsWith('media-')) continue;
    (step.with.key.startsWith('media-original-sources-') ? originals : compiled).push(step);
  }
  assert.equal(compiled.length, 4);
  const ordinaryKeys = [];
  for (const step of compiled) {
    assert.ok(step.with.key.endsWith('${{ inputs.cold-cache && format(\'-campaign-{0}\', github.run_id) || \'\' }}'));
    assert.equal(step.with['restore-keys'], undefined);
    assert.ok(!step.with.key.includes('github.run_attempt'));
    const ordinary = step.with.key.replace('${{ inputs.cold-cache && format(\'-campaign-{0}\', github.run_id) || \'\' }}', '');
    ordinaryKeys.push(ordinary);
  }
  assert.deepEqual(ordinaryKeys.sort(), [
    'media-complete-${{ runner.os }}-${{ runner.arch }}-${{ steps.identity.outputs.key }}',
    'media-complete-${{ runner.os }}-${{ runner.arch }}-${{ steps.identity.outputs.key }}',
    'media-dependency-${{ matrix.group }}-${{ runner.os }}-${{ runner.arch }}-${{ steps.identity.outputs.key }}',
    'media-formats-${{ runner.os }}-${{ runner.arch }}-${{ steps.identity.outputs.key }}',
  ].sort());
  assert.equal(originals.length, 5);
  assert.ok(originals.every(step => !step.with.key.includes('campaign')));
  assert.equal(media.on.workflow_call.outputs.path.value, '${{ jobs.ready.outputs.path }}');
  assert.equal(media.jobs.ready.steps.find(step => step.id === 'ready').env.MEDIA_COLD_CACHE, '${{ inputs.cold-cache }}');
});

test('documentation npm audits keep bounded request and fallback time within input limits', () => {
  const env = workflow('ci').jobs['docs-build'].env;
  assert.equal(env.npm_config_audit, 'true');
  const retries = Number(env.npm_config_fetch_retries), timeout = Number(env.npm_config_fetch_timeout);
  assert.ok(Number.isInteger(retries) && retries >= 0 && retries <= 2);
  assert.ok(timeout > 0 && timeout <= 15000);
  let request = timeout * (retries + 1);
  for (let i = 0; i < retries; i++) request += Math.min(Number(env.npm_config_fetch_retry_mintimeout) * 2 ** i, Number(env.npm_config_fetch_retry_maxtimeout));
  assert.ok(request * 2 < 180000, 'bulk and fallback requests must leave time within the unchanged input operation bound');
});
