// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict';
import { mkdtemp, mkdir, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';
import { checkRepository, markdownAnchors, markdownTargets, validateLinks, validateMarkdown, validateText, validateVersions, validateYAML } from '../scripts/check.mjs';

test('text integrity detects damaged encoding, BOM, CRLF, and conflict markers', () => {
  assert.deepEqual(validateText('clean.md', Buffer.from('# A title\n\nA normal paragraph.\n')).problems, []);
  assert.match(validateText('bad.md', Buffer.from([0xc3, 0x28])).problems.join('\n'), /Invalid UTF-8/);
  assert.match(validateText('bom.md', Buffer.from('\uFEFF# Title\n')).problems.join('\n'), /BOM/);
  assert.match(validateText('crlf.md', Buffer.from('# Title\r\n')).problems.join('\n'), /LF/);
  assert.equal(validateText('crlf.md', Buffer.from('# Title\r\n\r\nText\r\n')).problems.filter(problem => problem.includes('Use LF')).length, 1);
  assert.match(validateText('damaged.md', Buffer.from('# Title\n\u00C3\u00A9\n')).problems.join('\n'), /mojibake/);
  assert.match(validateText('conflict.md', Buffer.from('# Title\n<<<<<<< HEAD\n')).problems.join('\n'), /merge conflict/);
});

test('Markdown validation understands fenced examples and top-down Mermaid graphs', () => {
  const valid = '# Title\n\n```markdown\n# An example heading\n```\n\n```mermaid\nflowchart TB\n  A --> B\n```\n';
  assert.deepEqual(validateMarkdown('good.md', valid), []);
  assert.match(validateMarkdown('bad.md', '# One\n# Two\n').join('\n'), /exactly one H1/);
  assert.match(validateMarkdown('bad.md', '# One\n```\nexample\n```\n').join('\n'), /needs a language/);
  assert.match(validateMarkdown('bad.md', '# One\n```mermaid\nflowchart LR\nA-->B\n```\n').join('\n'), /TB or TD/);
  assert.match(validateMarkdown('bad.md', '# One\n```text\nunclosed\n').join('\n'), /Unclosed/);
});

test('anchors include repeated headings and explicit HTML identifiers', () => {
  const anchors = markdownAnchors('# Title\n## API, models & keys\n## API, models & keys\n<a id="custom"></a>\n```markdown\n## Hidden\n```\n');
  assert.ok(anchors.has('api-models--keys'));
  assert.ok(anchors.has('api-models--keys-1'));
  assert.ok(anchors.has('custom'));
  assert.ok(!anchors.has('hidden'));
});

test('link extraction ignores code examples and supports references, assets, and parentheses', () => {
  const targets = markdownTargets('# Title\n[Target](guide(1).md#hello)\n[Reference]: <other.md>\n<picture><img src="logo.svg"></picture>\n`[code](missing.md)`\n```markdown\n[example](missing.md)\n```\n').map(({ target }) => target);
  assert.deepEqual(targets, ['guide(1).md#hello', 'other.md', 'logo.svg']);
});

test('relative links and anchors resolve from root, site, and versioned documents', async () => {
  const root = await mkdtemp(join(tmpdir(), 'insonic-links-test-'));
  try {
    await mkdir(join(root, 'site')); await mkdir(join(root, 'docs', 'v0.0.0'), { recursive: true });
    await writeFile(join(root, 'README.md'), '# Project\n## Overview\n');
    await writeFile(join(root, 'docs', 'v0.0.0', 'index.md'), '# Start\n## Install\n');
    assert.deepEqual(await validateLinks(root, 'site/README.md', '# Site\n[Source](../README.md#overview)\n[Guide](../docs/v0.0.0/index.md#install)\n'), []);
    assert.deepEqual(await validateLinks(root, 'docs/v0.0.0/index.md', '# Start\n[Project](../../README.md#overview)\n'), []);
    assert.deepEqual(await validateLinks(root, 'docs/v0.0.0/references/CONTRIBUTING.md', '# Contributing\n[Guide](docs/v0.0.0/index.md#install)\n'), []);
    assert.match((await validateLinks(root, 'docs/v0.0.0/references/CONTRIBUTING.md', '# Contributing\n[Guide](docs/v0.0.0/index.md#absent)\n')).join('\n'), /Missing Markdown anchor/);
    assert.match((await validateLinks(root, 'README.md', '# Project\n[Missing](missing.md)\n[Missing anchor](README.md#absent)\n')).join('\n'), /Missing relative link target/);
    assert.match((await validateLinks(root, 'README.md', '# Project\n[Missing anchor](README.md#absent)\n')).join('\n'), /Missing Markdown anchor/);
    assert.match((await validateLinks(root, 'README.md', '# Project\n[Outside](../outside.md)\n')).join('\n'), /leaves the repository/);
  } finally { await rm(root, { recursive: true, force: true }); }
});

test('documentation navigation detects version drift and absent pages', async () => {
  const root = await mkdtemp(join(tmpdir(), 'insonic-navigation-test-'));
  try {
    await mkdir(join(root, 'docs', 'v0.0.0'), { recursive: true });
    await writeFile(join(root, 'VERSION'), '0.0.0\n');
    await writeFile(join(root, 'docs', 'v0.0.0', 'index.md'), '# Start\n');
    const navigation = { latest: '0.0.0', versions: [{ version: '0.0.0', pages: [{ slug: 'index', title: 'Start' }] }] };
    await writeFile(join(root, 'docs', 'versions.json'), JSON.stringify(navigation));
    assert.deepEqual(await validateVersions(root), []);
    navigation.latest = '0.0.1'; navigation.versions[0].pages.push({ slug: 'missing', title: 'Missing' });
    await writeFile(join(root, 'docs', 'versions.json'), JSON.stringify(navigation));
    const problems = (await validateVersions(root)).join('\n');
    assert.match(problems, /does not match VERSION/); assert.match(problems, /Navigation page is missing/);
  } finally { await rm(root, { recursive: true, force: true }); }
});

test('project scan checks real JSON and constitution while exempting vendor examples and outputs', async () => {
  const root = await mkdtemp(join(tmpdir(), 'insonic-scan-test-'));
  try {
    for (const directory of ['.local', 'build']) {
      await mkdir(join(root, directory), { recursive: true });
      await writeFile(join(root, directory, 'generated.js'), Buffer.from([0xff]));
    }
    for (const directory of ['docs/v0.0.0', '.agents/skills/example', '.specify/templates', '.specify/memory', 'node_modules/example', 'site/out', 'site/.brandbuilder/runtime', 'site/public/brand', 'site/public/mermaid', 'site/public/source', 'site/public/project']) await mkdir(join(root, directory), { recursive: true });
    await writeFile(join(root, 'VERSION'), '0.0.0\n');
    await writeFile(join(root, 'docs', 'v0.0.0', 'index.md'), '# Start\n');
    await writeFile(join(root, 'docs', 'versions.json'), JSON.stringify({ latest: '0.0.0', versions: [{ version: '0.0.0', pages: [{ slug: 'index', title: 'Start' }] }] }));
    await writeFile(join(root, '.agents', 'skills', 'example', 'SKILL.md'), 'A vendor template\n```\nplaceholder\n```\n');
    await writeFile(join(root, '.specify', 'templates', 'template.md'), 'A vendor template\n[placeholder](missing.md)\n');
    await writeFile(join(root, '.specify', 'memory', 'constitution.md'), '# Constitution\r\n');
    for (const file of ['actual.json', 'node_modules/example/invalid.json', 'site/out/invalid.json', 'site/.brandbuilder/runtime/invalid.json', 'site/public/brand/invalid.json', 'site/public/mermaid/invalid.json', 'site/public/source/invalid.json', 'site/public/project/invalid.json', '.preview-check.mjs']) await writeFile(join(root, file), '{invalid');
    const { problems } = await checkRepository(root, { verifyBrand: false });
    assert.equal(problems.length, 2, problems.join('\n'));
    assert.ok(problems.some((problem) => problem.startsWith('actual.json: Invalid JSON')));
    assert.ok(problems.some((problem) => problem.startsWith('.specify/memory/constitution.md:1: Use LF')));
  } finally { await rm(root, { recursive: true, force: true }); }
});


test('YAML validation catches workflow colon scalars and duplicate keys before publication', () => {
  const command='python -m pip install --only-binary=:all: --target build/native/python zstandard==0.25.0';
  assert.match(validateYAML('ci.yml', 'steps:\n  - run: '+command+'\n').join('\n'), /Invalid YAML/);
  assert.deepEqual(validateYAML('ci.yml', 'steps:\n  - run: |\n      '+command+'\n'), []);
  assert.match(validateYAML('ci.yml', 'name: first\nname: second\n').join('\n'), /Invalid YAML/);
  assert.deepEqual(validateYAML('ci.yml', 'on: [push, pull_request]\njobs:\n  check:\n    if: needs.foundation.outputs.native == \'true\'\n'), []);
});

test('artifact workflows require immutable pins and explicit archive, digest and attempt boundaries', () => {
  const pin = 'a'.repeat(40);
  const attempt = '${{ github.run_attempt }}';
  const upload = { uses: `actions/upload-artifact@${pin}`, with: { archive: true, name: `source-${attempt}`, path: 'build/source' } };
  const download = { uses: `actions/download-artifact@${pin}`, with: { 'digest-mismatch': 'error', 'skip-decompress': false, pattern: `source-*-${attempt}`, path: 'build/download' } };
  const validate = steps => validateYAML('.github/workflows/fixture.yml', JSON.stringify({ jobs: { fixture: { steps } } }));
  assert.deepEqual(validate([upload, download]), []);
  const mixedUpload = { ...upload, uses: upload.uses.replace('actions/upload-artifact', 'Actions/UPLOAD-ARTIFACT') };
  const mixedDownload = { ...download, uses: download.uses.replace('actions/download-artifact', 'ACTIONS/Download-Artifact') };
  assert.deepEqual(validate([mixedUpload, mixedDownload]), []);
  for (const action of [upload, download, mixedUpload, mixedDownload]) {
    assert.match(validate([{ ...action, uses: action.uses.replace(pin, 'v8') }]).join('\n'), /immutable/);
    const missingAttempt = structuredClone(action);
    for (const key of ['name', 'pattern']) if (key in missingAttempt.with) missingAttempt.with[key] = 'source';
    assert.match(validate([missingAttempt]).join('\n'), /run_attempt/);
  }
  assert.match(validate([{ ...mixedUpload, with: { ...upload.with, archive: false } }]).join('\n'), /archive/);
  assert.match(validate([{ ...mixedDownload, with: { ...download.with, 'digest-mismatch': 'warn' } }]).join('\n'), /digest-mismatch/);
  assert.match(validate([{ ...mixedDownload, with: { ...download.with, 'skip-decompress': true } }]).join('\n'), /skip-decompress/);
  for (const value of [false, undefined, '${{ inputs.archive }}']) {
    const step = structuredClone(upload);
    step.with.archive = value;
    assert.match(validate([step]).join('\n'), /archive/);
  }
  for (const value of ['warn', undefined]) {
    const step = structuredClone(download);
    step.with['digest-mismatch'] = value;
    assert.match(validate([step]).join('\n'), /digest-mismatch/);
  }
  for (const value of [true, undefined]) {
    const step = structuredClone(download);
    step.with['skip-decompress'] = value;
    assert.match(validate([step]).join('\n'), /skip-decompress/);
  }
  assert.deepEqual(validate([{ ...upload, with: { ...upload.with, archive: 'true' } }, { ...download, with: { ...download.with, 'skip-decompress': 'false' } }]), []);
  assert.deepEqual(validateYAML('configuration.yml', JSON.stringify({ jobs: { fixture: { steps: [{ uses: 'actions/upload-artifact@v8' }] } } })), []);
});


test('only the manual native diagnostic may replay a statically pinned incident artifact',()=>{
 const download={uses:'actions/download-artifact@9000827ccba6bdab643e8b6fd33ac0654aef8333',if:"matrix.kind == 'desktop-startup'",with:{'digest-mismatch':'error','skip-decompress':false,'run-id':39097965060,name:'windows-desktop-input-d44d95b6e97d741a32cbf87323bf78c824c3155b-1'}};
 const workflow=step=>JSON.stringify({on:{workflow_dispatch:{}},jobs:{diagnostic:{steps:[step]}}});
 assert.deepEqual(validateYAML('.github/workflows/installer-invocation-diagnostic.yml',workflow(download)),[]);
 assert.match(validateYAML('.github/workflows/ci.yml',workflow(download)).join('\n'),/run_attempt/);
 for(const changed of [{...download,if:undefined},{...download,with:{...download.with,'run-id':'dynamic'}},{...download,with:{...download.with,name:'windows-desktop-input-main-1'}}])assert.match(validateYAML('.github/workflows/installer-invocation-diagnostic.yml',workflow(changed)).join('\n'),/run_attempt/);
 const automatic=JSON.stringify({on:{push:{}},jobs:{diagnostic:{steps:[download]}}});
 assert.match(validateYAML('.github/workflows/installer-invocation-diagnostic.yml',automatic).join('\n'),/run_attempt/);
});
