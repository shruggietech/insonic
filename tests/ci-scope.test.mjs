import test from 'node:test';
import assert from 'node:assert/strict';
import { classify, selectedScope } from '../scripts/ci-scope.mjs';
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

test('only an explicitly elected maintenance smoke avoids duplicate native packages', () => {
 assert.equal(selectedScope(null, 'workflow_dispatch', 'true').native, false);
 for (const event of ['push','pull_request','workflow_dispatch']) {
  assert.equal(selectedScope(null,event,'false').native,true);
  if(event!=='workflow_dispatch')assert.equal(selectedScope(null,event,'true').native,true);
 }
});

test('platform qualification waits only for its own exact same-run source build', () => {
 const ci = parse(readFileSync(new URL('../.github/workflows/ci.yml', import.meta.url), 'utf8'));
 const qualification = parse(readFileSync(new URL('../.github/workflows/native-qualification.yml', import.meta.url), 'utf8'));
 assert.equal(ci.jobs.media, undefined);
 for (const [label, os] of [['linux', 'ubuntu-24.04'], ['windows', 'windows-2022']]) {
  const media = ci.jobs[`media-${label}`], native = ci.jobs[`qualify-${label}`];
  assert.equal(media.needs, 'scope');
  assert.equal(media.uses, './.github/workflows/media-source.yml');
  assert.equal(media.with.os, os);
  assert.equal(media.with.revision, '${{ github.sha }}');
  assert.deepEqual(native.needs, ['scope', 'foundation', 'docs-build', `media-${label}`]);
  assert.ok(native.if.includes(`needs.media-${label}.result == 'success'`));
  assert.equal(native.uses, './.github/workflows/native-qualification.yml');
  assert.equal(native.with.os, os);
  assert.equal(native.with.revision, '${{ github.sha }}');
  for (const field of ['core', 'cli', 'native']) assert.equal(native.with[field], '${{ needs.scope.outputs.' + field + " == 'true' }}");
 }
 assert.deepEqual(ci.jobs.core.needs, ['foundation','scope']);
 assert.deepEqual(ci.jobs.core.strategy.matrix.os, ['ubuntu-24.04']);
 assert.deepEqual(ci.jobs['core-windows'].needs, ['foundation','scope']);
 for (const job of [ci.jobs.core, ci.jobs['core-windows']]) {
  const commands = job.steps.map(step => step.run ?? '').join('\n');
  for (const command of ['go test ./...', 'go vet ./...', "python -m unittest discover -s tests -p '*_test.py'"]) assert.ok(commands.includes(command));
 }
 const runtime = qualification.jobs.runtime_checks;
 const commands = runtime.steps.map(step => step.run ?? '').join('\n');
 for (const command of ['python scripts/media-tools.py', 'python scripts/qualify.py prepare', 'python scripts/qualify.py native', 'python scripts/qualify.py cli', 'python scripts/qualify.py secrets', 'python scripts/qualify.py desktop', 'npm --prefix desktop/frontend run check', 'npm --prefix desktop/frontend test', 'npm --prefix desktop/frontend run build', 'npm --prefix site run build']) assert.ok(commands.includes(command), command);
 assert.ok(!commands.includes('go test ./...')); // Platform core tests now overlap source compilation.
 assert.equal(qualification.jobs.packages.strategy.matrix.variant, '${{ inputs.os == \'windows-2022\' && fromJSON(\'["cli"]\') || fromJSON(\'["cli","desktop"]\') }}');
 const windows = qualification.jobs['windows-desktop'];
 assert.equal(windows.uses, './.github/workflows/windows-desktop-qualification.yml');
 assert.equal(windows.with.revision, '${{ inputs.revision }}');
 assert.equal(windows.if, "inputs.native && inputs.os == 'windows-2022' && inputs.lane != 'runtime'");
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
 assert.deepEqual(ci.jobs.adapters.needs, ['scope','foundation', 'media-linux']);
 assert.deepEqual(Object.keys(ci.jobs.adapters.services).sort(), ['arcade', 'postgres', 's3']);
 const backends = ci.jobs.adapters.steps.map(step => step.run ?? '').join('\n');
 for (const target of ['qualification', 'catalog', 'artifact', 'models', 'library', 'graph', 'app', 'backup']) assert.ok(backends.includes(`./internal/${target}`));
});

