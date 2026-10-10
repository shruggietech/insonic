// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import { documentationTarget, publicationManifest, publicationDocument, releaseDownloads } from '../site/lib/release-status.mjs';

test('release rendering is elected and leaves source/baseline publication state intact', () => {
  const source = { latest: '1.0.0', versions: [{ version: '0.0.0', status: 'specification' }, { version: '1.0.0', status: 'candidate' }] };
  assert.equal(documentationTarget({}), 'snapshot');
  assert.throws(() => documentationTarget({ INSONIC_DOCUMENTATION_TARGET: 'published-maybe' }));
  const rendered = publicationManifest(source, 'release');
  assert.equal(rendered.versions[1].status, 'released');
  assert.equal(rendered.versions[0].status, 'specification');
  assert.equal(source.versions[1].status, 'candidate');
  assert.throws(() => publicationManifest({ latest: '0.0.0', versions: [source.versions[0]] }, 'release'));
});

test('release artifact index removes the candidate assertion while ordinary snapshots preserve it', () => {
  const source = '# insonic v1.0.0\n\nv1.0.0 documents the prepared release candidate. Official product downloads have not been published.\n';
  const entry = { version: '1.0.0', status: 'released' };
  assert.equal(publicationDocument(source, entry, 'index', 'snapshot'), source);
  assert.equal(publicationDocument(source, entry, 'technology', 'release'), source);
  const output = publicationDocument(source, entry, 'index', 'release');
  assert.match(output, /documents this release/);
  assert.doesNotMatch(output, /candidate|unpublished|not been published/);
});

test('released downloads identify all six exact platform/variant assets', () => {
  const assets = releaseDownloads('1.0.0');
  assert.equal(assets.length, 3);
  for (const item of assets) for (const variant of ['desktop', 'cli'])
    assert.equal(item[variant], `https://github.com/shruggietech/insonic/releases/download/v1.0.0/insonic_1.0.0_${item.platform}_${variant}${item.extension}`);
});
