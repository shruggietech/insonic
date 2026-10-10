// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict';
import { cpSync, mkdtempSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import test from 'node:test';
import { parse } from 'yaml';
import { validateOwnedVersions } from '../scripts/check-schemas.mjs';
const source = new URL('../', import.meta.url);

test('PR candidate collection uses complete independent package lanes with no publication authority', () => {
  const workflow = parse(readFileSync(new URL('.github/workflows/ci.yml', source), 'utf8'));
  const native = parse(readFileSync(new URL('.github/workflows/native-qualification.yml', source), 'utf8'));
  const candidate = workflow.jobs.candidate;
  assert.equal(candidate.permissions.contents, 'read');
  assert.equal(candidate.environment, undefined);
  assert.equal(candidate.env.SOURCE_REVISION, '${{ github.sha }}');
  assert.deepEqual(candidate.needs, ['docs', 'package-linux', 'package-windows', 'package-macos']);
  assert.ok(!JSON.stringify(candidate).includes('secrets.'));
  assert.ok(!JSON.stringify(candidate).includes('scripts/release.py publish'));
  assert.equal(candidate.steps[0].with['persist-credentials'], false);
  assert.ok(candidate.steps.some(step => step.run?.includes('release.py collect')));
  assert.equal(native.on.workflow_call.inputs.lane.default, 'both');
  assert.ok(native.jobs.runtime.if.includes("inputs.lane != 'packages'"));
  assert.ok(native.jobs.packages.if.includes("inputs.lane != 'runtime'"));
  for (const platform of ['linux', 'windows', 'macos']) {
    const runtime = workflow.jobs[`qualify-${platform}`], packages = workflow.jobs[`package-${platform}`];
    assert.equal(runtime.with.lane, 'runtime');
    assert.equal(packages.with.lane, 'packages');
    assert.equal(packages.with.revision, '${{ github.sha }}');
    assert.deepEqual(packages.needs, runtime.needs);
  }
  const assets = native.jobs.packages.steps.find(step => step.with?.name?.startsWith('candidate-package-'));
  assert.ok(assets.if.includes('success()'));
  assert.ok(assets.with.path.includes('build/packages/*.zip'));
  assert.ok(assets.with.path.includes('build/packages/*.tar.gz'));
  assert.ok(workflow.jobs.docs.steps.some(step => step.run?.includes('release.py docs')));
});

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

test('publication and promotion execute trusted tools and isolate deployment credentials', () => {
  const workflow = parse(readFileSync(new URL('.github/workflows/release.yml', source), 'utf8'));
  const job = workflow.jobs.publish;
  assert.ok(job.if.includes('inputs.publish && needs.candidate.result'));
  assert.ok(job.if.includes("needs.deployment-preflight.result == 'success'"));
  assert.equal(workflow.jobs.media.if, 'inputs.publish || !inputs.promote_docs');
  assert.equal(job.permissions.checks, 'read');
  assert.equal(job.permissions.actions, 'read');
  const publication = job.steps.find(step => step.run?.startsWith('python scripts/release.py publish '));
  assert.ok(publication.run.includes('--source-root build/release/source'));
  assert.equal(job.env.INSONIC_DOCUMENTATION_DEPLOYMENT_CREDENTIAL, undefined);
  assert.equal(job.env.INSONIC_DOCUMENTATION_DEPLOYMENT_JSON, undefined);
  const preflight = workflow.jobs['deployment-preflight'];
  const promote = workflow.jobs.promote;
  assert.ok(promote.if.includes('!inputs.publish || needs.publish.result'));
  for (const protectedJob of [preflight, promote]) {
    assert.equal(protectedJob.environment, 'documentation-deployment');
    assert.equal(protectedJob.permissions.contents, 'read');
    assert.equal(protectedJob.env.INSONIC_DOCUMENTATION_DEPLOYMENT_CREDENTIAL, '${{ secrets.INSONIC_DOCUMENTATION_DEPLOYMENT_CREDENTIAL }}');
    assert.ok(protectedJob.steps.some(step => step.run?.startsWith('python scripts/release.py deployment ')));
  }
  assert.ok(promote.steps.some(step => step.run?.startsWith('python scripts/release.py published --source-root build/release/source')));
  assert.ok(promote.steps.some(step => step.run?.startsWith('python scripts/release.py promote ')));
  assert.ok(!promote.steps.some(step => step.run?.includes('scripts/package-desktop.py') || step.run?.includes('scripts/release.py publish ')));
  for (const trustedJob of [job, preflight, promote]) {
    assert.ok(trustedJob.if.includes("github.ref == 'refs/heads/main'"));
    const checkout = trustedJob.steps[0];
    assert.equal(checkout.with.ref, 'main');
    assert.equal(checkout.with['persist-credentials'], false);
    const gate = trustedJob.steps.findIndex(step => step.run?.startsWith('python scripts/release.py trusted '));
    assert.equal(gate, 1);
    const data = trustedJob.steps.findIndex(step => step.with?.path === 'build/release/source');
    if (data >= 0) {
      assert.ok(data > gate);
      assert.equal(trustedJob.steps[data].with.ref, '${{ inputs.revision }}');
      assert.equal(trustedJob.steps[data].with['persist-credentials'], false);
    }
    assert.ok(trustedJob.steps.every(step => !step.run?.includes('build/release/source/scripts/')));
  }
});

test('media and release jobs select the toolkit installed by the pinned MSYS2 action', () => {
  const media = parse(readFileSync(new URL('.github/workflows/media-source.yml', source), 'utf8'));
  const release = parse(readFileSync(new URL('.github/workflows/release.yml', source), 'utf8'));
  for (const job of [...Object.values(media.jobs), release.jobs.native]) {
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

test('fresh release qualification prepares its sibling CLI before the macOS desktop app', () => {
  const workflow = parse(readFileSync(new URL('.github/workflows/release.yml', source), 'utf8'));
  const steps = workflow.jobs.native.steps;
  const cli = steps.findIndex(step => step.run?.includes('python scripts/qualify.py cli'));
  const desktop = steps.findIndex(step => step.run?.includes('python scripts/qualify.py desktop'));
  const packages = steps.findIndex(step => step.run?.includes('python scripts/package-desktop.py all --variant both'));
  assert.ok(cli >= 0 && cli < desktop);
  assert.ok(desktop < packages);
  const linux = steps.find(step => step.name === 'Native Linux dependencies');
  assert.ok(linux.run.includes('sudo python3 scripts/prepare-apt-mirror.py'));
  assert.ok(linux.run.includes('Acquire::https::Timeout=15'));
});
