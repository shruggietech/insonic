// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import { documentationTarget, publicationManifest, publicationDocument, releaseDownloads } from '../site/lib/release-status.mjs';

test('release rendering is elected and leaves source/baseline publication state intact', () => {
  const source = { latest: '1.0.1', versions: [{ version: '0.0.0', status: 'specification' }, { version: '1.0.1', status: 'candidate' }] };
  assert.equal(documentationTarget({}), 'snapshot');
  assert.throws(() => documentationTarget({ INSONIC_DOCUMENTATION_TARGET: 'published-maybe' }));
  const rendered = publicationManifest(source, 'release');
  assert.equal(rendered.versions[1].status, 'released');
  assert.equal(rendered.versions[0].status, 'specification');
  assert.equal(source.versions[1].status, 'candidate');
  assert.throws(() => publicationManifest({ latest: '0.0.0', versions: [source.versions[0]] }, 'release'));
});

test('release artifact index removes the candidate assertion while ordinary snapshots preserve it', () => {
  const source = '# insonic v1.0.1\n\nv1.0.1 documents the prepared release candidate. Official product downloads have not been published.\n';
  const entry = { version: '1.0.1', status: 'candidate' };
  const manifest = { latest: entry.version, versions: [entry] };
  assert.equal(publicationDocument(source, publicationManifest(manifest, 'snapshot').versions[0], 'index'), source);
  const released = publicationManifest(manifest, 'release').versions[0];
  assert.equal(publicationDocument(source, released, 'technology'), source);
  const output = publicationDocument(source, released, 'index');
  assert.match(output, /documents this release/);
  assert.doesNotMatch(output, /candidate|unpublished|not been published/);
});

test('released downloads identify all four current platform/variant assets', () => {
  const assets = releaseDownloads('1.0.1');
  assert.equal(assets.length, 2);
  assert.deepEqual(assets.map(item => item.platform), ["windows_amd64", "linux_amd64"]);
  assert.equal(releaseDownloads("1.0.0", { historical: true }).length, 3);
  assert.equal(releaseDownloads("1.0.1", { historical: true }).length, 2);
  for (const item of assets) for (const variant of ['desktop', 'cli'])
    assert.equal(item[variant], `https://github.com/shruggietech/insonic/releases/download/v1.0.1/insonic_1.0.1_${item.platform}_${variant}${item.extension}`);
});

test('MSI is an additional current Windows choice while immutable 1.0.0 has no installer link', () => {
  const current = releaseDownloads('1.0.1');
  assert.equal(current[0].installer, 'https://github.com/shruggietech/insonic/releases/download/v1.0.1/insonic_1.0.1_windows_amd64_installer.msi');
  assert.equal(current[1].installer, undefined);
  assert.equal(releaseDownloads('1.0.0')[0].installer, undefined);
  assert.equal(releaseDownloads('1.0.0', { historical: true })[0].installer, undefined);
});
