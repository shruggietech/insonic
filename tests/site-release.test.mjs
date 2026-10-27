// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import { documentationTarget, publicationManifest, publicationDocument, releaseDownloads } from '../site/lib/release-status.mjs';

test('release rendering is elected and leaves source/baseline publication state intact', () => {
  const source = { latest: '1.0.2', versions: [{ version: '0.0.0', status: 'specification' }, { version: '1.0.2', status: 'candidate' }] };
  assert.equal(documentationTarget({}), 'snapshot');
  assert.throws(() => documentationTarget({ INSONIC_DOCUMENTATION_TARGET: 'published-maybe' }));
  const rendered = publicationManifest(source, 'release');
  assert.equal(rendered.versions[1].status, 'released');
  assert.equal(rendered.versions[0].status, 'specification');
  assert.equal(source.versions[1].status, 'candidate');
  assert.throws(() => publicationManifest({ latest: '0.0.0', versions: [source.versions[0]] }, 'release'));
});

test('release artifact index removes the candidate assertion while ordinary snapshots preserve it', () => {
  const entry = { version: '1.0.2', status: 'candidate' };
  const source = `# insonic v${entry.version}\n\nv${entry.version} documents the prepared release candidate. Official product downloads have not been published.\n`;
  const manifest = { latest: entry.version, versions: [entry] };
  assert.equal(publicationDocument(source, publicationManifest(manifest, 'snapshot').versions[0], 'index'), source);
  const released = publicationManifest(manifest, 'release').versions[0];
  assert.equal(publicationDocument(source, released, 'technology'), source);
  const output = publicationDocument(source, released, 'index');
  assert.match(output, /documents this release/);
  assert.doesNotMatch(output, /candidate|unpublished|not been published/);
});

test('current release downloads identify all six exact-tag product assets', () => {
  const assets = releaseDownloads('1.0.2');
  assert.deepEqual(assets.map(item => item.platform), ['windows_amd64', 'linux_amd64']);
  const urls = assets.flatMap(item => ['desktop', 'cli', 'setup', 'installer'].filter(key => item[key]).map(key => item[key]));
  assert.equal(new Set(urls).size, 6);
  for (const item of assets) for (const variant of ['desktop', 'cli'])
    assert.equal(item[variant], `https://github.com/shruggietech/insonic/releases/download/v1.0.2/insonic_1.0.2_${item.platform}_${variant}${item.extension}`);
  assert.equal(assets[0].setup, 'https://github.com/shruggietech/insonic/releases/download/v1.0.2/insonic_1.0.2_windows_amd64_setup.exe');
  assert.equal(assets[0].installer, 'https://github.com/shruggietech/insonic/releases/download/v1.0.2/insonic_1.0.2_windows_amd64_installer.msi');
  assert.equal(assets[1].setup, undefined);
  assert.equal(assets[1].installer, undefined);
});

test('immutable historical releases retain their original platform and installer choices', () => {
  const initial = releaseDownloads('1.0.0', { historical: true });
  assert.equal(initial.length, 3);
  assert.equal(initial[0].setup, undefined);
  assert.equal(initial[0].installer, undefined);
  const previous = releaseDownloads('1.0.1', { historical: true });
  assert.equal(previous.length, 2);
  assert.equal(previous[0].setup, undefined);
  assert.equal(previous[0].installer, 'https://github.com/shruggietech/insonic/releases/download/v1.0.1/insonic_1.0.1_windows_amd64_installer.msi');
});
