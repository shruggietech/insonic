// SPDX-License-Identifier: Apache-2.0
import {useEffect,useRef,useState} from 'react';
import {Actions,Area,Button,Card,Check,Input,Select,Table} from './components';
import {Client,result,type Obj} from './client';
import {uuid} from './forms';
import type {Run} from './screens';
import {GraphView,type Positions} from './graph-view';
const portable=['text-search','media-list','speaker-search','model-list','time-range','evidence-traverse','graph-view'];
function monthViewport(date:Date,months:number){const from=new Date(Date.UTC(date.getUTCFullYear(),date.getUTCMonth(),1)),through=new Date(Date.UTC(date.getUTCFullYear(),date.getUTCMonth()+months,0));return {from:from.toISOString().slice(0,10),through:through.toISOString().slice(0,10)}}
export function Explore({client,run}:{client:Client;run:Run}){
 const [tab,setTab]=useState('Calendar'),[calendar,setCalendar]=useState<Obj>({items:[]}),[view,setView]=useState(monthViewport(new Date(),1)),[months,setMonths]=useState(1),[zone,setZone]=useState('UTC'),[undated,setUndated]=useState(true);
 const [query,setQuery]=useState<Obj>({items:[]}),[operation,setOperation]=useState('text-search'),[text,setText]=useState(''),[mode,setMode]=useState('normalized'),[dialect,setDialect]=useState('ladybug-cypher'),[native,setNative]=useState('MATCH (n:Entity) RETURN n.entity_id AS id, n.kind AS kind LIMIT 100'),[params,setParams]=useState('{}');
 const [mediaID,setMediaID]=useState(''),[speakerID,setSpeakerID]=useState(''),[start,setStart]=useState(''),[end,setEnd]=useState(''),[modelKind,setModelKind]=useState(''),[depth,setDepth]=useState('1'),[relationships,setRelationships]=useState('');
 const [selected,setSelected]=useState<Obj>(),[recording,setRecording]=useState<Obj>({items:[]}),[playback,setPlayback]=useState<Obj>(),[saved,setSaved]=useState<Obj[]>([]),[versions,setVersions]=useState<Obj[]>([]),[queryID,setQueryID]=useState(''),[queryRevision,setQueryRevision]=useState(0),[title,setTitle]=useState('Current source search'),[layoutID,setLayoutID]=useState(uuid()),[layoutRevision,setLayoutRevision]=useState(0),[positions,setPositions]=useState<Positions>({}),[paused,setPaused]=useState(false);
 const [graphFilter,setGraphFilter]=useState('');
 const selection=useRef(0),calendarRequest=useRef(0),queryRequest=useRef(0),sourceRequest=useRef(0),savedRequest=useRef(0),media=useRef<HTMLMediaElement>(null),alive=useRef(true);
 const exactClock=(value:string)=>{const n=Number(value);if(!Number.isSafeInteger(n)||n<0)throw new Error('Source interval values must be exact nonnegative microsecond integers.');return n};
 // Preserve fields the compact editor does not expose, including advanced CLI queries.
 const loadedQuery=useRef<Obj|undefined>(undefined),queryEdits=useRef(new Set<string>());
 const editField=(field:string,setter:(value:string)=>void)=>(value:string)=>{queryEdits.current.add(field);setter(value)};
 const queryDefinition=(cursor=''):Obj=>{
  const current:Obj=mode==='native'?{title,definition:{mode,dialect,text:native},parameters:JSON.parse(params)}:{title,definition:{mode,operation,filters:{...(text?{text}:{}),...(mediaID?{media_ids:mediaID.split(',').map(v=>v.trim()).filter(Boolean)}:{}),...(speakerID?{speaker_ids:speakerID.split(',').map(v=>v.trim()).filter(Boolean)}:{}),...(modelKind?{model_kind:modelKind}:{}),...(start&&end?{source_interval:{start_us:exactClock(start),end_us:exactClock(end)}}:{})},...(operation==='graph-view'||operation==='evidence-traverse'?{traversal:{direction:'both',max_depth:Number(depth),...(relationships?{relationship_types:relationships.split(',').map(v=>v.trim()).filter(Boolean)}:{})}}:{}),order_by:[{field:'id',direction:'asc'}],pagination:{limit:200}}};
  const base=loadedQuery.current,edits=queryEdits.current;
  const out:Obj=!base||base.definition.mode!==mode?current:JSON.parse(JSON.stringify(base));out.title=title;
  if(base&&base.definition.mode===mode){
   if(edits.has('operation'))out.definition.operation=operation;
   if(mode==='normalized'){
    for(const key of ['text','media_ids','speaker_ids','model_kind','source_interval'])if(edits.has(key)){
     out.definition.filters??={};const value=current.definition.filters[key];
     if(value===undefined)delete out.definition.filters[key];else out.definition.filters[key]=value;
    }
    if(edits.has('max_depth')||edits.has('relationship_types')){
     out.definition.traversal??={direction:'both',max_depth:1};
     if(edits.has('max_depth'))out.definition.traversal.max_depth=Number(depth);
     if(edits.has('relationship_types'))out.definition.traversal.relationship_types=relationships.split(',').map(v=>v.trim()).filter(Boolean);
    }
    if(edits.has('operation')&&!['graph-view','evidence-traverse'].includes(operation))delete out.definition.traversal;
   }else{
    if(edits.has('dialect'))out.definition.dialect=dialect;
    if(edits.has('native'))out.definition.text=native;
    if(edits.has('parameters'))out.parameters=JSON.parse(params);
   }
  }
  if(cursor){out.definition.pagination??={limit:200};out.definition.pagination.cursor=cursor}
  return out;
 };
 const loadCalendar=async(cursor='',viewport=view)=>{
  const request=++calendarRequest.current;const response=await client.call('timeline.calendar','',{filter:{...viewport,timezone:zone,include_undated:undated},pagination:{limit:100,...(cursor?{cursor}:{})}});
  if(!alive.current||request!==calendarRequest.current)return;setCalendar(previous=>({...response,items:cursor?[...previous.items,...response.items]:response.items}));
 };
 const refreshSaved=async()=>{const response=await client.call('query.list');if(alive.current)setSaved(response.items)};
 useEffect(()=>{run(async()=>{await loadCalendar();await refreshSaved()});return()=>{alive.current=false;selection.current++;calendarRequest.current++;queryRequest.current++;sourceRequest.current++}},[]);
 useEffect(()=>{queryRequest.current++},[mode,operation,text,mediaID,speakerID,start,end,modelKind,depth,relationships,dialect,native,params]);
 const execute=async(cursor='',definition?:Obj)=>{
  const request=++queryRequest.current;const response=await client.call('query.run','',definition??queryDefinition(cursor));
  if(!alive.current||request!==queryRequest.current)return;setQuery(previous=>({...response,items:cursor?[...previous.items,...response.items]:response.items,edges:cursor?[...(previous.edges??[]),...(response.edges??[])]:response.edges??[]}));
 };
 const inspect=async(row:Obj)=>{
  const request=++sourceRequest.current;selection.current++;media.current?.pause();setPlayback(undefined);setSelected(row);setRecording({items:[]});
  if(row.media_id){const response=await client.call('timeline.recording',row.media_id,{pagination:{limit:100}});if(alive.current&&request===sourceRequest.current){setRecording(response);if(!row.document_digest&&response.items.length){const current=response.items[0];setSelected({...row,recording_revision:current.recording_revision,document_digest:current.document_digest})}}}
 };
 const play=async(row:Obj)=>{
  const request=selection.current;const response=await client.bridge.Playback(row.media_id,row.media_revision??0,row.recording_revision??0,row.document_digest??'');
  if(!alive.current||request!==selection.current){if(response.result?.url)await client.bridge.ClosePlayback(response.result.url);return}
  const ticket=result(response);setPlayback({...ticket,seek:row.seek_seconds??0});
 };
 useEffect(()=>()=>{if(playback?.url)void client.bridge.ClosePlayback(playback.url).catch(()=>{})},[playback?.url]);
 useEffect(()=>{
  if(!playback?.url)return;let active=true,checking=false;const request=selection.current;
  const timer=setInterval(async()=>{if(checking)return;checking=true;try{const response=await client.bridge.VerifyPlayback(playback.url);if(!active||request!==selection.current)return;result(response)}catch{if(!active||request!==selection.current)return;media.current?.pause();setPlayback(undefined);setSelected(undefined);setRecording({items:[]});run(async()=>{throw new Error('The source changed or playback expired. Refresh current results before playing again.')})}finally{checking=false}},5000);
  return()=>{active=false;clearInterval(timer)};
 },[playback?.url]);
 useEffect(()=>{
  if(!selected?.media_id||!selected.document_digest)return;let active=true,checking=false;const request=sourceRequest.current;
  const timer=setInterval(async()=>{if(checking)return;checking=true;try{await client.call('recordings.cues',selected.media_id,{revision:selected.recording_revision,document_digest:selected.document_digest,limit:1})}catch{if(!active||request!==sourceRequest.current)return;sourceRequest.current++;selection.current++;media.current?.pause();setPlayback(undefined);setSelected(undefined);setRecording({items:[]});setQuery({items:[]});run(async()=>{throw new Error('The current recording changed. Refresh the results before inspecting or playing it.')})}finally{checking=false}},5000);
  return()=>{active=false;clearInterval(timer)};
 },[selected?.media_id,selected?.recording_revision,selected?.document_digest]);
 useEffect(()=>{if(media.current&&media.current.readyState>=1&&playback)media.current.currentTime=Math.max(0,playback.seek-(playback.timeline_offset_seconds??0))},[playback]);
 const shift=(direction:number)=>{const date=new Date(view.from+'T00:00:00Z');date.setUTCMonth(date.getUTCMonth()+direction*months);const next=monthViewport(date,months);calendarRequest.current++;setView(next);run(()=>loadCalendar('',next))};
 const zoom=(nextMonths:number)=>{setMonths(nextMonths);const next=monthViewport(new Date(view.from+'T00:00:00Z'),nextMonths);setView(next);calendarRequest.current++;run(()=>loadCalendar('',next))};
 const chooseSaved=async(id:string,revision=0)=>{
  ++queryRequest.current;const request=++savedRequest.current;const response=await client.call('query.show',id);if(!alive.current||request!==savedRequest.current)return;setVersions(response.items);const v=revision?response.items.find((value:Obj)=>value.revision===revision):response.items.at(-1);if(!v)throw new Error('The selected query version is unavailable.');loadedQuery.current=JSON.parse(JSON.stringify({title:v.title,definition:v.definition,...(v.parameters?{parameters:v.parameters}:{})}));queryEdits.current.clear();setQueryID(id);setQueryRevision(v.revision);setTitle(v.title);setMode(v.definition.mode);setOperation(v.definition.operation??'graph-view');setText(v.definition.filters?.text??'');setMediaID((v.definition.filters?.media_ids??[]).join(', '));setSpeakerID((v.definition.filters?.speaker_ids??[]).join(', '));setModelKind(v.definition.filters?.model_kind??'');setStart(String(v.definition.filters?.source_interval?.start_us??''));setEnd(String(v.definition.filters?.source_interval?.end_us??''));setDepth(String(v.definition.traversal?.max_depth??1));setRelationships((v.definition.traversal?.relationship_types??[]).join(','));setDialect(v.definition.dialect??'ladybug-cypher');setNative(v.definition.text??'');setParams(JSON.stringify(v.parameters??{},null,2));setLayoutRevision(0);setLayoutID(uuid());setPositions({});
 };
 return <>
  <Actions>{['Calendar','Query','Graph'].map(v=><Button key={v} variant={tab===v?'primary':'secondary'} onClick={()=>{setTab(v);if(v==='Graph'){queryEdits.current.add('mode');queryEdits.current.add('operation');setMode('normalized');setOperation('graph-view')}}}>{v}</Button>)}</Actions>
  {tab==='Calendar'?<Card heading="Origination calendar">
   <p>Browse the entire library by its selected origination date. Date-only, approximate and undated sources retain their interpretation.</p>
   <Actions><Input label="Calendar from" type="date" value={view.from} onChange={v=>{calendarRequest.current++;setView(p=>({...p,from:v}))}}/><Input label="Calendar through" type="date" value={view.through} onChange={v=>{calendarRequest.current++;setView(p=>({...p,through:v}))}}/><Input label="Calendar timezone" value={zone} onChange={v=>{calendarRequest.current++;setZone(v)}}/><Check label="Include undated recordings" value={undated} onChange={v=>{calendarRequest.current++;setUndated(v)}}/></Actions>
   <Actions><Button onClick={()=>run(()=>loadCalendar())}>Apply calendar viewport</Button><Button onClick={()=>shift(-1)}>Previous period</Button><Button onClick={()=>shift(1)}>Next period</Button><Button onClick={()=>zoom(Math.max(1,Math.floor(months/2)))}>Zoom calendar in</Button><Button onClick={()=>zoom(Math.min(120,months*2))}>Zoom calendar out</Button></Actions>
   <div className="calendar-entries" role="group" aria-label="Recording calendar. Arrow keys change period." tabIndex={0} onKeyDown={e=>{if(e.key==='ArrowLeft'||e.key==='ArrowRight'){e.preventDefault();shift(e.key==='ArrowLeft'?-1:1)}}}>
    {calendar.items.map((r:Obj)=><Button key={r.id} onClick={()=>run(()=>inspect(r))}>{r.recording_date||'Undated'} · {r.label}{r.provenance?.selected_dates?.selected?.precision==='range'?' · Approximate range':''}</Button>)}
   </div>
   <p>{calendar.items.length} of {calendar.total??0} matching recordings loaded.</p>{calendar.next_cursor&&<Button onClick={()=>run(()=>loadCalendar(calendar.next_cursor))}>Load more calendar recordings</Button>}
  </Card>:<Card heading={tab==='Graph'?'Evidence relationships':'Search and saved queries'}>
   <Actions><Select label="Query mode" value={mode} onChange={editField('mode',setMode)} options={['normalized','native']}/>{mode==='normalized'?<Select label="Search operation" value={operation} onChange={editField('operation',setOperation)} options={portable}/>:<Select label="Native query dialect" value={dialect} onChange={editField('dialect',setDialect)} options={['ladybug-cypher','arcade-opencypher','arcade-sql']}/>}</Actions>
   {mode==='normalized'?<>
    <Input label="Words or terms" value={text} onChange={editField('text',setText)}/><Actions><Input label="Media ID filter" value={mediaID} onChange={editField('media_ids',setMediaID)}/><Input label="Speaker ID filter" value={speakerID} onChange={editField('speaker_ids',setSpeakerID)}/><Input label="Model kind filter" value={modelKind} onChange={editField('model_kind',setModelKind)}/></Actions>
    <Actions><Input label="Source interval start (microseconds)" value={start} onChange={editField('source_interval',setStart)}/><Input label="Source interval end (microseconds)" value={end} onChange={editField('source_interval',setEnd)}/><Input label="Relationship depth" type="number" value={depth} onChange={editField('max_depth',setDepth)}/><Input label="Relationship types (comma separated)" value={relationships} onChange={editField('relationship_types',setRelationships)}/></Actions>
   </>:<><Area label="Native read query" value={native} onChange={editField('native',setNative)}/><Area label="Typed parameters JSON" value={params} onChange={editField('parameters',setParams)}/></>}
   <Actions><Button onClick={()=>run(()=>execute())}>Run current query</Button><Button onClick={()=>run(async()=>{const r=await client.call('query.explain','',queryDefinition());if(alive.current)setQuery({items:[],explanation:r})})}>Explain query</Button><Button onClick={()=>run(async()=>{await client.call('graph.rebuild');await execute()})}>Rebuild evidence graph</Button></Actions>
   <Input label="Saved query title" value={title} onChange={setTitle}/>
   <Actions><Button onClick={()=>run(async()=>{const id=queryID||uuid();const q=await client.call('query.save',id,{expected_revision:queryRevision,query:queryDefinition()});if(!alive.current)return;setQueryID(id);setQueryRevision(q.revision);await refreshSaved()})}>Save query version</Button><Button onClick={()=>{setQueryID('');setQueryRevision(0);setVersions([]);setLayoutID(uuid());setLayoutRevision(0)}}>Create separate saved query</Button><Select label="Saved query" value={queryID} onChange={id=>{if(id)run(()=>chooseSaved(id))}} options={[{value:'',label:'Choose saved query'},...saved.map(q=>({value:q.query_id,label:q.title}))]}/>{versions.length>1&&<Select label="Saved query revision" value={String(queryRevision)} onChange={v=>run(()=>chooseSaved(queryID,Number(v)))} options={versions.map(v=>({value:String(v.revision),label:'Version '+v.revision}))}/> }</Actions>
   {query.graph_error&&<p role="status">Showing accepted current catalog evidence. Graph publication is unavailable ({query.graph_error}).</p>}
   {query.explanation&&<pre>{JSON.stringify(query.explanation,null,2)}</pre>}
   {operation==='graph-view'&&mode==='normalized'?<>
    <GraphView filter={graphFilter} setFilter={setGraphFilter} rows={query.items} edges={query.edges??[]} select={r=>run(()=>inspect(r))} expand={r=>run(()=>execute('',{title,definition:{mode:'normalized',operation:'graph-view',filters:{entity_ids:[r.id]},traversal:{direction:'both',max_depth:Number(depth)},pagination:{limit:200}}}))} positions={positions} setPositions={setPositions} paused={paused} setPaused={setPaused}/>
    <Actions><Input label="Saved graph layout ID" value={layoutID} onChange={v=>{setLayoutID(v);setLayoutRevision(0)}}/><Button disabled={!queryID} onClick={()=>run(async()=>{const layout=await client.call('views.save',layoutID,{expected_revision:layoutRevision,query_id:queryID,layout:{positions,paused,filter:graphFilter,limit:200}});if(alive.current)setLayoutRevision(layout.revision)})}>Save graph layout</Button><Button onClick={()=>run(async()=>{const layout=await client.call('views.show',layoutID);if(!alive.current)return;setPositions(layout.layout.positions??{});setPaused(layout.layout.paused??false);setGraphFilter(layout.layout.filter??'');setLayoutRevision(layout.revision)})}>Load graph layout</Button></Actions>
   </>:<Table caption="Current query results" heads={['Source','Kind','Original-clock position','Action']}>{query.items.map((r:Obj,i:number)=><tr key={r.id??i}><td>{r.label??JSON.stringify(r)}</td><td>{r.kind??dialect}</td><td>{r.source_start_us??'Untimed'}</td><td>{r.media_id&&<Button onClick={()=>run(()=>inspect(r))}>Inspect source</Button>}</td></tr>)}</Table>}
   <p>{query.items.length} of {query.total??query.items.length} matching rows loaded. {Math.max(0,(query.total??query.items.length)-query.items.length)} rows not yet loaded. {query.omitted_edges??0} of {query.total_edges??(query.edges??[]).length} relationships outside the last returned page.</p>{query.next_cursor&&<Button onClick={()=>run(()=>execute(query.next_cursor))}>Load more query results</Button>}
  </Card>}
  {selected&&<Card heading="Current source and recording timeline">
   <p>{selected.label}</p>{selected.statement&&<><p>{selected.statement.subject} · {selected.statement.relation} · {selected.statement.object}</p><p>{selected.statement.polarity} · {selected.statement.modality} · {(selected.statement.conditions??[]).join('; ')}</p>{selected.statement.quoted_attribution&&<p>Quoted attribution: {selected.statement.quoted_attribution}</p>}</>}
   {selected.provenance&&<details><summary>Source provenance and date interpretation</summary><pre>{JSON.stringify(selected.provenance,null,2)}</pre></details>}
   <Actions>{selected.media_id&&<Button onClick={()=>run(()=>play(selected))}>Play selected source</Button>}{selected.document_digest&&<Button onClick={()=>run(async()=>{await client.call('evidence.extract',selected.media_id,{expected_revision:selected.recording_revision,document_digest:selected.document_digest,configuration:{adapter:'cue-statement',contract_version:'1',window_cues:16,overlap_cues:2}})})}>Extract literal source statements</Button>}</Actions>
   {playback&&(playback.mime_type?.startsWith('video/')?<video ref={media as React.RefObject<HTMLVideoElement>} src={playback.url} controls onLoadedMetadata={()=>{if(media.current)media.current.currentTime=Math.max(0,playback.seek-(playback.timeline_offset_seconds??0))}}/>:<audio ref={media as React.RefObject<HTMLAudioElement>} src={playback.url} controls onLoadedMetadata={()=>{if(media.current)media.current.currentTime=Math.max(0,playback.seek-(playback.timeline_offset_seconds??0))}}/>)}
   <Table caption="Source-clock recording timeline" heads={['Original time (microseconds)','Current cue or assertion','Voices','Action']}>{recording.items.map((r:Obj)=><tr key={r.id}><td>{r.source_start_us??'Untimed'} {r.source_end_us?'to '+r.source_end_us:''}</td><td>{r.text||r.label}</td><td>{[...(r.local_speaker_ids??[]),...(r.speaker_ids??[])].join(', ')}</td><td><Button onClick={()=>run(()=>play(r))}>Play from source</Button><Button onClick={()=>{selection.current++;media.current?.pause();setPlayback(undefined);setSelected(r)}}>Inspect evidence</Button></td></tr>)}</Table>
   {recording.next_cursor&&<Button onClick={()=>run(async()=>{const request=sourceRequest.current;const response=await client.call('timeline.recording',selected.media_id,{pagination:{limit:100,cursor:recording.next_cursor}});if(alive.current&&request===sourceRequest.current)setRecording(previous=>({...response,items:[...previous.items,...response.items]}))})}>Load more recording intervals</Button>}
  </Card>}
 </>;
}
