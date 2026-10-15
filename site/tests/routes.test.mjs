// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict';
import test from 'node:test';
import { getLatestEntry, latestPath, latestReferencePath, latestTopicSlugs, latestReferenceSlugs } from '../lib/docs-routes.mjs';
import { renderDocument } from '../lib/docs.mjs';
import { relativeTarget } from '../scripts/postbuild.mjs';

const manifest = {
  latest: '2.0.0',
  versions: [
    { version: '1.0.0', status: 'released', pages: [{ slug: 'index', title: 'Old start' }, { slug: 'retired', title: 'Retired topic' }] },
    { version: '2.0.0', status: 'candidate', pages: [{ slug: 'index', title: 'Start' }, { slug: 'setup', title: 'Setup' }] },
  ],
};

test('current routing follows manifest election without changing archived identity or status', () => {
  const before = structuredClone(manifest);
  assert.equal(getLatestEntry(manifest).version, '2.0.0');
  assert.equal(getLatestEntry(manifest).status, 'candidate');
  const changed = { ...manifest, latest: '1.0.0' };
  assert.equal(getLatestEntry(changed).version, '1.0.0');
  assert.equal(getLatestEntry(changed).status, 'released');
  assert.deepEqual(manifest, before);
  assert.throws(() => getLatestEntry({ ...manifest, latest: '9.9.9' }), /latest/);
});

test('stable aliases contain current and historical-only topics for truthful missing-topic fallbacks', () => {
  assert.equal(latestPath(), '/docs/latest/');
  assert.equal(latestPath('index'), '/docs/latest/');
  assert.equal(latestPath('setup'), '/docs/latest/setup/');
  assert.equal(latestReferencePath('changelog-md'), '/docs/latest/references/changelog-md/');
  assert.deepEqual(latestTopicSlugs(manifest), ['index', 'retired', 'setup']);
  assert.deepEqual(latestReferenceSlugs([{ slug: 'contributing-md' }, { slug: 'changelog-md' }, { slug: 'contributing-md' }]), ['changelog-md', 'contributing-md']);
  for (const slug of ['../old', 'setup#bad', 'setup?bad', '', 'https://example.org/']) {
    assert.throws(() => latestPath(slug), /topic/);
    assert.throws(() => latestReferencePath(slug), /reference/);
  }
});

test('portable links preserve query and fragment suffixes without treating them as filesystem paths', () => {
  assert.equal(relativeTarget('/export/docs/latest/setup/index.html', '/docs/v2.0.0/setup/?mode=compact#location', '/export'), '../../v2.0.0/setup/index.html?mode=compact#location');
  assert.equal(relativeTarget('/export/docs/index.html', '/docs/latest/#start', '/export'), 'latest/index.html#start');
  assert.equal(relativeTarget('/export/docs/index.html', 'https://example.org/docs/#start', '/export'), 'https://example.org/docs/#start');
});

test('complex content retains semantic tables and keyboard-scrollable code/diagram containers', () => {
  const entry = manifest.versions[1];
  const result = renderDocument('# Example\n\n| Key | Value |\n| --- | --- |\n| one | two |\n\n```json\n{"wide":"value"}\n```\n\n```mermaid\nflowchart TB\n  A --> B\n```\n', entry);
  assert.match(result.html, /<div class="table-scroll" role="region" aria-label="Scrollable table" tabindex="0"><table>/);
  assert.match(result.html, /<th scope="col">Key<\/th>/);
  assert.match(result.html, /<pre tabindex="0" aria-label="Code example">/);
  assert.match(result.html, /<figure class="diagram" role="group" tabindex="0" aria-label="System diagram and source">/);
  assert.match(result.html, /<details><summary>Diagram source<\/summary><pre tabindex="0" aria-label="Diagram source">/);
});
