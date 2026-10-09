// SPDX-License-Identifier: Apache-2.0
import { useEffect, useState } from 'react';
import { Actions, Area, Button, Card, Facts, Input, Select, Table } from './components';
import { Client, type Obj } from './client';
import { uuid } from './forms';
type Run = (action: () => Promise<void>) => void;

export function ModelSettings({client,run}: {client:Client;run:Run}) {
  const [aliases,setAliases]=useState<Obj>({items:[]}), [sources,setSources]=useState<Obj>({items:[]});
  const [alias,setAlias]=useState<Obj>(), [source,setSource]=useState<Obj>();
  const [aliasName,setAliasName]=useState(''),[target,setTarget]=useState('{"kind":"base","id":"","operation":"transcription"}');
  const [sourceName,setSourceName]=useState(''),[sourceConfig,setSourceConfig]=useState('{"url":"https://models.example.org/catalog.json"}');
  const [discoveryID,setDiscoveryID]=useState(''),[discovery,setDiscovery]=useState<Obj>();
  const [reference,setReference]=useState(''),[operation,setOperation]=useState('transcription'),[resolution,setResolution]=useState<Obj>();
  const load=async()=>{
    const [a,s]=await Promise.all([client.call('models.alias.list','',{limit:100}),client.call('models.source.list','',{limit:100})]);
    setAliases(a);setSources(s);
  };
  useEffect(()=>{run(load);},[client.workspace]);
  const saveAlias=async()=>{
    const id=alias?.id??uuid();
    const updated=await client.call('models.alias.set',id,{expected_revision:alias?.revision??0,alias:{id,name:aliasName,state:'active',target:JSON.parse(target)}});
    setAlias(updated);await load();
  };
  const saveSource=async()=>{
    const id=source?.id??uuid();
    const updated=await client.call('models.source.set',id,{expected_revision:source?.revision??0,source:{...JSON.parse(sourceConfig),id,name:sourceName,state:'active'}});
    setSource(updated);await load();
  };
  const more=async(kind:'alias'|'source')=>{
    const old=kind==='alias'?aliases:sources;
    const next=await client.call(`models.${kind}.list`,'',{after_id:old.next_id,limit:100});
    const updated={...next,items:[...old.items,...next.items]};
    if(kind==='alias')setAliases(updated);else setSources(updated);
  };
  return <>
    <Card heading="Model aliases">
      <p>Short names select exact versions. Retargeting affects future submissions; accepted jobs retain their frozen model selection.</p>
      <Table caption="Workspace model aliases" heads={['Alias','Operation','Exact target','State','Revision','Actions']}>
        {aliases.items.map((a:Obj)=><tr key={a.id}><td>{a.name}</td><td>{a.target.operation}</td><td>{a.target.kind}:{a.target.id??a.target.remote_model}</td><td>{a.state}</td><td>{a.revision}</td><td><Actions>
          <Button variant="secondary" onClick={()=>run(async()=>{const value=await client.call('models.alias.show',a.id);setAlias(value);setAliasName(value.name);setTarget(JSON.stringify(value.target,null,2));})}>Edit alias {a.name}</Button>
          {a.state==='active'&&<Button variant="secondary" onClick={()=>run(async()=>{await client.call('models.alias.remove',a.id,{expected_revision:a.revision});if(alias?.id===a.id)setAlias(undefined);await load();})}>Remove alias {a.name}</Button>}
        </Actions></td></tr>)}
      </Table>
      {aliases.next_id&&<Button variant="secondary" onClick={()=>run(()=>more('alias'))}>More model aliases</Button>}
      <Button variant="secondary" onClick={()=>{setAlias(undefined);setAliasName('');setTarget('{"kind":"base","id":"","operation":"transcription"}');}}>New model alias</Button>
      <Input label="Model alias name" value={aliasName} onChange={setAliasName}/>
      <Area label="Exact model target JSON" value={target} onChange={setTarget} description="Declare base or speaker kind with an exact ID and operation, or hosted kind with its configured adapter, endpoint and remote model. Credential values never belong here."/>
      <p>Expected alias revision: {alias?.revision??0}</p>
      <Button onClick={()=>run(async()=>{try{await saveAlias();}catch(error){await load();throw error;}})}>Save model alias</Button>
    </Card>
    <Card heading="Model sources and discovery">
      <p>Discovery reads configured catalog metadata. A discovered entry is not an installed or verified model. Custom pinned manifests remain supported.</p>
      <Table caption="Configured model sources" heads={['Name','State','Revision','Actions']}>
        {sources.items.map((s:Obj)=><tr key={s.id}><td>{s.name}</td><td>{s.state}</td><td>{s.revision}</td><td><Actions>
          <Button variant="secondary" onClick={()=>run(async()=>{const value=await client.call('models.source.show',s.id);setSource(value);setSourceName(value.name);const {id,name,revision,state,...config}=value;setSourceConfig(JSON.stringify(config,null,2));})}>Edit source {s.name}</Button>
          {s.state==='active'&&<Button variant="secondary" onClick={()=>run(async()=>{await client.call('models.source.remove',s.id,{expected_revision:s.revision});if(source?.id===s.id)setSource(undefined);await load();})}>Remove source {s.name}</Button>}
        </Actions></td></tr>)}
      </Table>
      {sources.next_id&&<Button variant="secondary" onClick={()=>run(()=>more('source'))}>More model sources</Button>}
      <Button variant="secondary" onClick={()=>{setSource(undefined);setSourceName('');setSourceConfig('{"url":"https://models.example.org/catalog.json"}');}}>New model source</Button>
      <Input label="Model source name" value={sourceName} onChange={setSourceName}/>
      <Area label="Model source configuration JSON" value={sourceConfig} onChange={setSourceConfig} description="Catalog URL, optional opaque credential_id and explicit loopback-only local_http. Use a declared source, never a credential-bearing URL."/>
      <p>Expected source revision: {source?.revision??0}</p>
      <Button onClick={()=>run(async()=>{try{await saveSource();}catch(error){await load();throw error;}})}>Save model source</Button>
      <Select label="Discovery source" value={discoveryID} onChange={setDiscoveryID} options={[{value:'',label:'Choose a configured source'},...sources.items.filter((s:Obj)=>s.state==='active').map((s:Obj)=>({value:s.id,label:s.name}))]}/>
      <Button variant="secondary" onClick={()=>run(async()=>{setDiscovery(await client.call('models.discover','',{source_id:discoveryID}));})}>Discover source models</Button>
      {discovery&&<><Table caption="Discovered catalog entries" heads={['Reference','Version','Capability','Availability','Compatibility','Actions']}>
        {(discovery.items??[]).map((entry:Obj)=><tr key={entry.reference}><td>{entry.reference}</td><td>{entry.manifest?.model_version}</td><td>{entry.manifest?.capabilities?.join('/')}</td><td>{entry.state}</td><td>{entry.compatible?'Compatible':'Incompatible'}</td><td>
          <Button variant="secondary" onClick={()=>{setReference(entry.reference);setOperation(entry.manifest?.capabilities?.find((value:string)=>['transcription','diarization','voice-matching','speaker-model-training'].includes(value))??'transcription');setResolution(undefined);}}>Inspect discovered {entry.selector}</Button>
        </td></tr>)}
      </Table><Facts value={discovery}/></>}
    </Card>
    <Card heading="Inspect a model reference">
      <Input label="Model reference" value={reference} onChange={setReference} description="Alias, exact UUID, base:UUID, speaker:UUID or source:NAME/SELECTOR."/>
      <Select label="Model operation" value={operation} onChange={setOperation} options={['transcription','diarization','voice-matching','speaker-model-training']}/>
      <Button onClick={()=>run(async()=>{setResolution(await client.call('models.resolve','',{reference,operation}));})}>Resolve model reference</Button>
      {resolution&&<><p>{resolution.compatible?'Compatible':'Incompatible'}: {resolution.state}. Unsupported matching or training adapters remain unavailable.</p><Facts value={resolution}/></>}
      <Button variant="secondary" onClick={()=>run(load)}>Refresh model references</Button>
    </Card>
  </>;
}