test('Windows delivery joins every isolated scope before exposing candidate artifacts', () => {
 const workflow = parse(readFileSync(new URL('../.github/workflows/windows-desktop-qualification.yml', import.meta.url), 'utf8'));
 const {assemble, proofs, finalize} = workflow.jobs;
 assert.equal(proofs.needs, undefined);
 const proofWait = proofs.steps.findIndex(s => s.run?.includes('wait-ci-artifact.mjs windows-desktop-input'));
 const proofDownload = proofs.steps.findIndex(s => s.name === 'Restore the same immutable Windows delivery');
 assert.ok(proofWait > proofs.steps.findIndex(s => s.uses === './build/actions/setup-go'));
 assert.ok(proofWait > proofs.steps.findIndex(s => s.uses === './build/actions/setup-node'));
 assert.ok(proofDownload > proofWait);
 assert.equal(proofs.steps[proofWait].env.GH_TOKEN, '${{ github.token }}');
 assert.ok(proofs.steps[proofWait].run.includes("'${{ inputs.revision }}'"));
 assert.deepEqual(proofs.strategy.matrix.scope, ['archive', 'interactive', 'maintenance', 'setup']);
 assert.equal(proofs.strategy['fail-fast'], false);
 assert.deepEqual(finalize.needs, ['assemble']);
 const prepared = finalize.steps.findIndex(s => s.name === 'Restore the same immutable Windows delivery');
 const wait = finalize.steps.findIndex(s => s.run?.includes('wait-ci-artifact.mjs windows-desktop-proofs'));
 const download = finalize.steps.findIndex(s => s.name === 'Download each completed qualification scope');
 assert.ok(prepared < wait && wait < download);
 assert.equal(finalize.steps[wait].env.GH_TOKEN, '${{ github.token }}');
 assert.ok(finalize.steps[wait].run.includes("'${{ inputs.revision }}'"));
 for (const job of [assemble, proofs, finalize]) {
  assert.equal(job['runs-on'], 'windows-2022');
  assert.equal(job['timeout-minutes'], 65);
  assert.equal(job.env.SOURCE_REVISION, '${{ inputs.revision }}');
  assert.equal(job.steps[0].with.ref, '${{ inputs.revision }}');
 }
 assert.equal(assemble.steps.filter(s => s.run?.includes('windows_package_pipeline.py assemble')).length, 1);
 assert.ok(!proofs.steps.some(s => s.run?.includes('package-desktop.py build')));
 for (const job of [proofs, finalize]) {
  const transfer = job.steps.find(s => s.name === 'Restore the same immutable Windows delivery');
  assert.equal(transfer.with.name, 'windows-desktop-input-${{ inputs.revision }}-${{ github.run_attempt }}');
  assert.equal(transfer.with['digest-mismatch'], 'error');
 }
 for (const job of [assemble, proofs]) assert.ok(!job.steps.some(s => s.with?.name?.startsWith('candidate-package-')));
 const joined = finalize.steps.findIndex(s => s.run?.includes('windows_package_pipeline.py finalize'));
 for (const [index, step] of finalize.steps.entries()) if (step.with?.name?.startsWith('candidate-package-')) {
  assert.ok(index > joined);
  assert.ok(!step.if || step.if === 'success()');
 }
});

