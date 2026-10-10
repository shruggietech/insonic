import test from 'node:test';
import assert from 'node:assert/strict';
import { classify } from '../scripts/ci-scope.mjs';
import { readFileSync } from 'node:fs';
import { parse } from 'yaml';

test('prose keeps integrity gates without native downloads',()=>{
 assert.deepEqual(classify(['docs/v0.0.0/storage.md','specs/002-portable-catalogs/tasks.md']),{core:false,cli:false,native:false,adapters:false});
});
test('catalog keeps platform and server acceptance',()=>{
 assert.deepEqual(classify(['internal/catalog/jobs.go']),{core:true,cli:true,native:true,adapters:true});
});
test('native/dependency/workflow inputs receive full qualification',()=>{
 for(const path of ['go.mod','go.sum','internal/qualification/dependencies.json','.github/workflows/ci.yml','desktop/bridge.go','scripts/qualify.py','schemas/v0.0.0/runtime-request.schema.json','unknown/new.file']){
  assert.equal(classify([path]).native,true,path);
 }
});
test('invalid or unavailable diffs conservatively qualify everything',()=>{
 assert.equal(classify(null).native,true);
 assert.equal(classify(['../../escape']).native,true);
});

test('platform qualification waits only for its own exact same-run source build', () => {
 const ci = parse(readFileSync(new URL('../.github/workflows/ci.yml', import.meta.url), 'utf8'));
 const qualification = parse(readFileSync(new URL('../.github/workflows/native-qualification.yml', import.meta.url), 'utf8'));
 assert.equal(ci.jobs.media, undefined);
 for (const [label, os] of [['linux', 'ubuntu-24.04'], ['windows', 'windows-2022'], ['macos', 'macos-15']]) {
  const media = ci.jobs[`media-${label}`], native = ci.jobs[`qualify-${label}`];
  assert.equal(media.needs, 'foundation');
  assert.equal(media.uses, './.github/workflows/media-source.yml');
  assert.equal(media.with.os, os);
  assert.equal(media.with.revision, '${{ github.sha }}');
  assert.deepEqual(native.needs, ['foundation', `media-${label}`]);
  assert.ok(native.if.includes(`needs.media-${label}.result == 'success'`));
  assert.equal(native.uses, './.github/workflows/native-qualification.yml');
  assert.equal(native.with.os, os);
  assert.equal(native.with.revision, '${{ github.sha }}');
  for (const field of ['core', 'cli', 'native']) assert.equal(native.with[field], '${{ needs.foundation.outputs.' + field + " == 'true' }}");
 }
 assert.equal(ci.jobs.core.needs, 'foundation');
 assert.deepEqual(ci.jobs.core.strategy.matrix.os, ['ubuntu-24.04', 'macos-15']);
 assert.equal(ci.jobs['core-windows'].needs, 'foundation');
 for (const job of [ci.jobs.core, ci.jobs['core-windows']]) {
  const commands = job.steps.map(step => step.run ?? '').join('\n');
  for (const command of ['go test ./...', 'go vet ./...', "python -m unittest discover -s tests -p '*_test.py'"]) assert.ok(commands.includes(command));
 }
 const runtime = qualification.jobs.runtime;
 const commands = runtime.steps.map(step => step.run ?? '').join('\n');
 for (const command of ['python scripts/media-tools.py', 'python scripts/qualify.py prepare', 'python scripts/qualify.py native', 'python scripts/qualify.py cli', 'python scripts/qualify.py secrets', 'python scripts/qualify.py desktop', 'npm --prefix desktop/frontend run check', 'npm --prefix desktop/frontend test', 'npm --prefix desktop/frontend run build', 'npm --prefix site run build']) assert.ok(commands.includes(command), command);
 assert.ok(!commands.includes('go test ./...')); // Platform core tests now overlap source compilation.
 assert.deepEqual(qualification.jobs.packages.strategy.matrix.variant, ['cli', 'desktop']);
 assert.equal(qualification.jobs.packages.needs, undefined); // Runtime and each package qualify independently.
 for (const job of [runtime, qualification.jobs.packages]) {
  assert.equal(job['runs-on'], '${{ inputs.os }}');
  assert.equal(job.env.SOURCE_REVISION, '${{ inputs.revision }}');
  assert.equal(job.steps.find(step => step.uses?.startsWith('actions/checkout@')).with.ref, '${{ inputs.revision }}');
  assert.equal(job.steps[0].with['persist-credentials'], false);
  assert.ok(job.steps.some(step => step.with?.name === 'media-source-${{ inputs.os }}-${{ github.run_attempt }}'));
  assert.ok(job.steps.some(step => step.run?.includes('media-transfer.py restore --stage complete --revision "$SOURCE_REVISION"')));
  const apt = job.steps.find(step => step.name === 'Linux webview dependencies');
  assert.ok(apt.run.includes('sudo python3 scripts/prepare-apt-mirror.py'));
  assert.ok(apt.run.includes('--no-install-recommends libgtk-3-dev libwebkit2gtk-4.1-dev'));
 }
 const packages = qualification.jobs.packages.steps.map(step => step.run ?? '').join('\n');
 assert.ok(packages.includes('scripts/package-desktop.py all --variant ${{ matrix.variant }}'));
 assert.ok(packages.includes('scripts/media-tools.py --prepare-only'));
 assert.ok(packages.includes('npm --prefix desktop/frontend ci --ignore-scripts'));
 assert.deepEqual(ci.jobs.adapters.needs, ['foundation', 'media-linux']);
 assert.deepEqual(Object.keys(ci.jobs.adapters.services).sort(), ['arcade', 'postgres', 's3']);
 const backends = ci.jobs.adapters.steps.map(step => step.run ?? '').join('\n');
 for (const target of ['qualification', 'catalog', 'artifact', 'models', 'library', 'graph', 'app', 'backup']) assert.ok(backends.includes(`./internal/${target}`));
});

