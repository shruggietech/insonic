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
  assert.ok(job.steps.some(step => step.with?.name === 'media-source-${{ inputs.os }}'));
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
 assert.ok(media.jobs.ffmpeg.steps.some(step => step.with?.name === 'media-formats-${{ inputs.os }}'));
 for (const job of Object.values(media.jobs)) {
  assert.equal(job['timeout-minutes'], 10);
  assert.equal(job.env.SOURCE_REVISION, '${{ inputs.revision }}');
  assert.equal(job.steps.find(step => step.uses?.startsWith('actions/checkout@')).with.ref, '${{ inputs.revision }}');
 }
});