test('native caches can retain tagged builds independently of the immutable core cache', () => {
 const qualification = parse(readFileSync(new URL('../.github/workflows/native-qualification.yml', import.meta.url), 'utf8'));
 assert.equal(qualification.jobs.runtime_checks.steps.find(step => step.name === 'Cross-process CLI acceptance').run,
  "python scripts/qualify.py cli ${{ inputs.native && '--native' || '' }}");
 const goCache = job => job.steps.find(step => step.uses === './build/actions/setup-go').with['cache-dependency-path'];
 const runtime = goCache(qualification.jobs.runtime_checks).trim().split('\n');
 const packageInputs = goCache(qualification.jobs.packages).trim().split('\n');
 assert.deepEqual(runtime, ['go.sum', 'internal/qualification/dependencies.json', '.github/workflows/cache-keys/native-${{ matrix.runtime_lane }}']);
 assert.deepEqual(packageInputs.slice(0, 2), runtime.slice(0, 2));
 const select = variant => packageInputs.map(value => value.startsWith('${{') ? (variant === 'desktop' ? 'desktop/frontend/package-lock.json' : 'internal/contracts/contracts.go') : value);
 const engine = runtime.map(value => value.replace('${{ matrix.runtime_lane }}', 'engine'));
 const desktop = runtime.map(value => value.replace('${{ matrix.runtime_lane }}', 'desktop'));
 assert.equal(new Set([engine.join('\n'), desktop.join('\n'), select('cli').join('\n'), select('desktop').join('\n')]).size, 4);
 for (const value of [...engine, ...desktop, ...select('cli'), ...select('desktop')]) assert.ok(readFileSync(new URL('../' + value, import.meta.url)).length);
 const ci = parse(readFileSync(new URL('../.github/workflows/ci.yml', import.meta.url), 'utf8'));
 for (const job of [ci.jobs.core, ci.jobs['core-windows']]) {
  assert.equal(goCache(job), undefined); // The default go.sum core key cannot capture native additions.
 }
});

test('cold source dependency formats and FFmpeg transfer independently verified provenance', () => {
 const media = parse(readFileSync(new URL('../.github/workflows/media-source.yml', import.meta.url), 'utf8'));
 assert.deepEqual(media.jobs.formats.needs, ['probe', 'groups']);
 assert.deepEqual(media.jobs.codec.needs, ['probe', 'formats']);
 assert.ok(media.jobs.codec.if.includes("inputs.os == 'windows-2022'"));
 assert.deepEqual(media.jobs.ffmpeg.needs, ['probe', 'formats', 'codec']);
 assert.ok(media.jobs.ffmpeg.if.includes('always()'));
 assert.ok(media.jobs.ffmpeg.if.includes("needs.codec.result == 'success'"));
 const codec = media.jobs.codec.steps.map(step => step.run ?? '').join('\n');
 assert.ok(codec.includes('build-media-source.py --stage codec'));
 assert.ok(codec.includes('media-transfer.py pack --stage codec --revision "$SOURCE_REVISION"'));
 assert.deepEqual(media.jobs.groups.strategy.matrix.group, ['compression', 'audio', 'av1']);
 const formats = media.jobs.formats.steps.map(step => step.run ?? '').join('\n');
 assert.ok(formats.includes('media-transfer.py merge --revision "$SOURCE_REVISION"'));
 assert.ok(formats.includes('scripts/build-media-source.py --stage dependencies'));
 assert.ok(formats.includes('media-transfer.py pack --stage dependencies --revision "$SOURCE_REVISION"'));
 assert.ok(!formats.includes('build-media-source.py --stage complete'));
 const ffmpeg = media.jobs.ffmpeg.steps.map(step => step.run ?? '').join('\n');
 assert.ok(ffmpeg.includes('media-transfer.py restore --stage dependencies --revision "$SOURCE_REVISION"'));
 assert.ok(ffmpeg.includes('build-media-source.py --stage complete'));
 assert.ok(ffmpeg.includes('media-transfer.py restore --stage codec --revision "$SOURCE_REVISION"'));
 assert.ok(ffmpeg.includes('media-transfer.py pack --stage complete --revision "$SOURCE_REVISION"'));
 assert.ok(media.jobs.ffmpeg.steps.some(step => step.with?.name === 'media-formats-${{ inputs.os }}-${{ github.run_attempt }}'));
 for (const job of Object.values(media.jobs)) {
  assert.equal(job['timeout-minutes'], 60);
  assert.equal(job.env.SOURCE_REVISION, '${{ inputs.revision }}');
  assert.equal(job.steps.find(step => step.uses?.startsWith('actions/checkout@')).with.ref, job === media.jobs.ready ? '${{ job.workflow_sha }}' : '${{ inputs.revision }}');
 }
});

