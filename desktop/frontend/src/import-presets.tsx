// SPDX-License-Identifier: Apache-2.0
import {useEffect,useRef,useState} from 'react';
import {Actions,Area,Button,Card,Input,Select,Table} from './components';
import {type Client,type Obj} from './client';
import {uuid} from './forms';
import {type Run} from './screens';
export function ImportPresets({client,run}:{client:Client;run:Run}){
 const selectionOwner=useRef(0);
 useEffect(()=>()=>{selectionOwner.current++},[]);
 const [page,setPage]=useState<Obj>({items:[]}),[current,setCurrent]=useState<Obj>(),[name,setName]=useState(''),[zone,setZone]=useState(''),[rule,setRule]=useState('owner-first'),[pipeline,setPipeline]=useState(''),[destination,setDestination]=useState(''),[pipelines,setPipelines]=useState<Obj[]>([]);
 const load=async(after='')=>{setPage(await client.call('import-presets.list','',{limit:25,...(after?{after_id:after}:{})}));};
 useEffect(()=>{run(async()=>{await load();const p=await client.call('pipelines.list','',{limit:100});setPipelines(p.items);});},[client.workspace]);
 const select=async(id:string)=>{const owner=++selectionOwner.current;const p=await client.call('import-presets.show','',{selection:id});if(owner!==selectionOwner.current)return;setCurrent(p);setName(p.name);setZone(p.options.timezone??'');setRule(p.options.date_precedence??'owner-first');setPipeline(p.options.pipeline_id??'');setDestination(p.options.destination?JSON.stringify(p.options.destination,null,2):'');};
 const reset=()=>{selectionOwner.current++;setCurrent(undefined);setName('');setZone('');setRule('owner-first');setPipeline('');setDestination('');};
 return <>
 <Card heading="Saved import presets"><p>Save destination, date policy and optional configured processing choices. Select a preset when adding media or refreshing metadata.</p><Table caption="Saved import presets" heads={['Name','ID','Revision','Processing','Action']}>{page.items.map((p:Obj)=><tr key={p.id}><td>{p.name}</td><td>{p.id}</td><td>{p.revision}</td><td>{p.options.pipeline_id?'Configured pipeline':'Admission only'}</td><td><Button onClick={()=>run(()=>select(p.id))}>Edit {p.name}</Button></td></tr>)}{page.items.length===0&&<tr><td colSpan={5}>No saved import presets. Create one below.</td></tr>}</Table><Actions><Button variant="secondary" onClick={()=>run(()=>load())}>Refresh presets</Button>{page.next_id&&<Button variant="secondary" onClick={()=>run(()=>load(page.next_id))}>Next page</Button>}<Button onClick={reset}>New preset</Button></Actions></Card>
 <Card heading={current?'Edit import preset':'Create import preset'}><form onSubmit={e=>{e.preventDefault();run(async()=>{const owner=selectionOwner.current;const options:Obj={...current?.options,date_precedence:rule,pipeline_id:pipeline};if(zone)options.timezone=zone;else delete options.timezone;if(destination.trim())options.destination=JSON.parse(destination);else delete options.destination;const preset={kind:'import-preset',schema_version:'1.0.3',id:current?.id??uuid(),name:name.trim(),options};const saved=await client.call(current?'import-presets.update':'import-presets.create',current?.id??'',{preset,...(current?{expected_revision:current.revision}:{})});if(owner!==selectionOwner.current)return;await load();if(owner!==selectionOwner.current)return;await select(saved.id);});}}>
 <Input label="Preset name" value={name} onChange={setName} required/>
 <Input label="Preset timezone" value={zone} onChange={setZone}/>
 <Select label="Preset date policy" value={rule} onChange={setRule} options={['owner-first','embedded-first','filesystem-fallback'].map(value=>({value,label:value}))}/>
 <Select label="Import processing pipeline" value={pipeline} onChange={setPipeline} options={[{value:'',label:'Admission only'},...pipelines.map(p=>({value:p.id,label:p.name}))]}/>
 <Input label="Pipeline UUID" value={pipeline} onChange={setPipeline}/>
 <Area label="Destination storage profile JSON" value={destination} onChange={setDestination} description="Leave empty for the workspace storage destination. An explicit filesystem/S3 profile includes profile_id, profile_revision, adapter_id, contract_version and nonsecret configuration. Credential values stay in configured credential IDs."/>
 <Actions><Button type="submit">{current?'Save preset':'Create preset'}</Button>{current&&<Button variant="secondary" onClick={()=>run(async()=>{const owner=selectionOwner.current;await client.call('import-presets.delete',current.id,{expected_revision:current.revision});if(owner!==selectionOwner.current)return;reset();await load();})}>Delete preset</Button>}</Actions>
 </form></Card></>;
}
