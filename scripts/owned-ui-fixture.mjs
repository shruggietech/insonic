// SPDX-License-Identifier: Apache-2.0
// Disposable UI state is owned by allocation identity, never by a name alone.
import * as fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import {randomBytes} from 'node:crypto';

export function validateTemporaryParent(value, paths=path) {
  if(typeof value!=='string'||!value.trim()||!paths.isAbsolute(value))throw Error('UI temporary parent must be an absolute non-root directory.');
  const normalized=paths.normalize(value);
  let semantic=normalized;
  if(paths===path.win32){
    semantic=semantic.replace(/^\\\\\?\\UNC\\/i,'\\\\').replace(/^\\\\\?\\(?=[a-z]:\\)/i,'');
    if(!/^(?:[a-z]:\\|\\\\[^\\]+\\[^\\]+\\?)/i.test(semantic))throw Error('UI temporary parent must name an absolute drive or share directory.');
  }
  if(paths.normalize(semantic)===paths.parse(semantic).root)throw Error('UI temporary parent cannot be a volume or share root.');
  return normalized;
}

const same=(a,b)=>a.dev===b.dev&&a.ino===b.ino;
function residual(directory,error) {
  return Object.assign(Error('Owned UI fixture remains at '+directory+': '+(error.code?error.code+': ':'')+error.message,{cause:error}),{code:'fixture_cleanup_failed',residual:directory});
}

export async function createOwnedFixture(prefix,{parent=os.tmpdir(),markerName,kind,parentPolicy='system-temporary',validateParent=()=>{},initialize=async()=>{},filesystem=fs}={}) {
  if(!/^insonic-[a-z-]+-$/.test(prefix)||!/^[-a-z]+\.json$/.test(markerName||''))throw Error('Invalid internal UI fixture policy.');
  const selected=validateTemporaryParent(parent);
  const canonical=validateTemporaryParent(await filesystem.realpath(selected));
  await validateParent(canonical);
  const parentIdentity=await filesystem.lstat(canonical,{bigint:true});
  if(!parentIdentity.isDirectory()||parentIdentity.isSymbolicLink())throw Error('UI temporary parent is not a real directory.');
  const id=randomBytes(16).toString('hex');
  const directory=await filesystem.mkdtemp(path.join(canonical,prefix));
  let identity,markerCommitted=false,removed=false;
  const evidence={parent:canonical,path:directory,parent_policy:parentPolicy,non_root:true,removed:false};
  const verify=async()=>{
    const currentParent=await filesystem.lstat(canonical,{bigint:true});
    const current=await filesystem.lstat(directory,{bigint:true});
    if(!identity||!same(parentIdentity,currentParent)||!currentParent.isDirectory()||currentParent.isSymbolicLink()||await filesystem.realpath(canonical)!==canonical||path.dirname(directory)!==canonical||!path.basename(directory).startsWith(prefix)||!current.isDirectory()||current.isSymbolicLink()||!same(identity,current)||await filesystem.realpath(directory)!==directory)throw Error('Fixture ownership or confinement changed.');
    if(markerCommitted){
      const marker=JSON.parse(await filesystem.readFile(path.join(directory,markerName),'utf8'));
      if(marker.fixture_id!==id||marker.kind!==kind)throw Error('Fixture ownership marker changed.');
    }
  };
  const cleanup=async()=>{
    if(removed)return;
    try{
      await verify();
      await filesystem.rm(directory,{recursive:true,force:true,maxRetries:10,retryDelay:100});
      try{await filesystem.lstat(directory);throw Error('Owned fixture still exists after removal.')}catch(error){if(error.code!=='ENOENT')throw error}
      removed=true;evidence.removed=true;
    }catch(error){throw residual(directory,error)}
  };
  try{
    identity=await filesystem.lstat(directory,{bigint:true});
    await verify();
    await filesystem.writeFile(path.join(directory,markerName),JSON.stringify({fixture_id:id,kind})+'\n',{encoding:'utf8',flag:'wx'});
    markerCommitted=true;
    await initialize(directory,filesystem);
    return {directory,parent:canonical,id,evidence,cleanup};
  }catch(error){
    try{await cleanup()}catch(cleanupError){throw Object.assign(new AggregateError([error,cleanupError],error.message+'; '+cleanupError.message),{code:'fixture_cleanup_failed',residual:directory})}
    throw error;
  }
}
