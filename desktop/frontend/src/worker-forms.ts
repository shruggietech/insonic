// SPDX-License-Identifier: Apache-2.0
import type {Obj} from './client';
export const workerDevices = ['cpu','cuda'];
export function workerElection(role:string,mode:string,python:string,offline:string):Obj {
  if(!['recognition','diarization'].includes(role)||!['managed','reuse'].includes(mode))throw new Error('Choose a supported environment role and preparation mode.');
  if(mode==='reuse'&&!python.trim())throw new Error('Select the existing interpreter to verify.');
  return {role,mode,...(mode==='reuse'?{python_path:python.trim()}:{...(offline.trim()?{offline_assets:offline.trim()}:{})})};
}
export function setupElection(plan:Obj,current:Obj):Obj {
  const election=plan?.election;
  if(!election?.expected_revision||['role','mode','python_path','offline_assets'].some(k=>(election[k]??'')!==(current[k]??'')))throw new Error('Preview the current selections again before preparing the environment.');
  return {...election};
}
export function defaultModelElection(choice:string,credential:string,local:string,destination:string,acquire=false):Obj {
  return {choice,...(credential?{credential_id:credential}:{}),...(local.trim()?{local_directory:local.trim()}:{}),...(destination.trim()?{destination_profile_id:destination.trim()}:{}),...(acquire?{acquire:true}:{})};
}

const probeStages:Record<string,string>={interpreter_hash:'Interpreter verification',private_home:'Private verification directory',process_start:'Interpreter launch',package_versions:'Package versions',library_import:'Library import',cpu_prerequisites:'CPU prerequisites',cuda_prerequisites:'CUDA prerequisites',dependency_inventory:'Dependency inventory',dependency_hashing:'Dependency verification',output_validation:'Result validation',receipt_verification:'Environment receipt verification',complete:'Complete',not_started:'Before verification'};
const probeReasons:Record<string,string>={success:'Verification completed',interpreter_missing:'Interpreter not selected',interpreter_unreadable:'Interpreter could not be verified',home_unavailable:'Private verification directory unavailable',launch_failed:'Interpreter launch failed',process_failed:'Verification process failed',deadline:'Verification deadline reached',cancelled:'Verification cancelled',output_invalid:'Invalid verification result',output_limit:'Verification result exceeded its limit',package_missing:'Required package missing',version_mismatch:'Incompatible package version',import_failed:'Library import failed',cpu_unavailable:'CPU prerequisite unavailable',inventory_limit:'Dependency inventory exceeded its limit',inventory_unreadable:'Dependency inventory could not be verified',checkpoint_invalid:'Invalid verification progress'};
export function workerProbeSummary(inspection:Obj):Obj|undefined{
 const p=inspection?.probe;
 const bounded=(x:unknown,max:number)=>typeof x==='number'&&Number.isSafeInteger(x)&&x>=0&&x<=max;
 if(!p||!['inspect','snapshot'].includes(p.mode)||!Object.hasOwn(probeStages,p.stage)||!Object.hasOwn(probeReasons,p.reason)||!bounded(p.elapsed_ms,3600000)||!bounded(p.budget_ms,60000))return;
 const out:Obj={role:['recognition','diarization'].includes(inspection.role)?inspection.role:'environment',outcome:probeReasons[p.reason],stage:probeStages[p.stage],native_elapsed_ms:p.elapsed_ms,original_budget_ms:p.budget_ms};
 const c=p.child;
 if(c&&bounded(c.elapsed_ms,3600000)&&bounded(c.files,30000)&&bounded(c.hashed_files,30000)&&c.hashed_files<=c.files&&bounded(c.hashed_bytes,9007199254740991))Object.assign(out,{python_elapsed_ms:c.elapsed_ms,inventory_files:c.files,verified_files:c.hashed_files,verified_bytes:c.hashed_bytes});
 return out;
}
