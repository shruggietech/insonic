// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict';
import test from 'node:test';
import { readFileSync } from 'node:fs';
const currentVersion = readFileSync(new URL('../../VERSION', import.meta.url), 'utf8').trim();
import { landingRenderer, replaceLanding } from '../scripts/landing-fixture.mjs';
import { releaseDownloads } from '../lib/release-status.mjs';

const render = await landingRenderer();
const anchors = html => [...html.matchAll(/<a\b([^>]*)href="([^"]+)"([^>]*)>([\s\S]*?)<\/a>/g)].map(([, before, href, after, body]) => ({ href, label: body.replace(/<span[^>]*aria-hidden="true"[^>]*>[\s\S]*?<\/span>/g, '').replace(/<[^>]+>/g, '').trim(), attributes: before + after }));
const released = render({ version: currentVersion, status: 'released' });

test('released landing preserves exactly six versioned OS/client/format choices', () => {
  const links = anchors(released);
  const downloads = links.filter(link => link.href.includes('/releases/download/'));
  assert.equal(downloads.length, 6);
  const expected = releaseDownloads(currentVersion).flatMap(item => [item.setup, item.installer, item.desktop, item.cli].filter(Boolean));
  assert.deepEqual(downloads.map(link => link.href).sort(), expected.sort());
  assert.deepEqual(downloads.map(link => link.label), ['Download Windows setup EXE', 'Download raw MSI', 'Download Windows Desktop + CLI ZIP', 'Download Windows CLI-only ZIP', 'Download Linux Desktop + CLI tar.gz', 'Download Linux CLI-only tar.gz']);
  assert.match(released, /Desktop, CLI or both/);
  assert.match(released, /Both are selected by default/);
  assert.equal((released.match(/class="download-group"/g) || []).length, 3);
  assert.match(released, /Portable ZIP packages/);
  assert.match(released, /Portable tar.gz packages/);
  assert.doesNotMatch(released, /macOS|darwin|desktop-only/);
});

test('unpublished landing never emits premature package destinations', () => {
  for (const status of ['candidate', 'specification']) {
    const html = render({ version: currentVersion, status });
    assert.doesNotMatch(html, /releases\/download\//);
    assert.match(html, /official product downloads have not been published/i);
    assert.match(html, /Windows download pending/);
    assert.match(html, /Linux download pending/);
  }
});

test('rendered landing gives one primary hero jump and truthful distinct guidance', () => {
  const links = anchors(released);
  assert.equal(links.filter(link => /class="button"/.test(link.attributes)).length, 1);
  assert.equal(links.find(link => link.href === '#downloads')?.label, 'Choose a download');
  const guidance = links.filter(link => link.href.includes('desktop/') || link.href.includes('windows-installation/'));
  assert.deepEqual(guidance.map(({label,href}) => [label,href]), [
    ['Windows installation, updates and removal', '/docs/latest/windows-installation/'],
    ['Windows ZIP installation and maintenance', '/docs/latest/desktop/#install-a-windows-zip'],
    ['Linux prerequisites and installation', '/docs/latest/desktop/#linux-prerequisites-and-installation'],
  ]);
  assert.doesNotMatch(released, /Installation and packaging requirements/);
  assert.equal(links.length, 18);
  for (const link of links) {
    assert.ok(link.label);
    if (link.href.startsWith('/docs/') && link.href !== '/docs/versions/') assert.ok(link.href.startsWith('/docs/latest/'));
    assert.doesNotMatch(link.attributes, /role="button"|onclick=/);
  }
});

test('same-source fixture replaces only the real landing frame and rejects ambiguous input', () => {
  const frame = '<html><head><link href="style.css"/></head><body><header>chrome</header>' + render({version:currentVersion,status:'candidate'}) + '<footer>company</footer></body></html>';
  const result = replaceLanding(frame, released);
  assert.ok(result.startsWith('<html><head>'));
  assert.match(result, /<header>chrome<\/header>/);
  assert.match(result, /<footer>company<\/footer>/);
  assert.match(result, /Download Windows setup EXE/);
  assert.match(result, /href="docs\/latest\/windows-installation\/index.html"/);
  assert.doesNotMatch(result, /href="\/docs\//);
  assert.ok(replaceLanding(frame, released.replace('Download Windows setup EXE', () => 'Literal $& package')).includes('Literal $& package'));
  assert.throws(() => replaceLanding('<main>other</main>', released), /one real public landing/);
  assert.throws(() => replaceLanding(frame.replace('</body>',released+'</body>'), released), /one real public landing/);
});
