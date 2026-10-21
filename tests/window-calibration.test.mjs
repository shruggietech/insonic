// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import YAML from 'yaml';
import { validateYAML } from '../scripts/check.mjs';

test('window comparison restores one explicit immutable assembly and cannot run historical completion diagnosis', () => {
 const flow=YAML.parse(readFileSync(new URL('../.github/workflows/installer-invocation-diagnostic.yml',import.meta.url),'utf8'));
 const job=flow.jobs.invocation;
 const download=job.steps.find(step=>step.name==='Restore exact instrumented Windows assembly for calibration');
 assert.equal(download.if,"matrix.kind == 'window-calibration'");
 assert.equal(download.with['digest-mismatch'],'error');
 assert.equal(download.with['run-id'],'${{ inputs.calibration_run }}');
 assert.equal(download.with.name,'windows-desktop-input-${{ inputs.calibration_revision }}-${{ inputs.calibration_attempt }}');
 const source=job.steps.find(step=>step.name==='Restore exact calibration source');
 assert.equal(source.with.path,'build/calibration-source');
 assert.equal(source.with.ref,'${{ inputs.calibration_revision }}');
 const compare=job.steps.find(step=>step.run==='python scripts/diagnose-window-calibration.py');
 assert.equal(compare.env.SOURCE_REVISION,'${{ inputs.calibration_revision }}');
 assert.equal(compare.if,"matrix.kind == 'window-calibration'");
 const historical=job.steps.find(step=>step.name==='Trace one documentation completion invocation');
 assert.ok(historical.if.includes("matrix.kind != 'window-calibration'"));
 const upload=job.steps.find(step=>step.if==='always()');
 assert.ok(upload.with.path.includes('build/window-calibration/evidence'));
 assert.equal(upload.with.archive,true);
});

test('selected calibration attempt cannot weaken ordinary CI attempt binding', () => {
 const text=readFileSync(new URL('../.github/workflows/installer-invocation-diagnostic.yml',import.meta.url),'utf8');
 assert.deepEqual(validateYAML('.github/workflows/installer-invocation-diagnostic.yml',text),[]);
 assert.ok(validateYAML('.github/workflows/ci.yml',text).some(value=>value.includes('github.run_attempt')));
 const changed=text.replace('run-id: ${{ inputs.calibration_run }}','run-id: ${{ github.run_id }}');
 assert.ok(validateYAML('.github/workflows/installer-invocation-diagnostic.yml',changed).some(value=>value.includes('github.run_attempt')));
});