test('native caches can retain tagged builds independently of the immutable core cache', () => {
 const qualification = parse(readFileSync(new URL('../.github/workflows/native-qualification.yml', import.meta.url), 'utf8'));
 const goCache = job => job.steps.find(step => step.uses?.startsWith('actions/setup-go@')).with['cache-dependency-path'];
 const runtime = goCache(qualification.jobs.runtime).trim().split('\n');
 const packageInputs = goCache(qualification.jobs.packages).trim().split('\n');
 assert.deepEqual(runtime, ['go.sum', 'internal/qualification/dependencies.json', 'desktop/frontend/package-lock.json']);
 assert.deepEqual(packageInputs.slice(0, 2), runtime.slice(0, 2));
 const select = variant => packageInputs.map(value => value.startsWith('${{') ? (variant === 'desktop' ? 'desktop/frontend/package-lock.json' : 'internal/contracts/contracts.go') : value);
 assert.deepEqual(select('desktop'), runtime);
 assert.notDeepEqual(select('cli'), runtime);
 for (const value of [...runtime, ...select('cli')]) assert.ok(readFileSync(new URL('../' + value, import.meta.url)).length);
 const ci = parse(readFileSync(new URL('../.github/workflows/ci.yml', import.meta.url), 'utf8'));
 for (const job of [ci.jobs.core, ci.jobs['core-windows']]) {
  assert.equal(goCache(job), undefined); // The default go.sum core key cannot capture native additions.
 }
});

