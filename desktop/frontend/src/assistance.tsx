// SPDX-License-Identifier: Apache-2.0
import {useEffect,useRef,useState,type RefObject} from 'react';
import {Actions,Area,Button,Card,Check,Input,Select} from './components';
import {Client,type Obj} from './client';
import type {Run} from './screens';
const defaults:Obj={enabled:false,adapter:'insonic-http',contract_version:'1',route:'local',endpoint:'',model:'',mode:'suggest',limits:{timeout_ms:30000,query_timeout_ms:20000,max_prompt_bytes:16384,max_response_bytes:262144,max_rows:100,max_result_bytes:524288,max_context_rows:25,max_context_bytes:32768}};
export function Assistance({client,run,fence,currentQuery,adopt,executed}:{client:Client;run:Run;fence:RefObject<number>;currentQuery:()=>Obj;adopt:(q:Obj)=>void;executed:(out:Obj)=>void}){
 const [config,setConfig]=useState<Obj>(defaults),[revision,setRevision]=useState(0),[prompt,setPrompt]=useState(''),[limits,setLimits]=useState(JSON.stringify(defaults.limits,null,2)),[excerpt,setExcerpt]=useState(false),[proposal,setProposal]=useState<Obj>(),[pending,setPending]=useState(false);
 const ownership=useRef(0),alive=useRef(true);
 useEffect(()=>{const request=ownership.current;run(async()=>{const out=await client.call('query.assistance-show');if(!alive.current||request!==ownership.current)return;setConfig(out.configuration??defaults);setRevision(out.revision??0);setLimits(JSON.stringify(out.configuration?.limits??defaults.limits,null,2))});return()=>{alive.current=false;ownership.current++}},[]);
 const invalidate=()=>{ownership.current++;setPending(false)};
 const change=(field:string,value:unknown)=>{invalidate();setConfig(p=>({...p,[field]:value}))};
 const configuration=()=>({...config,limits:JSON.parse(limits)});
 const generate=(mode:string)=>{const request=++ownership.current,editor=fence.current;setPending(true);setProposal(undefined);run(async()=>{
  try{
   let context:Obj|undefined;if(excerpt){context=JSON.parse(JSON.stringify(currentQuery()));if(context!.definition.mode!=='normalized')throw new Error('Result context requires a normalized query.');context!.definition.pagination={limit:Math.min(configuration().limits.max_context_rows??25,25)}}
   const out=await client.call('query.assist','',{prompt,configuration:configuration(),settings_revision:revision,mode,...(context?{context_query:context}:{})});
   if(!alive.current||request!==ownership.current||editor!==fence.current)return;setProposal(out);if(out.execution)executed(out);
  }catch(error){if(alive.current&&request===ownership.current&&editor===fence.current)throw error}
  finally{if(alive.current&&request===ownership.current)setPending(false)}
 })};
 return <Card heading="Optional query assistance">
  <p>Send a prompt to your elected local or hosted provider. Suggestions remain available for inspection and editing. Execution uses the current selected backend and original source clocks.</p>
  <Check label="Enable query assistance" value={Boolean(config.enabled)} onChange={v=>change('enabled',v)}/>
  <Actions><Select label="Assistance route" value={config.route} onChange={v=>change('route',v)} options={['local','hosted']}/><Input label="Assistance endpoint" value={config.endpoint??''} onChange={v=>change('endpoint',v)}/><Input label="Assistance model" value={config.model??''} onChange={v=>change('model',v)}/><Input label="Assistance credential ID" value={config.credential_id??''} onChange={v=>change('credential_id',v)}/></Actions>
  <Select label="Default assistance mode" value={config.mode??'suggest'} onChange={v=>change('mode',v)} options={['suggest','auto-run']}/>
  <details><summary>Assistance limits</summary><Area label="Assistance limits JSON" value={limits} onChange={v=>{invalidate();setLimits(v)}}/></details>
  <Actions><Button onClick={()=>{const request=++ownership.current;run(async()=>{const out=await client.call('query.assistance-set','',{expected_revision:revision,configuration:configuration()});if(!alive.current||request!==ownership.current)return;setConfig(out.configuration);setRevision(out.revision);setLimits(JSON.stringify(out.configuration.limits,null,2))})}}>Save assistance configuration</Button></Actions>
  <Area label="Assistance prompt" value={prompt} onChange={v=>{invalidate();setPrompt(v)}}/>
  <Check label="Include a bounded current result excerpt" value={excerpt} onChange={v=>{invalidate();setExcerpt(v)}}/>
  <Actions><Button disabled={!config.enabled||!prompt.trim()||pending} onClick={()=>generate('suggest')}>Suggest query</Button><Button disabled={!config.enabled||!prompt.trim()||pending} onClick={()=>generate('auto-run')}>Assist and run query</Button>{pending&&<Button onClick={invalidate}>Discard pending assistance</Button>}</Actions>
  {pending&&<p role="status">Waiting for assistance.</p>}
  {proposal&&<><p role="status">Proposal validation: {proposal.validation?.status}. {proposal.execution?'Executed current query.':'Suggestion ready.'}</p><p>{proposal.explanation}</p><pre aria-label="Complete proposed query">{JSON.stringify(proposal.proposal,null,2)}</pre><details><summary>Assistance provenance</summary><pre>{JSON.stringify(proposal.provenance,null,2)}</pre></details><Button onClick={()=>adopt(proposal.proposal)}>Load proposal into editor</Button></>}
 </Card>;
}
