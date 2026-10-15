// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile, readdir } from 'node:fs/promises';
import { parse } from 'yaml';

const root = new URL('../', import.meta.url);
test('setup actions acquire verified archives at step time with unchanged inputs and post hooks', async () => {
  const lock = JSON.parse(await readFile(new URL('scripts/bootstrap-actions.lock.json', root), 'utf8'));
  assert.equal(lock['setup-go'].revision, 'b7ad1dad31e06c5925ef5d2fc7ad053ef454303e');
  assert.equal(lock['setup-node'].revision, '949feb2413d6458794dcd2491c4babbbce0c15c1');
  let consumers = 0;
  for (const file of (await readdir(new URL('.github/workflows/', root))).filter(x => x.endsWith('.yml'))) {
    const text = await readFile(new URL(`.github/workflows/${file}`, root), 'utf8');
    assert.doesNotMatch(text, /uses:\s*actions\/setup-(?:go|node)@/);
    const workflow = parse(text);
    for (const job of Object.values(workflow.jobs)) {
      for (const [i, step] of (job.steps ?? []).entries()) {
        if (!/^\.\/build\/actions\/setup-(go|node)$/.test(step.uses ?? '')) continue;
        consumers += 1;
        const name = step.uses.split('/').at(-1);
        assert.match(job.steps[i - 1].run, new RegExp(`bootstrap-actions.py"? ${name} --output build/actions/${name}$`));
        assert.equal(job.steps[i - 1].if, step.if);
        assert.ok(step.with[name === 'setup-go' ? 'go-version-file' : 'node-version']);
        if (file === 'release.yml' || file === 'native-qualification.yml') {
          const tools = job.steps.find(s => s.with?.path === 'build/automation');
          assert.equal(tools.with.ref, '${{ job.workflow_sha }}');
          assert.equal(tools.with.repository, '${{ job.workflow_repository }}');
          assert.equal(tools.with['persist-credentials'], false);
          assert.equal(tools.if, 'inputs.revision != job.workflow_sha');
          assert.ok(job.steps.indexOf(tools) < i - 1);
          assert.match(tools.with['sparse-checkout'], /scripts\/http_read.py/);
          assert.match(job.steps[i - 1].run, /inputs.revision == job.workflow_sha/);
          assert.match(job.steps[i - 1].run, /build\/automation/);
        }
      }
    }
  }
  assert.ok(consumers >= 14);
});
