// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { parse } from 'yaml';
import { classifyNewIssue } from '../scripts/classify-issue.mjs';
const manifest = JSON.parse(readFileSync(new URL('../.github/bootstrap.json', import.meta.url), 'utf8'));
const opened = () => ({ action: 'opened', repository: { full_name: manifest.repository }, issue: { number: 41, body: '### Desired outcome\n\nImport audio.\n\n### Affected areas\n\naudio, ingestion\n', labels: ['enhancement', 'type: feature', 'priority: medium', 'effort: m'] } });

test('forms declare one canonical type, priority and effort and known selectable areas', () => {
  for (const name of ['bug', 'feature']) {
    const form = parse(readFileSync(new URL(`../.github/ISSUE_TEMPLATE/${name}.yml`, import.meta.url), 'utf8'));
    for (const group of ['type', 'priority', 'effort']) assert.equal(form.labels.filter(label => label.startsWith(`${group}: `)).length, 1);
    assert.deepEqual(form.body.find(field => field.id === 'area').attributes.options, manifest.taxonomy.area);
  }
});

test('new form areas add canonical labels while preserving compatibility and other edits', () => {
  const event = opened(); const previous = [...event.issue.labels]; const calls = [];
  const result = classifyNewIssue(event, manifest, (args, input) => {
    calls.push(args); event.issue.labels.push(...JSON.parse(input).labels);
  });
  assert.deepEqual(result.added, ['area: audio', 'area: ingestion']);
  assert.deepEqual(event.issue.labels, [...previous, 'area: audio', 'area: ingestion']);
  assert.equal(calls[0][calls[0].indexOf('--method') + 1], 'POST');
  assert.deepEqual(classifyNewIssue(event, manifest, () => assert.fail('Repeated mutation')).added, []);
});

test('historical edited or closed issues, malformed forms and unknown values do not mutate', () => {
  for (const modify of [e => e.action = 'edited', e => e.action = 'closed', e => e.issue.pull_request = {}, e => e.issue.body = '### Affected areas\n\naudio, injected label', e => e.issue.body = 'unstructured text', e => e.issue.labels.push('priority: critical'), e => e.issue.labels.push('priority: unknown'), e => e.repository.full_name = 'other/repository']) {
    const event = opened(); modify(event);
    assert.equal(classifyNewIssue(event, manifest, () => assert.fail('Unexpected label mutation')).applied, false);
  }
});
