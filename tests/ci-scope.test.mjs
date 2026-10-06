import test from 'node:test';
import assert from 'node:assert/strict';
import { classify } from '../scripts/ci-scope.mjs';

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