test('workflow attempts avoid artifact-name collisions and consume only matching producers', () => {
 const workflow = name => parse(readFileSync(new URL(`../.github/workflows/${name}.yml`, import.meta.url), 'utf8'));
 const transfers = documents => documents.flatMap(document => Object.values(document.jobs).flatMap(job => job.steps ?? []))
  .filter(step => /actions\/(?:upload|download)-artifact@/.test(step.uses ?? ''));
 const expand = (template, values) => template.replace(/\$\{\{\s*([^}]+?)\s*\}\}/g, (_, field) => {
  assert.ok(Object.hasOwn(values, field), `unknown artifact field: ${field}`);
  return values[field];
 });
 const matches = (selector, value) => new RegExp('^' + selector.split('*').map(part => part.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join('.*') + '$').test(value);
 for (const documents of [[workflow('ci'), workflow('native-qualification'), workflow('media-source')], [workflow('release'), workflow('native-qualification'), workflow('media-source')]]) {
  const steps = transfers(documents).filter(step => documents[0].name === 'CI' || !step.if?.includes('inputs.reuse-ci-go-dependencies'));
  for (const step of steps) {
   assert.ok((step.with.name ?? step.with.pattern).endsWith('-${{ github.run_attempt }}'));
   if (step.uses.includes('/upload-')) assert.ok(!step.with.overwrite); // Workflow naming cannot guarantee platform retention.
  }
  const platforms = ['ubuntu-24.04', 'windows-2022', 'macos-15'];
  const variants = ['cli', 'desktop'], groups = ['compression', 'audio', 'av1'];
  const names = attempt => new Set(steps.filter(step => step.uses.includes('/upload-')).flatMap(step =>
   platforms.flatMap(os => variants.flatMap(variant => ['engine', 'desktop'].flatMap(runtime_lane => groups.map(group => expand(step.with.name, {
     'github.run_attempt': String(attempt), 'github.sha': 'a'.repeat(40),
     'inputs.revision': 'a'.repeat(40), 'inputs.os': os, 'matrix.os': os,
     'matrix.variant': variant, 'matrix.group': group, 'matrix.runtime_lane': runtime_lane,
    })))))));
  const oldNames = names(1), currentNames = names(2);
  assert.equal([...currentNames].some(name => oldNames.has(name)), false);
  for (const os of platforms) {
   for (const step of steps.filter(step => step.uses.includes('/download-'))) {
    const selector = expand(step.with.name ?? step.with.pattern, {
     'github.run_attempt': '2', 'github.sha': 'a'.repeat(40),
     'inputs.revision': 'a'.repeat(40), 'inputs.os': os, 'matrix.os': os,
     'matrix.variant': 'desktop', 'matrix.group': 'compression', 'matrix.runtime_lane': 'engine',
    });
    assert.ok([...currentNames].some(name => matches(selector, name)), `missing same-attempt producer: ${selector}`);
    assert.equal([...oldNames].some(name => matches(selector, name)), false, `stale attempt matched: ${selector}`);
   }
  }
 }
});

test('qualification watchdogs agree with the shared policy while unrelated limits remain scoped', () => {
 const policy=JSON.parse(readFileSync(new URL('../scripts/ci-timing-policy.json',import.meta.url),'utf8'));
 assert.deepEqual(policy,{qualification_target_seconds:1800,qualification_limit_seconds:3600,finalization_reserve_seconds:60});
 for (const name of ['ci','native-qualification','windows-desktop-qualification','media-source','release','website','installer-invocation-diagnostic']) {
  const workflow=parse(readFileSync(new URL('../.github/workflows/'+name+'.yml',import.meta.url),'utf8'));
  for(const [id,job] of Object.entries(workflow.jobs)){
   if(job.uses)continue;
   const preparation=name==='media-source'||(name==='ci'&&['scope','dependencies-windows','readiness'].includes(id));
   // Preparation and independent release watchdogs are explicit policy scopes.
   assert.equal(job['timeout-minutes'],preparation?60:65,name+'/'+id);
  }
 }
});

