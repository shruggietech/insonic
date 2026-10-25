// SPDX-License-Identifier: Apache-2.0
import {useEffect,useRef,useState} from 'react';
import {Button,Card,Input,Select,Facts,Actions} from './components';
import {Client,result,StaleResponse,type Obj} from './client';
import {uuid} from './forms';
import type {Run} from './screens';
import {workerElection,setupElection,defaultModelElection,workerProbeSummary} from './worker-forms';

function ProbeSummary({inspection}:{inspection:Obj}){const summary=workerProbeSummary(inspection);return summary?<section aria-label="Worker verification"><p><strong>{summary.role} verification:</strong> {summary.outcome} ({summary.stage}).</p><Facts value={summary}/></section>:null}
function bytes(n:number){return `${n.toLocaleString()} bytes (${(n/1048576).toFixed(1)} MiB)`}
export function Workers({client,run,onApplied}:{client:Client;run:Run;onApplied?:()=>Promise<void>}){
  const [inventory,setInventory]=useState<Obj>(),[defaults,setDefaults]=useState<Obj[]>([]),[plan,setPlan]=useState<Obj>(),[inspection,setInspection]=useState<Obj>(),[readiness,setReadiness]=useState<Obj>();
  const [role,setRole]=useState('recognition'),[mode,setMode]=useState('managed'),[python,setPython]=useState(''),[offline,setOffline]=useState('');
  const [choice,setChoice]=useState('whisper-tiny'),[local,setLocal]=useState(''),[destination,setDestination]=useState(''),[credential,setCredential]=useState(''),[token,setToken]=useState(''),[preview,setPreview]=useState<Obj>();
  const [work,setWork]=useState<Obj>(),[pipeline,setPipeline]=useState<Obj>(),[pipelineName,setPipelineName]=useState('Local CPU speech'),[busy,setBusy]=useState(false);
  const active=useRef<AbortController|undefined>(undefined);
  const load=async(signal?:AbortSignal)=>{setInventory(await client.call('workers.catalog','',undefined,signal));const entries=await client.call('models.defaults','',undefined,signal);setDefaults(Array.isArray(entries)?entries:[]);};
  useEffect(()=>{const controller=new AbortController();void run(()=>load(controller.signal));return()=>{controller.abort();active.current?.abort()}},[client,client.workspace]);
  useEffect(()=>setPlan(undefined),[role,mode,python,offline]);
  useEffect(()=>setPreview(undefined),[choice,local,destination,credential]);
  const action=(fn:(signal:AbortSignal)=>Promise<unknown>)=>run(async()=>{const controller=new AbortController();active.current=controller;setBusy(true);try{await fn(controller.signal)}finally{if(active.current===controller)active.current=undefined;setBusy(false)}});
  const selected=inventory?.catalog?.profiles?.find((p:Obj)=>p.id===role&&p.platform===inventory.platform);
  const chosen=defaults.find(c=>c.choice.id===choice);
  const refreshWork=async(signal:AbortSignal)=>{if(!work?.work_id)return;const current=await client.call('work.show',work.work_id,undefined,signal);setWork(current);if(current.state==='succeeded'){await load(signal);await onApplied?.()}};
  const acquire=async(signal:AbortSignal)=>{
    let reference=choice==='community-1'?credential:'';const generation=client.generation,workspace=client.workspace;
    if(token&&choice==='community-1'){reference=reference||uuid();try{result(await client.bridge.Credential([credential?'replace':'add',reference],JSON.stringify({value:token})));if(generation!==client.generation||workspace!==client.workspace)throw new StaleResponse();setCredential(reference)}finally{setToken('')}}
    const out=await client.call('models.prepare','',defaultModelElection(choice,reference,local,destination,true),signal);setPreview(out);setWork(out.work);await load(signal);return out;
  };
  return <>
    <Card heading="Local processing setup">
      <p>Prepare optional transcription or diarization libraries, or verify an existing environment. CPU is the managed default. Imports and supplied subtitles can be used independently.</p>
      <Select disabled={busy} label="Processing role" value={role} onChange={setRole} options={[{value:'recognition',label:'Transcription'},{value:'diarization',label:'Diarization and speaker profile enrollment'}]}/>
      <Select disabled={busy} label="Environment preparation" value={mode} onChange={setMode} options={[{value:'managed',label:'Create isolated managed CPU environment'},{value:'reuse',label:'Verify and reuse existing environment'}]}/>
      {mode==='reuse'?<>
        <Input disabled={busy} label="Existing Python interpreter" value={python} onChange={setPython} description="The selected environment is inspected and pinned without installing into it."/>
        {!!inventory?.candidates?.length&&<Select disabled={busy} label="Detected interpreter candidates (not yet verified)" value={python} onChange={setPython} options={[{value:'',label:'Select a candidate'},...inventory.candidates.map((p:string)=>({value:p,label:p}))]}/>}
      </>:<><p>{selected?`${offline?"Offline network transfer: 0 bytes. Nominal profile size:":"Download"} ${bytes(selected.download_bytes)}. Keep at least ${bytes(selected.min_space_bytes)} free for installation and temporary files.`:'Read the catalog for this host before preparing an environment.'}</p><Input disabled={busy} label="Offline asset directory (optional)" value={offline} onChange={setOffline} description="Provide the exact declared Python, uv and wheel filenames. Every file is verified; offline setup performs no network acquisition."/></>}
      <Actions><Button variant="secondary" disabled={busy} onClick={()=>action(async signal=>{const out=await client.call('workers.plan','',workerElection(role,mode,python,offline),signal);setPlan(out);return out})}>Preview environment setup</Button>
        <Button disabled={busy||!plan||!!plan.prerequisite||plan.space_sufficient===false} onClick={()=>action(async signal=>{const out=await client.call('workers.setup','',setupElection(plan!,workerElection(role,mode,python,offline)),signal);setWork(out);return out})}>Prepare verified environment</Button>
        <Button variant="secondary" disabled={busy||mode==='reuse'&&!python} onClick={()=>action(async signal=>{const out=await client.call('workers.inspect','',{role,...(mode==='reuse'?{python_path:python}:{})},signal);setInspection(out);return out})}>Inspect environment and devices</Button>
        {busy&&<Button variant="secondary" onClick={()=>active.current?.abort()}>Stop inspection request</Button>}</Actions>
      {plan&&<><Facts value={{destination:plan.destination,download_bytes:plan.download_bytes,profile_download_bytes:plan.profile_download_bytes,local_assets:plan.local_assets,required_space_bytes:plan.required_space_bytes,available_space_bytes:plan.available_space_bytes,prerequisite:plan.prerequisite}}/><details><summary>Exact dependency identities and setup election</summary><Facts value={plan}/></details></>}
      {inspection&&<><ProbeSummary inspection={inspection}/><details><summary>Environment identities and device details</summary><Facts value={inspection}/></details></>}
      <p>CUDA is an explicit pipeline selection. Device detection, compatible libraries and model execution are separate facts; a failed CUDA job keeps its original election.</p>
    </Card>
    <Card heading="Supported local model bundles">
      <Select disabled={busy} label="Exact model bundle" value={choice} onChange={setChoice} options={[{value:'whisper-tiny',label:'Whisper Tiny transcription'},{value:'community-1',label:'Community-1 diarization and speaker profiles'}]}/>
      {chosen&&<><p>{chosen.choice.manifest.license}; required model files: {bytes(chosen.total_bytes)}.</p><p><a href={chosen.choice.model_card} target="_blank" rel="noreferrer">Upstream model card and access terms</a></p><p>{chosen.choice.access}</p></>}
      <Input disabled={busy} label="Existing verified model directory (optional)" value={local} onChange={setLocal} description="Reuse exact local snapshot files through normal artifact storage; original files remain unchanged."/>
      <Input disabled={busy} label="Storage profile ID (optional)" value={destination} onChange={setDestination} description="Omit to elect this workspace's configured filesystem or S3 storage. A shared acquisition preserves its original destination."/>
      {choice==='community-1'&&<><Input disabled={busy} label="Existing protected Hugging Face credential reference (optional)" value={credential} onChange={setCredential}/><Input disabled={busy} label="New Hugging Face read token (optional)" type="password" value={token} onChange={setToken} description="A new token is registered through protected native input with an automatically created reference. Values do not enter model requests or work records. Local reuse needs no download token."/></>}
      <Actions><Button variant="secondary" disabled={busy} onClick={()=>action(async signal=>{const out=await client.call('models.prepare','',defaultModelElection(choice,choice==='community-1'?credential:'',local,destination),signal);setPreview(out);return out})}>Preview model acquisition</Button><Button disabled={busy||!preview||preview.scratch_space_sufficient===false||preview.storage_space_sufficient===false||preview.access_ready===false&&!token} onClick={()=>action(acquire)}>Acquire or reuse exact model</Button></Actions>
      {preview&&<Facts value={{model_id:preview.model_id,total_bytes:preview.total_bytes,download_bytes:preview.download_bytes,verified_reuse_bytes:preview.verified_reuse_bytes,state:preview.state,destination_frozen:preview.destination_frozen,destination:preview.destination,scratch_required_bytes:preview.scratch_required_bytes,scratch_available_bytes:preview.scratch_available_bytes,storage_capacity:preview.storage_capacity,capacity_relation:preview.capacity_relation,storage_required_bytes:preview.storage_required_bytes,access_ready:preview.access_ready,credential_state:preview.credential_state,upstream_authorization:preview.upstream_authorization,access_correction:preview.access_correction,acquisition_work_id:preview.acquisition_work_id}}/>}
    </Card>
    {work&&<Card heading="Processing setup work">{work.result?.worker_probe&&<ProbeSummary inspection={work.result.worker_probe}/>}<Facts value={work}/><Actions><Button variant="secondary" disabled={busy} onClick={()=>action(refreshWork)}>Refresh setup work</Button><Button variant="secondary" disabled={busy||!['pending','running','interrupted'].includes(work.state)} onClick={()=>action(async signal=>{const out=await client.call('work.cancel',work.work_id,undefined,signal);setWork(out);return out})}>Cancel setup work</Button><Button variant="secondary" disabled={busy||!['failed','cancelled','interrupted'].includes(work.state)} onClick={()=>action(async signal=>{const out=await client.call('work.retry',work.work_id,undefined,signal);setWork(out);return out})}>Retry frozen setup work</Button></Actions><p>Closing or stopping an inspection request does not cancel durable setup or model acquisition. Use Cancel setup work, or inspect it in Jobs.</p></Card>}
    <Card heading="Task readiness and local pipeline">
      <Actions><Button variant="secondary" disabled={busy} onClick={()=>action(async signal=>{const out=await client.call('processing.readiness','',pipeline?{pipeline_id:pipeline.id,pipeline_revision:pipeline.revision}:{},signal);setReadiness(out);return out})}>Inspect task readiness</Button></Actions>
      {readiness&&<>{["recognition","diarization"].map(role=>readiness.environments?.[role]?<ProbeSummary key={role} inspection={readiness.environments[role]}/>:null)}<Facts value={readiness.tasks}/><details><summary>Observed environment and verification details</summary><Facts value={readiness}/></details></>}
      <Input disabled={busy} label="Local pipeline name" value={pipelineName} onChange={setPipelineName}/>
      <Button disabled={busy||!!pipeline||!pipelineName.trim()||defaults.length<2} onClick={()=>action(async signal=>{const id=uuid();const rec=defaults.find(x=>x.choice.id==='whisper-tiny')!,dia=defaults.find(x=>x.choice.id==='community-1')!;const out=await client.call('pipelines.set',id,{expected_revision:0,pipeline:{id,name:pipelineName.trim(),preset:'local',configuration:{recognition:{adapter:'faster-whisper',contract_version:'1',mode:'local',model_id:rec.model_id,recognition:{device:'cpu'}},diarization:{adapter:'pyannote',contract_version:'1',mode:'local',model_id:dia.model_id,diarization:{device:'cpu'}}}}},signal);setPipeline(out);return out})}>Save local CPU pipeline</Button>
      {pipeline&&<Facts value={pipeline}/>}<p>Saving a pipeline does not acquire its models or run inference. Complete elected acquisitions and inspect readiness, then select the saved pipeline in Library or an import preset.</p>
    </Card>
  </>;
}
