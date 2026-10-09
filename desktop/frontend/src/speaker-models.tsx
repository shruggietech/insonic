// SPDX-License-Identifier: Apache-2.0
import { useEffect, useRef, useState } from 'react';
import { Actions, Area, Button, Card, Facts, Input, Select, Table } from './components';
import { Client, type Obj } from './client';

type Props = {client:Client;run:(action:()=>Promise<void>)=>void};
const enrollmentAdapter={id:'pyannote-profile',contract_version:'1',mode:'local',architecture:'pyannote',output_kinds:['voice-embedding'],consumers:['voice-matching'],license:'configured-base-model'};
const json=(value:string)=>JSON.parse(value);

export function SpeakerTraining({client,run,speakerID}:{speakerID:string}&Props) {
 const [datasets,setDatasets]=useState<Obj[]>([]),[datasetNext,setDatasetNext]=useState(''),[datasetID,setDatasetID]=useState(''),[summary,setSummary]=useState<Obj>(),[work,setWork]=useState<Obj>();
 const [pipelines,setPipelines]=useState<Obj[]>([]),[pipelineID,setPipelineID]=useState(''),[pipelineNext,setPipelineNext]=useState('');
 const [recipe,setRecipe]=useState('{}'),[name,setName]=useState('Speaker profile'),[kind,setKind]=useState('voice-embedding'),[adapter,setAdapter]=useState(JSON.stringify(enrollmentAdapter,null,2)),[base,setBase]=useState(''),[parameters,setParameters]=useState('{}'),[checkpoint,setCheckpoint]=useState('');
 const selection=useRef(0);
 const edit=(setter:(value:string)=>void)=>(value:string)=>{selection.current++;setter(value);setWork(undefined)};
 const load=async()=>{const token=++selection.current;const [result,saved]=await Promise.all([client.call('models.dataset.list','',{speaker_id:speakerID}),client.call('pipelines.list','',{limit:100})]);if(token===selection.current){setDatasets(result.items??[]);setDatasetNext(result.next_id??'');setPipelines(saved.items??[]);setPipelineNext(saved.next_id??'')}};
 useEffect(()=>{selection.current++;setDatasetID('');setSummary(undefined);setWork(undefined);setDatasets([]);setDatasetNext('');setPipelines([]);setPipelineID('');setPipelineNext('');if(speakerID)run(load);return()=>{selection.current++}},[speakerID,client.workspace]);
 return <Card heading="Speaker datasets and elected training">
  <Area label="Corpus selection recipe" value={recipe} onChange={edit(setRecipe)} description="Optional recording filter, minimum/maximum duration and overlap exclusion. Preparation resolves the complete current independent corpus."/>
  <Actions><Button onClick={()=>run(async()=>{const token=++selection.current;const value=await client.call('models.dataset.create','',{speaker_id:speakerID,recipe:json(recipe)});if(token===selection.current){setSummary(value);setDatasetID(value.dataset?.id??value.id??'');await load();}})}>Prepare current speaker dataset</Button><Button variant="secondary" onClick={()=>run(load)}>Refresh datasets</Button></Actions>
  <Select label="Training dataset" value={datasetID} onChange={value=>{selection.current++;setDatasetID(value);setSummary(undefined);setWork(undefined)}} options={[{value:'',label:'Choose a current dataset'},...datasets.map(d=>({value:d.dataset?.id??d.id,label:`${d.dataset?.id??d.id} (${d.dataset?.state??d.state})`}))]}/>
  {datasetNext&&<Button variant="secondary" onClick={()=>run(async()=>{const token=selection.current;const page=await client.call('models.dataset.list','',{speaker_id:speakerID,limit:100,after_id:datasetNext});if(token===selection.current){setDatasets(old=>[...old,...page.items]);setDatasetNext(page.next_id??'')}})}>More speaker datasets</Button>}
  {datasetID&&<Button variant="secondary" onClick={()=>run(async()=>{const token=selection.current;const value=await client.call('models.dataset.show',datasetID);if(token===selection.current)setSummary(value);})}>Inspect dataset</Button>}
  {summary&&<Facts value={summary}/>}
  <Input label="Speaker model name" value={name} onChange={edit(setName)}/>
  <Select label="Saved training profile" value={pipelineID} onChange={value=>{selection.current++;setPipelineID(value);setWork(undefined)}} options={[{value:'',label:'Use inline training configuration'},...pipelines.filter(p=>p.configuration?.speaker_training).map(p=>({value:p.id,label:`${p.name} (revision ${p.revision})`}))]}/>
  {pipelineNext&&<Button variant="secondary" onClick={()=>run(async()=>{const token=selection.current;const page=await client.call('pipelines.list','',{limit:100,after_id:pipelineNext});if(token===selection.current){setPipelines(old=>[...old,...page.items]);setPipelineNext(page.next_id??'')}})}>More training profiles</Button>}
  {!pipelineID&&<><Input label="Output kind" value={kind} onChange={edit(setKind)}/>
  <Area label="Selected training adapter" value={adapter} onChange={edit(setAdapter)} description="Choose offline profile enrollment or your configured local/hosted training adapter. Hosted routes upload selected preparation; credential references belong here, credential values do not."/>
  <Input label="Training base model reference" value={base} onChange={edit(setBase)}/><Area label="Training parameters" value={parameters} onChange={edit(setParameters)}/></>}<Input label="Compatible checkpoint ID" value={checkpoint} onChange={edit(setCheckpoint)}/>
  <Button disabled={!datasetID} onClick={()=>run(async()=>{const token=++selection.current;const configured=pipelineID?{pipeline_id:pipelineID,pipeline_revision:pipelines.find(p=>p.id===pipelineID)?.revision}:{output_kind:kind,adapter:json(adapter),parameters:json(parameters),...(base?{base_model_id:base}:{})};const value=await client.call('models.train','',{dataset_id:datasetID,name,...configured,...(checkpoint?{checkpoint_id:checkpoint}:{})});if(token===selection.current)setWork(value);})}>Start elected training</Button>
  {work&&<><p>Training uses durable Jobs controls for cancellation, retry and recovery.</p><Facts value={work}/></>}
 </Card>;
}

