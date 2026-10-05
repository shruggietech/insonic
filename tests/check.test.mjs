// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict';
import { mkdtemp, mkdir, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';
import { checkRepository, markdownAnchors, markdownTargets, validateLinks, validateMarkdown, validateText, validateVersions } from '../scripts/check.mjs';

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
