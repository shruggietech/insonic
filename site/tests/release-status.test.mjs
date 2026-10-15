// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict';
import test from 'node:test';
import { readFileSync } from 'node:fs';
import { publicationManifest, publicationDocument } from '../lib/release-status.mjs';
import { getManifest, readDocument } from '../lib/docs.mjs';

const source = version => `# insonic v${version}\n\nv${version} documents the prepared release candidate. Official product downloads have not been published; package availability and executed qualification are recorded in its candidate manifest.\n`;
const manifest = { latest: '1.0.1', versions: [
  { version: '0.0.0', status: 'specification' },
  { version: '1.0.0', status: 'released' },
  { version: '1.0.1', status: 'candidate' },
] };

test('publication election preserves historical states and leaves source data immutable', () => {
  const before = structuredClone(manifest);
  for (const target of ['snapshot', 'release']) {
    const rendered = publicationManifest(manifest, target);
    assert.equal(rendered.versions[0].status, 'specification');
    assert.equal(rendered.versions[1].status, 'released');
    assert.equal(rendered.versions[2].status, target === 'release' ? 'released' : 'candidate');
    assert.deepEqual(manifest, before);
  }
});

test('released historical index is truthful in snapshot and publication targets without changing frozen source', () => {
  const original = source('1.0.0');
  for (const target of ['snapshot', 'release']) {
    const entry = publicationManifest(manifest, target).versions[1];
    const rendered = publicationDocument(original, entry, 'index');
    assert.match(rendered, /v1\.0\.0 documents this release\./);
    assert.doesNotMatch(rendered, /prepared release candidate|have not been published/);
    assert.equal(original, source('1.0.0'));
    assert.equal(publicationDocument(original, entry, 'architecture'), original);
  }
});

test('latest candidate claims change only after explicit publication election', () => {
  const original = source('1.0.1');
  assert.equal(publicationDocument(original, manifest.versions[2], 'index'), original);
  const released = publicationManifest(manifest, 'release').versions[2];
  assert.match(publicationDocument(original, released, 'index'), /documents this release\./);
  assert.equal(manifest.versions[2].status, 'candidate');
});

test('repository historical status reaches actual document rendering while frozen Markdown remains unchanged', () => {
  const original = readFileSync(new URL('../../docs/v1.0.0/index.md', import.meta.url), 'utf8');
  assert.match(original, /prepared release candidate/);
  const entry = getManifest().versions.find(item => item.version === '1.0.0');
  assert.equal(entry.status, 'released');
  const archived = readDocument('v1.0.0', 'index');
  assert.match(archived.source, /v1\.0\.0 documents this release\./);
  assert.doesNotMatch(archived.source, /have not been published/);
  assert.equal(readFileSync(new URL('../../docs/v1.0.0/index.md', import.meta.url), 'utf8'), original);
});