export function SpeakerModels({client,run}:Props) {
 const [speaker,setSpeaker]=useState(''),[items,setItems]=useState<Obj[]>([]),[next,setNext]=useState(''),[selected,setSelected]=useState(''),[detail,setDetail]=useState<Obj>(),[destination,setDestination]=useState(''),[profileRevision,setProfileRevision]=useState('0'),[result,setResult]=useState<Obj>();
 const selection=useRef(0);
 const list=async(more=false)=>{const token=++selection.current;const value=await client.call('models.speaker.list','',{...(speaker?{speaker_id:speaker}:{}),limit:100,...(more&&next?{after_id:next}:{})});if(token===selection.current){setItems(old=>more?[...old,...value.items]:value.items);setNext(value.next_id??'');if(value.profile)setProfileRevision(String(value.profile.revision));}};
 useEffect(()=>{selection.current++;run(()=>list());return()=>{selection.current++}},[client.workspace]);
 const choose=async(id:string)=>{const token=++selection.current;setSelected(id);setDetail(undefined);setResult(undefined);const value=await client.call('models.speaker.show',id);if(token===selection.current){setDetail(value);if(value.profile&&(!speaker||speaker===value.profile.id)){setProfileRevision(String(value.profile.revision));setSpeaker(value.profile.id);}}};
 return <><Card heading="Speaker model versions">
  <Input label="Speaker model identity filter" value={speaker} onChange={value=>{selection.current++;setSpeaker(value);setItems([]);setNext('');setSelected('');setDetail(undefined)}}/>
  <Actions><Button onClick={()=>run(()=>list())}>Find speaker model versions</Button>{next&&<Button variant="secondary" onClick={()=>run(()=>list(true))}>More speaker model versions</Button>}</Actions>
  <Table caption="Immutable speaker outputs" heads={['Name','Version','Kind','Speaker','Action']}>{items.map(item=><tr key={item.id}><td>{item.name}</td><td>{item.id}</td><td>{item.kind}</td><td>{item.speaker_id}</td><td><Button variant="secondary" onClick={()=>run(()=>choose(item.id))}>Inspect version</Button></td></tr>)}</Table>
  <Input label="Exact speaker model version" value={selected} onChange={value=>{selection.current++;setSelected(value);setDetail(undefined);setResult(undefined)}}/><Button variant="secondary" disabled={!selected} onClick={()=>run(()=>choose(selected))}>Inspect exact version</Button>
  {detail&&<Facts value={detail}/>}<Input label="Model fetch destination" value={destination} onChange={value=>{selection.current++;setDestination(value);setResult(undefined)}}/><Button disabled={!selected||!destination} onClick={()=>run(async()=>{const token=++selection.current;const value=await client.call('models.speaker.fetch',selected,{destination});if(token===selection.current)setResult(value);})}>Fetch exact speaker version</Button>
 </Card><Card heading="Current speaker profile">
  <p>Select an exact compatible output for roster matching. Clearing a profile leaves its immutable output available.</p><Input label="Profile speaker identity" value={speaker} onChange={value=>{selection.current++;setSpeaker(value);setResult(undefined)}}/><Input label="Expected profile revision" value={profileRevision} onChange={value=>{selection.current++;setProfileRevision(value);setResult(undefined)}}/>
  <Actions>{['set','clear'].map(action=><Button key={action} disabled={!speaker||(action==='set'&&!selected)} onClick={()=>run(async()=>{const token=++selection.current;const value=await client.call('models.profile.set',speaker,{expected_revision:Number(profileRevision),version_id:action==='set'?selected:''});if(token===selection.current){setResult(value);setProfileRevision(String(value.revision));}})}>{action==='set'?'Use exact version as speaker profile':'Clear current speaker profile'}</Button>)}</Actions>{result&&<Facts value={result}/>}
 </Card></>;
}

