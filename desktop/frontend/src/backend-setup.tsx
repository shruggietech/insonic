// SPDX-License-Identifier: Apache-2.0
import {useEffect,useRef,useState} from 'react';
import {Actions,Button,Card,Check,Facts,Input,Select} from './components';
import {result,type NativeBridge,type Obj} from './client';
import {uuid} from './forms';
import type {Run} from './screens';
const defaults:Obj={storage:{adapter_id:'filesystem',configuration:{root:'artifacts'}},catalog:{adapter_id:'sqlite',configuration:{path:'catalog.sqlite'}},graph:{adapter_id:'ladybugdb',configuration:{path:'graph/library.lbug'}}};
const options:Obj={filesystem:{root:'artifacts'},s3:{endpoint:'',bucket:'',prefix:'',region:'us-east-1',addressing_style:'path',authentication:'credential'},sqlite:{path:'catalog.sqlite'},postgresql:{host:'localhost',port:5432,database:'insonic',schema:'insonic',tls_mode:'verify-full'},ladybugdb:{path:'graph/library.lbug'},arcadedb:{endpoint:'',database:'insonic',preferred_dialect:'arcade-opencypher'}};
const fields:Obj={filesystem:['root'],s3:['endpoint','bucket','prefix','region','addressing_style','authentication'],sqlite:['path'],postgresql:['host','port','database','schema','tls_mode','ca_file'],ladybugdb:['path'],arcadedb:['endpoint','database','preferred_dialect']};
const enums:Obj={addressing_style:['auto','path','virtual'],authentication:['credential','environment','anonymous'],tls_mode:['verify-full','local','insecure'],preferred_dialect:['arcade-opencypher','arcade-sql']};
export function BackendSetup({bridge,run,path,name,onConfigured}:{bridge:NativeBridge;run:Run;path?:string;name?:string;onConfigured:(value:Obj)=>void}){
 const [roles,setRoles]=useState<Obj>(()=>structuredClone(defaults)),[location,setLocation]=useState(''),[label,setLabel]=useState('My voice archive'),[mode,setMode]=useState('native'),[pass,setPass]=useState(''),[secrets,setSecrets]=useState<Obj>({}),[credential,setCredential]=useState<Obj>({}),[plan,setPlan]=useState<Obj>(),[report,setReport]=useState<Obj>(),[current,setCurrent]=useState<Obj>();
 const active=useRef(''),alive=useRef(true),ownership=useRef(0),workspaceID=useRef(uuid());
 const cancel=()=>{ownership.current++;if(active.current)void bridge.CancelOperation?.(active.current);active.current=''};
 useEffect(()=>()=>{alive.current=false;cancel()},[]);
 const invalidate=()=>{cancel();setPlan(undefined);setReport(undefined)};
 const update=(role:string,key:string,value:unknown)=>{invalidate();setRoles(p=>({...p,[role]:{...p[role],configuration:{...p[role].configuration,[key]:value}}}))};
 useEffect(()=>{invalidate();workspaceID.current=uuid()},[path]);
 const input=()=>{const profiles=structuredClone(roles),session:Obj={};for(const role of Object.keys(profiles)){const cfg=profiles[role].configuration,adapter=profiles[role].adapter_id,has=cfg.authentication==='credential'||credential[role];if(has){if(!cfg.credential_id)cfg.credential_id=uuid();if(secrets[role])session[cfg.credential_id]=secrets[role]}else delete cfg.credential_id;if(!cfg.ca_file)delete cfg.ca_file;if(!profiles[role].expected_backend_version)delete profiles[role].expected_backend_version;}return {draft:{schema_version:'1.0.1',kind:'backend-plan',workspace_id:workspaceID.current,display_name:name??label,profiles},session}};
 const invoke=async(action:string,configuration:Obj,protectedInput:Obj={})=>{if(!bridge.BackendSetup)throw new Error('Use a matching desktop package with backend setup support.');const id=uuid();active.current=id;try{return result(await bridge.BackendSetup(action,path??location,JSON.stringify(configuration),JSON.stringify(protectedInput),mode,id))}finally{if(active.current===id)active.current=''}};
 const perform=(action:string)=>run(async()=>{const token=++ownership.current;try{const {draft,session}=input();let selected=plan;if(!selected){const out=await invoke('plan',draft);selected=out.configuration;if(!alive.current||token!==ownership.current)return;setPlan(selected);setRoles(p=>{const next=structuredClone(p);for(const role of Object.keys(next)){const id=selected!.profiles[role].configuration.credential_id;if(id)next[role].configuration.credential_id=id}return next})}if(action==='plan')return;const out=await invoke(action,selected!,{session,...(mode==='vault'?{passphrase:pass}:{})});if(!alive.current||token!==ownership.current)return;setReport(out);if(action==='provision')onConfigured(out)}finally{setSecrets({});setPass('')}});
 return <Card heading="Backend configuration">
  <p>Select three independent roles. Filesystem, SQLite and LadybugDB are local defaults. Preflight does not create databases or upload media. Provision creates a fresh workspace; populated backend changes use explicit migration/backup.</p>
  {path===undefined&&<Input label="Backend workspace path" value={location} onChange={v=>{invalidate();workspaceID.current=uuid();setLocation(v)}}/>}
  {name===undefined&&<Input label="Backend workspace name" value={label} onChange={v=>{invalidate();setLabel(v)}}/>}
  <Button variant="secondary" onClick={()=>run(async()=>{const out=await invoke('inspect',{});if(alive.current)setCurrent(out)})}>Inspect current backend selections</Button>
  {current&&<Facts value={current}/>}
  {Object.entries(roles).map(([role,p])=><fieldset key={role}><legend>{role==='storage'?'Artifact storage':role==='catalog'?'Operational catalog':'Graph projection'}</legend>
   <Select label={role+' backend'} value={p.adapter_id} options={role==='storage'?['filesystem','s3']:role==='catalog'?['sqlite','postgresql']:['ladybugdb','arcadedb']} onChange={v=>{invalidate();setRoles(x=>({...x,[role]:{adapter_id:v,configuration:structuredClone(options[v])}}));setCredential(x=>({...x,[role]:false}));setSecrets(x=>({...x,[role]:undefined}))}}/>
   {fields[p.adapter_id].map((key:string)=>enums[key]?<Select key={key} label={role+' '+key.replaceAll('_',' ')} value={p.configuration[key]??enums[key][0]} options={enums[key]} onChange={v=>update(role,key,v)}/>:<Input key={key} label={role+' '+key.replaceAll('_',' ')} type={key==='port'?'number':'text'} value={String(p.configuration[key]??'')} onChange={v=>update(role,key,key==='port'?Number(v):v)}/>)}
   <Input label={role+' expected backend version'} value={p.expected_backend_version??''} onChange={v=>{invalidate();setRoles(x=>({...x,[role]:{...x[role],expected_backend_version:v}}))}}/>
   {['postgresql','arcadedb'].includes(p.adapter_id)&&<Check label={role+' uses a protected credential'} value={Boolean(credential[role])} onChange={v=>{invalidate();setCredential(x=>({...x,[role]:v}))}}/>}
   {(p.configuration.authentication==='credential'||credential[role])&&<>
    <p>Enter new values for this role. An opaque reference is generated automatically; values are sent through the protected native channel and cleared after each attempt.</p>
    <Input label={role+' credential reference (optional existing ID)'} value={p.configuration.credential_id??''} onChange={v=>update(role,'credential_id',v)}/>
    {(p.adapter_id==='s3'?['access_key','secret_key','session_token']:['username','password']).map(key=><Input key={key} label={role+' credential '+key.replaceAll('_',' ')} type={key==='username'?'text':'password'} value={secrets[role]?.[key]??''} onChange={v=>{cancel();setReport(undefined);setSecrets(x=>({...x,[role]:{...x[role],[key]:v}}))}}/>)}</>}
   {p.adapter_id==='sqlite'&&<p>SQLite WAL requires same-host storage and shared memory. Remote media can use S3; PostgreSQL supports a connected catalog.</p>}
   {p.adapter_id==='postgresql'&&<p>verify-full checks certificate and hostname. local is a loopback address or Unix socket. insecure explicitly elects remote plaintext.</p>}
  </fieldset>)}
  <Select label="Bootstrap credential storage" value={mode} options={['native','vault','session']} onChange={v=>{invalidate();setMode(v);setPass('')}}/>
  {mode==='vault'&&<Input label="Bootstrap vault passphrase" type="password" value={pass} onChange={v=>{cancel();setReport(undefined);setPass(v)}}/>}
  <p>Native/vault registration is separate from successful provider authentication. Session values last in the runtime owner; supply them again after restart.</p>
  <Actions><Button onClick={()=>perform('plan')}>Prepare backend plan</Button><Button onClick={()=>perform('preflight')}>Preflight selected backends</Button><Button onClick={()=>perform('provision')}>Create configured workspace</Button><Button variant="secondary" onClick={cancel}>Cancel backend operation</Button></Actions>
  {plan&&<details><summary>Prepared nonsecret backend plan</summary><Facts value={plan}/></details>}{report&&<><h3>Backend readiness</h3><Facts value={report}/></>}
 </Card>
}
