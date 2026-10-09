// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict';
import { cpSync, mkdtempSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import test from 'node:test';
import { parse } from 'yaml';
import { validateOwnedVersions } from '../scripts/check-schemas.mjs';
const source = new URL('../', import.meta.url);

test('release version checks include desktop locks and the shared runtime binding', () => {
  const temporary = mkdtempSync(join(tmpdir(), 'insonic-version-'));
  try {
    for (const filename of ['VERSION', 'docs/versions.json', 'internal/contracts/contracts.go', ...['', 'site/', 'desktop/frontend/'].flatMap(prefix => [`${prefix}package.json`, `${prefix}package-lock.json`])]) {
      const target = join(temporary, filename); mkdirSync(dirname(target), { recursive: true }); cpSync(new URL(filename, source), target);
    }
    const version = readFileSync(new URL('VERSION', source), 'utf8').trim(); assert.equal(validateOwnedVersions(temporary), version);
    const filename = join(temporary, 'desktop/frontend/package-lock.json'); const lock = JSON.parse(readFileSync(filename, 'utf8'));
    lock.packages[''].version = '999.999.999'; writeFileSync(filename, JSON.stringify(lock));
    assert.throws(() => validateOwnedVersions(temporary), /package\/lock/);
    lock.packages[''].version = version; writeFileSync(filename, JSON.stringify(lock));
    writeFileSync(join(temporary, 'internal/contracts/contracts.go'), 'const Version = "999.999.999"\n');
    assert.throws(() => validateOwnedVersions(temporary), /Runtime contract/);
  } finally { rmSync(temporary, { recursive: true, force: true }); }
});

test('documentation promotion can run independently and never implicitly publishes', () => {
  const workflow = parse(readFileSync(new URL('.github/workflows/release.yml', source), 'utf8'));
  const job = workflow.jobs.publish;
  assert.equal(job.if, 'inputs.publish || inputs.promote_docs');
  assert.equal(job.permissions.checks, 'read');
  const publication = job.steps.find(step => step.run?.startsWith('python scripts/release.py publish '));
  const preparation = job.steps.find(step => step.run?.startsWith('python scripts/release.py deployment '));
  const promotion = job.steps.find(step => step.run?.startsWith('python scripts/release.py promote '));
  assert.equal(publication.if, 'inputs.publish');
  assert.equal(preparation.if, 'inputs.promote_docs');
  assert.equal(promotion.if, 'inputs.promote_docs');
  assert.ok(job.steps.indexOf(preparation) < job.steps.indexOf(publication));
  assert.ok(job.steps.indexOf(publication) < job.steps.indexOf(promotion));
  assert.ok(preparation.run.includes('build/release/documentation-deployment.json'));
  assert.ok(promotion.run.includes('build/release/documentation-deployment.json'));
});

test('media and release jobs select the toolkit installed by the pinned MSYS2 action', () => {
  const media = parse(readFileSync(new URL('.github/workflows/media-source.yml', source), 'utf8'));
  const release = parse(readFileSync(new URL('.github/workflows/release.yml', source), 'utf8'));
  for (const job of [media.jobs.dependencies, media.jobs.ffmpeg, release.jobs.native]) {
    const setup = job.steps.find(step => step.uses?.startsWith('msys2/setup-msys2@'));
    const selection = job.steps.find(step => step.env?.MSYS2_ACTION_LOCATION);
    assert.equal(setup.id, 'msys2');
    assert.equal(selection.env.MSYS2_ACTION_LOCATION, '${{ steps.msys2.outputs.msys2-location }}');
    assert.equal(selection.if, "runner.os == 'Windows'");
    assert.ok(selection.run.includes('GITHUB_ENV'));
    assert.ok(selection.run.includes('INSONIC_MSYS2_ROOT='));
    assert.ok(job.steps.indexOf(setup) < job.steps.indexOf(selection));
    const consumer = job.steps.find(step => step.run?.includes('scripts/media-transfer.py identity') || step.run?.includes('scripts/media-transfer.py restore'));
    assert.ok(job.steps.indexOf(selection) < job.steps.indexOf(consumer));
    assert.ok(setup.with.install.includes('mingw-w64-ucrt-x86_64-nasm'));
  }
});
