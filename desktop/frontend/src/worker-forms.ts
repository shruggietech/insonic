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