export function RosterMatching({client,run,recordingID,roster}:{recordingID:string;roster?:Obj}&Props) {
 const [model,setModel]=useState(''),[adapter,setAdapter]=useState(JSON.stringify(enrollmentAdapter,null,2)),[threshold,setThreshold]=useState('0.7'),[margin,setMargin]=useState('0.1'),[minimum,setMinimum]=useState('1000000'),[work,setWork]=useState<Obj>();
 const selection=useRef(0);
 const edit=(setter:(value:string)=>void)=>(value:string)=>{selection.current++;setter(value);setWork(undefined)};
 useEffect(()=>{selection.current++;setWork(undefined);return()=>{selection.current++}},[recordingID,roster?.revision,client.workspace]);
 return <Card heading="Match current recording voices">
  <p>Only declared roster members with compatible profiles are candidates. Unknown or ambiguous voices remain unassigned; explicit corrections take precedence.</p>
  <Input label="Matching model reference" value={model} onChange={edit(setModel)}/><Area label="Selected matching adapter" value={adapter} onChange={edit(setAdapter)}/>
  <Actions><Input label="Matching score threshold" value={threshold} onChange={edit(setThreshold)}/><Input label="Ambiguity margin" value={margin} onChange={edit(setMargin)}/><Input label="Minimum voice evidence in microseconds" value={minimum} onChange={edit(setMinimum)}/></Actions>
  <Button disabled={!model} onClick={()=>run(async()=>{const token=++selection.current;setWork(undefined);const value=await client.call('recordings.match',recordingID,{model_id:model,adapter:json(adapter),threshold:Number(threshold),ambiguity_margin:Number(margin),min_evidence_us:Number(minimum)});if(token===selection.current)setWork(value);})}>Match voices against current roster</Button>
  {work&&<><Facts value={{id:work.work_id??work.id,state:work.state,phase:work.phase,error:work.error,diagnostics:work.result?.diagnostics,missing_profiles:work.result?.missing_profiles}}/>
   <Table caption="Current voice matching outcomes" heads={['Recording voice','Outcome','Known speaker','Score','Evidence duration']}>
    {(work.result?.decisions??[]).map((decision:Obj)=><tr key={decision.local_speaker_id}><td>{decision.local_speaker_id}</td><td>{decision.state}</td><td>{decision.speaker_id??'Unassigned'}</td><td>{decision.score??'Unavailable'}</td><td>{decision.evidence_us??0} microseconds</td></tr>)}
   </Table><Button variant="secondary" onClick={()=>run(async()=>{const token=++selection.current;const value=await client.call('work.results',work.work_id??work.id,{limit:100});if(token===selection.current)setWork(value);})}>Refresh matching result</Button>
   {work.next_ordinal!=null&&<Button variant="secondary" onClick={()=>run(async()=>{const token=++selection.current;const value=await client.call('work.results',work.work_id??work.id,{limit:100,after_ordinal:work.next_ordinal});if(token===selection.current)setWork(old=>({...value,result:{...value.result,decisions:[...(old?.result?.decisions??[]),...(value.result?.decisions??[])]}}));})}>More matching outcomes</Button>}</>}
 </Card>;
}