test('cold source dependency formats and FFmpeg transfer independently verified provenance', () => {
 const media = parse(readFileSync(new URL('../.github/workflows/media-source.yml', import.meta.url), 'utf8'));
 assert.deepEqual(media.jobs.formats.needs, ['probe', 'groups']);
 assert.deepEqual(media.jobs.ffmpeg.needs, ['probe', 'formats']);
 assert.deepEqual(media.jobs.groups.strategy.matrix.group, ['compression', 'audio', 'av1']);
 const formats = media.jobs.formats.steps.map(step => step.run ?? '').join('\n');
 assert.ok(formats.includes('media-transfer.py merge --revision "$SOURCE_REVISION"'));
 assert.ok(formats.includes('scripts/build-media-source.py --stage dependencies'));
 assert.ok(formats.includes('media-transfer.py pack --stage dependencies --revision "$SOURCE_REVISION"'));
 assert.ok(!formats.includes('build-media-source.py --stage complete'));
 const ffmpeg = media.jobs.ffmpeg.steps.map(step => step.run ?? '').join('\n');
 assert.ok(ffmpeg.includes('media-transfer.py restore --stage dependencies --revision "$SOURCE_REVISION"'));
 assert.ok(ffmpeg.includes('build-media-source.py --stage complete'));
 assert.ok(ffmpeg.includes('media-transfer.py pack --stage complete --revision "$SOURCE_REVISION"'));
 assert.ok(media.jobs.ffmpeg.steps.some(step => step.with?.name === 'media-formats-${{ inputs.os }}-${{ github.run_attempt }}'));
 for (const job of Object.values(media.jobs)) {
  assert.equal(job['timeout-minutes'], 10);
  assert.equal(job.env.SOURCE_REVISION, '${{ inputs.revision }}');
  assert.equal(job.steps.find(step => step.uses?.startsWith('actions/checkout@')).with.ref, '${{ inputs.revision }}');
 }
});

test('full workflow attempts preserve old artifacts and consume only their matching producers', () => {
 const workflow = name => parse(readFileSync(new URL(`../.github/workflows/${name}.yml`, import.meta.url), 'utf8'));
 const transfers = documents => documents.flatMap(document => Object.values(document.jobs).flatMap(job => job.steps ?? []))
  .filter(step => /actions\/(?:upload|download)-artifact@/.test(step.uses ?? ''));
 const expand = (template, values) => template.replace(/\$\{\{\s*([^}]+?)\s*\}\}/g, (_, field) => {
  assert.ok(Object.hasOwn(values, field), `unknown artifact field: ${field}`);
  return values[field];
 });
 const matches = (selector, value) => new RegExp('^' + selector.split('*').map(part => part.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join('.*') + '$').test(value);
 for (const documents of [[workflow('ci'), workflow('native-qualification'), workflow('media-source')], [workflow('release'), workflow('media-source')]]) {
  const steps = transfers(documents);
  for (const step of steps) {
   assert.ok((step.with.name ?? step.with.pattern).endsWith('-${{ github.run_attempt }}'));
   if (step.uses.includes('/upload-')) assert.ok(!step.with.overwrite); // Keep earlier attempt evidence immutable.
  }
  const platforms = ['ubuntu-24.04', 'windows-2022', 'macos-15'];
  const variants = ['cli', 'desktop'], groups = ['compression', 'audio', 'av1'];
  const names = attempt => new Set(steps.filter(step => step.uses.includes('/upload-')).flatMap(step =>
   platforms.flatMap(os => variants.flatMap(variant => groups.map(group => expand(step.with.name, {
     'github.run_attempt': String(attempt), 'github.sha': 'a'.repeat(40),
     'inputs.revision': 'a'.repeat(40), 'inputs.os': os, 'matrix.os': os,
     'matrix.variant': variant, 'matrix.group': group,
    }))))));
  const oldNames = names(1), currentNames = names(2);
  assert.equal([...currentNames].some(name => oldNames.has(name)), false);
  for (const os of platforms) {
   for (const step of steps.filter(step => step.uses.includes('/download-'))) {
    const selector = expand(step.with.name ?? step.with.pattern, {
     'github.run_attempt': '2', 'github.sha': 'a'.repeat(40),
     'inputs.revision': 'a'.repeat(40), 'inputs.os': os, 'matrix.os': os,
     'matrix.variant': 'desktop', 'matrix.group': 'compression',
    });
    assert.ok([...currentNames].some(name => matches(selector, name)), `missing same-attempt producer: ${selector}`);
    assert.equal([...oldNames].some(name => matches(selector, name)), false, `stale attempt matched: ${selector}`);
   }
  }
 }
});