test('documentation CI qualifies and retains isolated public/offline browser evidence', () => {
 const ci = parse(readFileSync(new URL('../.github/workflows/ci.yml', import.meta.url), 'utf8'));
 const steps = ci.jobs.docs.steps;
 assert.ok(steps.some(step => step.run === 'npm ci --ignore-scripts'));
 assert.equal(ci.jobs.docs.needs, 'docs-build');
 const build = steps.findIndex(step => step.run?.includes('site-transfer.py restore'));
 assert.ok(build >= 0);
 assert.ok(ci.jobs['docs-build'].steps.some(step => step.run === 'npm --prefix site run build'));
 const browser = steps.findIndex(step => step.run?.includes('node scripts/check-docs-ui.mjs'));
 assert.ok(browser > build, 'browser qualification must inspect complete exports');
 assert.ok(steps[browser].run.includes('--browser /usr/bin/google-chrome'));
 const evidence = steps.find(step => step.with?.path === 'build/s028-docs-ui');
 assert.equal(evidence?.if, 'always()');
 assert.ok(evidence?.with.name.endsWith('-${{ github.sha }}-${{ github.run_attempt }}'));
 assert.equal(evidence?.with['retention-days'], 14);
});

test('known site presentation skips native work but mixed, new and dependency inputs qualify fully', () => {
 for (const item of ['site/app/page.jsx', 'site/app/styles.css', 'scripts/check-live-site-ui.mjs']) assert.equal(classify([item]).native,false);
 for (const item of ['site/package-lock.json','site/next.config.mjs','site/app/new.jsx','scripts/website.py','schemas/v1.0.1/master.schema.json']) assert.equal(classify(['site/app/page.jsx',item]).native,true);
});

test('same-run export reuse never removes later qualification gates or release builds', () => {
 const ci=parse(readFileSync(new URL('../.github/workflows/ci.yml',import.meta.url),'utf8'));
 const native=parse(readFileSync(new URL('../.github/workflows/native-qualification.yml',import.meta.url),'utf8'));
 assert.equal(ci.jobs.scope.needs,undefined);assert.equal(ci.jobs.foundation.needs,'readiness');
 for(const label of ['linux','windows'])for(const lane of ['qualify','package']){const job=ci.jobs[lane+'-'+label];assert.equal(job.with['reuse-ci-offline-help'],true);if(lane==='package'&&label==='windows'){assert.ok(job.needs.includes('readiness'));assert.equal(job.with['overlap-source-setup'],true);}else{assert.ok(job.needs.includes('docs-build'));assert.ok(job.if.includes("needs.foundation.result == 'success'"));}}
 assert.equal(native.on.workflow_call.inputs['reuse-ci-offline-help'].default,false);
 for(const job of [native.jobs.runtime_checks,native.jobs.packages]){assert.ok(job.steps.some(s=>s.run?.includes('site-transfer.py restore')));assert.ok(job.steps.some(s=>s.run==='npm --prefix site run build'&&s.if.includes('!inputs.reuse-ci-offline-help')));}
 assert.ok(ci.jobs.candidate.needs.includes('foundation'));assert.ok(ci.jobs.candidate.needs.includes('docs'));
});

test('Windows package setup follows readiness and product work follows the explicit qualification gate', () => {
 const ci=parse(readFileSync(new URL('../.github/workflows/ci.yml',import.meta.url),'utf8'));
 const native=parse(readFileSync(new URL('../.github/workflows/native-qualification.yml',import.meta.url),'utf8'));
 assert.ok(ci.jobs['package-windows'].needs.includes('readiness'));
 assert.ok(ci.jobs.readiness.needs.includes('media-windows'));
 assert.equal(ci.jobs['package-windows'].with['overlap-source-setup'],true);
 assert.equal(native.on.workflow_call.inputs['overlap-source-setup'].default,false);
 const steps=native.jobs.packages.steps;
 const gate=steps.findIndex(s=>s.run==='node scripts/ci-acceptance.mjs wait-inputs');
 const wait=steps.findIndex(s=>s.run?.includes('wait-ci-artifact.mjs'));
 assert.ok(gate>=0&&wait>gate);
 const download=steps.findIndex(s=>s.with?.name==='media-source-${{ inputs.os }}-${{ github.run_attempt }}');
 const verify=steps.findIndex(s=>s.run?.includes('media-transfer.py restore'));
 assert.ok(wait>steps.findIndex(s=>s.uses==='./build/actions/setup-go')&&download>wait&&verify>download);
 assert.equal(steps[download].with['digest-mismatch'],'error');
 assert.ok(ci.jobs.candidate.needs.includes('media-windows'));
 assert.ok(ci.jobs.candidate.if.includes("needs.media-windows.result == 'success'"));
});


