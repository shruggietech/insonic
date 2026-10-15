import test from 'node:test';
import assert from 'node:assert/strict';
import { waitForArtifact } from '../scripts/wait-ci-artifact.mjs';

const options = { repository: 'shruggietech/insonic', run: '123', attempt: '2', os: 'windows-2022', token: 'fixture' };
const artifact = { id: 42, name: 'media-source-windows-2022-2', expired: false, digest: 'sha256:' + 'a'.repeat(64), workflow_run: { id: 123 } };
const response = artifacts => ({ total_count: artifacts.length, artifacts });

test('source handoff waits for this exact run and attempt without downloading bytes', async () => {
  let calls = 0, elapsed = 0;
  const value = await waitForArtifact({ ...options, now: () => elapsed, sleep: async ms => { elapsed += ms; },
    request: async url => { assert.match(url, /^https:\/\/api.github.com\/repos\/shruggietech\/insonic\/actions\/runs\/123\/artifacts\?/); return response(++calls === 1 ? [{ ...artifact, name: 'media-source-windows-2022-1' }] : [artifact]); } });
  assert.equal(value, 42); assert.equal(calls, 2); assert.equal(elapsed, 5000);
});

test('ambiguous, expired, foreign and unbounded handoffs fail closed', async () => {
  for (const data of [response([artifact, artifact]), response([{ ...artifact, expired: true }]), response([{ ...artifact, workflow_run: { id: 999 } }]), { ...response([]), total_count: 101 }]) {
    await assert.rejects(waitForArtifact({ ...options, request: async () => data }), /artifact/);
  }
});

test('missing producer has one aggregate deadline, no stale fallback', async () => {
  let elapsed = 0;
  await assert.rejects(waitForArtifact({ ...options, deadlineMs: 10000, now: () => elapsed, sleep: async ms => { elapsed += ms; }, request: async () => response([]) }), /deadline/);
  assert.equal(elapsed, 10000);
  await assert.rejects(waitForArtifact({ ...options, run: '../other', request: async () => { throw new Error('must not request'); } }), /identity/);
});
