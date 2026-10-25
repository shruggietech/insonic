// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import {validateNativeRequest} from './contracts.mjs';
import {workerElection,setupElection,defaultModelElection,workerDevices,workerProbeSummary} from '../.test-build/worker-forms.js';
const wid='aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa';
function valid(operation,data){return validateNativeRequest({operation,workspace_id:wid,request_id:wid,...(data?{data}:{})})}
test('managed/reused worker forms produce precise shared requests with automatic revision pins',()=>{
  const current=workerElection('recognition','managed','','');
  assert.equal(valid('workers.plan',current).valid,true);
  assert.equal(valid('workers.setup',current).valid,false);
  const elected=setupElection({election:{...current,expected_revision:'a'.repeat(64),profile_digest:'b'.repeat(64)}},current);
  assert.equal(valid('workers.setup',elected).valid,true);
  assert.throws(()=>setupElection({election:elected},workerElection('diarization','managed','','')),/Preview/);
  assert.equal(valid('workers.plan',workerElection('diarization','reuse','C:/owned/python.exe','')).valid,true);
  assert.throws(()=>workerElection('recognition','reuse','',''),/interpreter/);
});
test('default model requests contain protected references without credential values',()=>{
  const input=defaultModelElection('community-1',wid,'C:/owned/models','',true);
  assert.equal(valid('models.prepare',input).valid,true);
  assert.equal(JSON.stringify(input).includes('value'),false);
  assert.equal(JSON.stringify(input).includes('token'),false);
  assert.equal(valid('models.prepare',{...input,token:'must-not-cross'}).valid,false);
  assert.equal(valid('processing.readiness',{recognition_device:'cuda',diarization_device:'cpu'}).valid,true);
});
test('offered devices match executable worker elections, without silent auto fallback',()=>{
  assert.deepEqual(workerDevices,['cpu','cuda']);
  assert.equal(valid('processing.readiness',{recognition_device:'auto'}).valid,false);
});

test("worker probe summaries distinguish budget expiry and keep native and child clocks separate",()=>{const summary=workerProbeSummary({role:"diarization",probe:{mode:"snapshot",stage:"dependency_hashing",reason:"deadline",elapsed_ms:60000,budget_ms:60000,child:{elapsed_ms:51000,files:27571,hashed_files:12000,hashed_bytes:42000}}});assert.equal(summary.outcome,"Verification deadline reached");assert.equal(summary.stage,"Dependency verification");assert.equal(summary.native_elapsed_ms,60000);assert.equal(summary.python_elapsed_ms,51000);assert.equal(summary.verified_files,12000);assert.equal(workerProbeSummary({}),undefined);assert.equal(workerProbeSummary({probe:{mode:"inspect",stage:"secret-token",reason:"deadline",elapsed_ms:1,budget_ms:1}}),undefined);});