test('focused native source diagnosis keeps production compiler and decoder prerequisites',()=>{
 const workflow=parse(readFileSync(new URL('../.github/workflows/installer-invocation-diagnostic.yml',import.meta.url),'utf8'));
 const steps=workflow.jobs.invocation.steps;
 const decoder=steps.findIndex(step=>step.run?.includes('zstandard==0.25.0'));
 const prepare=steps.findIndex(step=>step.run==='python scripts/qualify.py prepare');
 assert.ok(decoder>=0 && decoder<prepare);
 assert.match(steps[decoder].run,/--no-input --only-binary=:all: --target build\/native\/python/);
 assert.equal(steps[decoder].if,"matrix.kind == 'desktop-startup' && inputs.startup_source");
 assert.ok(steps.findIndex(step=>step.uses?.startsWith('msys2/setup-msys2@'))<prepare);
 assert.ok(steps.some(step=>step.run==='npm --prefix desktop/frontend ci --ignore-scripts'));
 assert.equal(workflow.permissions.actions,'read');
 assert.equal(workflow.jobs.invocation['timeout-minutes'],65);
});

test('candidate tools and Linux/docs downloads overlap the explicit Windows package gate', () => {
 const workflow=parse(readFileSync(new URL('../.github/workflows/ci.yml',import.meta.url),'utf8'));
 const job=workflow.jobs.candidate, steps=job.steps;
 assert.ok(!job.needs.includes('package-windows'));
 for(const gate of ['foundation','docs','package-linux','media-windows'])assert.ok(job.needs.includes(gate));
 assert.equal(job.permissions.actions,'read');
 const wait=steps.findIndex(s=>s.run?.includes('wait-ci-artifact.mjs windows-packages'));
 const linux=steps.findIndex(s=>s.with?.pattern==='candidate-package-ubuntu-24.04-*-${{ github.sha }}-${{ github.run_attempt }}');
 const docs=steps.findIndex(s=>s.with?.name==='candidate-documentation-${{ github.sha }}-${{ github.run_attempt }}');
 const windows=steps.findIndex(s=>s.with?.pattern==='candidate-package-windows-2022-*-${{ github.sha }}-${{ github.run_attempt }}');
 const collect=steps.findIndex(s=>s.run?.includes('scripts/release.py collect'));
 assert.ok(wait>steps.findIndex(s=>s.uses==='./build/actions/setup-go'));
 assert.ok(linux>=0&&linux<wait&&docs>=0&&docs<wait);
 assert.ok(windows>wait&&collect>windows);
 assert.equal(steps[wait].env.GH_TOKEN,'${{ github.token }}');
 for(const index of [linux,docs,windows])assert.equal(steps[index].with['digest-mismatch'],'error');
});

test('downstream CI joins inherit the original qualification deadline before waiting for artifacts', () => {
 for(const [name,ids] of [['windows-desktop-qualification',['assemble','proofs','finalize']],['ci',['candidate']]]){
  const workflow=parse(readFileSync(new URL('../.github/workflows/'+name+'.yml',import.meta.url),'utf8'));
  for(const id of ids){
   const steps=workflow.jobs[id].steps;
   const gate=steps.findIndex(step=>step.run==='node scripts/ci-acceptance.mjs wait-inputs');
   const wait=steps.findIndex(step=>step.run?.includes('wait-ci-artifact.mjs'));
   assert.ok(gate>=0&&gate<wait,name+'/'+id+' must retain the original readiness boundary');
   assert.ok(steps[gate].env.EXPECTED_HEAD.includes('github.event.pull_request.head.sha'));
  }
 }
 const administrative=parse(readFileSync(new URL('../.github/workflows/issue-classification.yml',import.meta.url),'utf8'));
 assert.equal(administrative.jobs.classify['timeout-minutes'],2);
});
