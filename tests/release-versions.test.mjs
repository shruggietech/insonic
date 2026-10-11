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
  for (const os of ['linux', 'windows', 'macos']) assert.equal(workflow.jobs[`media-${os}`].if, 'inputs.publish || !inputs.promote_docs');
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

test('media jobs select the toolkit installed by the pinned MSYS2 action', () => {
  const media = parse(readFileSync(new URL('.github/workflows/media-source.yml', source), 'utf8'));
  for (const job of Object.values(media.jobs)) {
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

test('release qualification reuses parallel native/package lanes and matching release-target docs', () => {
  const workflow = parse(readFileSync(new URL('.github/workflows/release.yml', source), 'utf8'));
  const native = parse(readFileSync(new URL('.github/workflows/native-qualification.yml', source), 'utf8'));
  const calls = ['linux', 'windows', 'macos'].flatMap(os => [`qualify-${os}`, `package-${os}`]);
  for (const name of calls) {
    const job = workflow.jobs[name];
    assert.equal(job.uses, './.github/workflows/native-qualification.yml');
    assert.equal(job.with.revision, '${{ inputs.revision }}');
    assert.equal(job.with['documentation-target'], 'release');
    assert.equal(job.with.lane, name.startsWith('package-') ? 'packages' : 'runtime');
    assert.ok(!JSON.stringify(job).includes('secrets.'));
  }
  assert.ok(calls.every(name => workflow.jobs.candidate.needs.includes(name)));
  assert.ok(workflow.jobs.candidate.needs.includes('docs'));
  assert.equal(workflow.jobs.docs.env.INSONIC_DOCUMENTATION_TARGET, 'release');
  assert.ok(workflow.jobs.candidate.steps.some(step => step.with?.pattern === 'candidate-package-*-${{ inputs.revision }}-${{ github.run_attempt }}'));
  assert.ok(native.jobs.runtime_checks.strategy.matrix.runtime_lane.includes('engine'));
  assert.ok(native.jobs.runtime_checks.strategy.matrix.runtime_lane.includes('desktop'));
  assert.deepEqual(native.jobs.runtime.needs, ['runtime_checks']);
  assert.equal(native.jobs.runtime.name, 'native (${{ inputs.os }})');
  assert.ok(native.jobs.runtime.if.includes('always()'));
  const steps = native.jobs.runtime_checks.steps;
  const cache = steps.find(step => step.uses?.startsWith('actions/setup-go@'));
  assert.ok(cache.with['cache-dependency-path'].includes('cache-keys/native-'));
  assert.ok(steps.find(step => step.run?.includes('qualify.py cli')).run.includes('--native'));
  for (const job of [native.jobs.runtime_checks, native.jobs.packages]) assert.equal(job.timeout_minutes ?? job['timeout-minutes'], 10);
  assert.equal(native.on.workflow_call.inputs['documentation-target'].default, 'snapshot');
});

test('deployment routes separate Pages authority from generic host secrets', () => {
  const workflow = parse(readFileSync(new URL('.github/workflows/release.yml', source), 'utf8'));
  assert.equal(workflow.on.workflow_dispatch.inputs['documentation-transport'].default, 'pages');
  const preflight = workflow.jobs['pages-preflight'], promotion = workflow.jobs['promote-pages'];
  assert.equal(preflight.permissions.pages, 'read');
  assert.equal(promotion.permissions.pages, 'write');
  assert.equal(promotion.permissions.contents, 'write');
  for (const job of [preflight, promotion]) {
    assert.equal(job.environment, undefined);
    assert.ok(job.if.includes("github.ref == 'refs/heads/main'"));
    assert.ok(job.if.includes("inputs.documentation-transport == 'pages'"));
    assert.ok(!JSON.stringify(job).includes('secrets.'));
    assert.equal(job.steps[0].with.ref, 'main');
    assert.equal(job.steps[0].with['persist-credentials'], false);
  }
  assert.ok(promotion.steps.some(step => step.run?.includes('release.py published --source-root')));
  assert.ok(promotion.steps.some(step => step.run?.includes('release.py pages-promote --source-root')));
  assert.ok(workflow.jobs.promote.if.includes("inputs.documentation-transport == 'generic'"));
});
