// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { parse } from 'yaml';

test('website publishing qualifies source with trusted main before executing selected build code', () => {
  const workflow = parse(readFileSync(new URL('../.github/workflows/website.yml', import.meta.url), 'utf8'));
  const build = workflow.jobs.build;
  assert.equal(build.permissions.contents, 'read');
  assert.equal(build.steps[0].with.ref, 'main');
  const gate = build.steps.findIndex(s => s.run?.includes('website.py source'));
  const selected = build.steps.findIndex(s => s.with?.ref === '${{ inputs.revision }}');
  assert.ok(gate > 0 && selected > gate);
  const publish = workflow.jobs.publish;
  assert.equal(publish.steps[0].with.ref, 'main');
  assert.equal(publish.steps[0].with['persist-credentials'], false);
  assert.equal(publish.permissions.contents, 'write');
  assert.equal(publish.permissions.pages, 'write');
  assert.equal(publish.concurrency.group, 'pages-publication');
  assert.equal(publish.concurrency['cancel-in-progress'], false);
  const original = parse(readFileSync(new URL('../.github/workflows/release.yml', import.meta.url), 'utf8'));
  assert.deepEqual(original.jobs['promote-pages'].concurrency, publish.concurrency);
  const serialized = JSON.stringify(workflow);
  for (const forbidden of ['release.py publish', 'tag --force', 'git push --force', 'secrets.']) assert.ok(!serialized.includes(forbidden));
  for (const job of Object.values(workflow.jobs)) assert.equal(job['timeout-minutes'],11);
  assert.equal(workflow.jobs.verify.permissions.contents, 'read');
  assert.ok(workflow.jobs.verify.steps.some(s => s.run?.includes('check-live-site-ui.mjs')));
  assert.ok(workflow.jobs.verify.steps.some(s => s.run?.includes('website.py deployment')));
});
