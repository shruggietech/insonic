// SPDX-License-Identifier: Apache-2.0
import {useEffect,useRef,useState,type RefObject} from 'react';
import {Actions,Area,Button,Card,Check,Input,Select} from './components';
import {Client,type Obj} from './client';
import type {Run} from './screens';
import {result} from './client';
import {uuid} from './forms';
const defaults:Obj={enabled:false,adapter:'insonic-http',contract_version:'1',route:'local',endpoint:'',model:'',mode:'suggest',limits:{timeout_ms:30000,query_timeout_ms:20000,max_prompt_bytes:16384,max_response_bytes:262144,max_rows:100,max_result_bytes:524288,max_context_rows:25,max_context_bytes:32768}};
export function Assistance({client,run,fence,currentQuery,adopt,executed}:{client:Client;run:Run;fence:RefObject<number>;currentQuery:()=>Obj;adopt:(q:Obj)=>void;executed:(out:Obj)=>void}){
 const [config,setConfig]=useState<Obj>(defaults),[revision,setRevision]=useState(0),[prompt,setPrompt]=useState(''),[limits,setLimits]=useState(JSON.stringify(defaults.limits,null,2)),[excerpt,setExcerpt]=useState(false),[proposal,setProposal]=useState<Obj>(),[pending,setPending]=useState(false);
 const [discovery,setDiscovery]=useState<Obj>(),[test,setTest]=useState<Obj>(),[token,setToken]=useState('');
 const abort=useRef<AbortController|undefined>(undefined);
 const ownership=useRef(0),alive=useRef(true);
 useEffect(()=>{const request=ownership.current;run(async()=>{const out=await client.call('query.assistance-show');if(!alive.current||request!==ownership.current)return;setConfig(out.configuration??defaults);setRevision(out.revision??0);setLimits(JSON.stringify(out.configuration?.limits??defaults.limits,null,2))});return()=>{alive.current=false;ownership.current++;abort.current?.abort()}},[]);
 const invalidate=()=>{ownership.current++;abort.current?.abort();setPending(false);setTest(undefined)};
 const change=(field:string,value:unknown)=>{invalidate();setConfig(p=>({...p,[field]:value}))};
 const configuration=()=>({...config,limits:JSON.parse(limits)});
 const generate=(mode:string)=>{abort.current?.abort();abort.current=new AbortController();const request=++ownership.current,editor=fence.current;setPending(true);setProposal(undefined);run(async()=>{
  try{
   let context:Obj|undefined;if(excerpt){context=JSON.parse(JSON.stringify(currentQuery()));if(context!.definition.mode!=='normalized')throw new Error('Result context requires a normalized query.');context!.definition.pagination={limit:Math.min(configuration().limits.max_context_rows??25,25)}}
   const out=await client.call('query.assist','',{prompt,configuration:configuration(),settings_revision:revision,mode,...(context?{context_query:context}:{})},abort.current?.signal);
   if(!alive.current||request!==ownership.current||editor!==fence.current)return;setProposal(out);if(out.execution)executed(out);
  }catch(error){if(alive.current&&request===ownership.current&&editor===fence.current)throw error}
  finally{if(alive.current&&request===ownership.current)setPending(false)}
 })};
 return <Card heading="Optional query assistance">
  <p>Send a prompt to your elected local or hosted provider. Suggestions remain available for inspection and editing. Execution uses the current selected backend and original source clocks.</p>
  <Select label="Assistance adapter" value={config.adapter??'insonic-http'} options={['insonic-http','ollama']} onChange={v=>change('adapter',v)}/>
  <Check label="Enable query assistance" value={Boolean(config.enabled)} onChange={v=>change('enabled',v)}/>
  <Actions><Select label="Assistance route" value={config.route} onChange={v=>change('route',v)} options={['local','hosted']}/><Input label="Assistance endpoint" value={config.endpoint??''} onChange={v=>change('endpoint',v)}/><Input label="Assistance model" value={config.model??''} onChange={v=>change('model',v)}/><Input label="Assistance credential ID" value={config.credential_id??''} onChange={v=>change('credential_id',v)}/></Actions>
  {config.adapter==='ollama'&&<>
   <p>Native Ollama uses metadata to distinguish endpoint location from local or hosted inference. Discovery changes no saved selection and sends no library contents or prompts.</p>
   <Actions><Button onClick={()=>{invalidate();abort.current=new AbortController();const request=ownership.current;run(async()=>{const out=await client.call('query.services','',{...(config.endpoint?{endpoints:[config.endpoint]}:{})},abort.current?.signal);if(alive.current&&request===ownership.current)setDiscovery(out)})}}>Discover Ollama services and models</Button>
   <Button onClick={()=>{invalidate();abort.current=new AbortController();const request=ownership.current;run(async()=>{const out=await client.call('query.assistance-test','',{settings_revision:revision},abort.current?.signal);if(alive.current&&request===ownership.current)setTest(out)})}}>Test saved model (may load it)</Button><Button variant="secondary" onClick={invalidate}>Cancel service operation</Button></Actions>
   {discovery&&<><p>Observed at {discovery.checked_at}. {discovery.truncated?'Some candidates or model details were not probed within the limits.':''}</p>{discovery.candidates?.map((candidate:Obj,i:number)=><fieldset key={i}><legend>{candidate.endpoint||'Invalid endpoint hint'}</legend><p>Sources: {candidate.sources?.join(', ')}. Installed: {String(candidate.installed)}; running: {String(candidate.running)}; reachable: {String(candidate.reachable)}; configured: {String(candidate.configured)}; tested: {String(candidate.tested)}.</p>{candidate.correction&&<p>{candidate.correction}</p>}{candidate.metadata?.models?.map((model:Obj)=><div key={model.name}><p>{model.name}: inference {model.inference_route}, text capability {String(model.task_compatible)}, metadata {model.detail_state}.</p>{model.correction&&<p>{model.correction}</p>}<Button onClick={()=>{invalidate();setConfig(p=>({...p,adapter:'ollama',endpoint:candidate.endpoint,model:model.name,route:model.inference_route==='local'?'local':'hosted'}))}}>Use {model.name} with {model.inference_route==='local'?'local':'hosted/unknown'} inference</Button></div>)}</fieldset>)}</>}
   {test&&<p role="status">Synthetic query assistance test passed for {test.endpoint}, model {test.model}, inference {test.inference_route}. No library content was sent.</p>}
  </>}
  <Input label="New assistance bearer token" type="password" value={token} onChange={setToken}/><Button disabled={!token} onClick={()=>run(async()=>{const id=uuid();try{result(await client.bridge.Credential(['add',id],JSON.stringify({value:token})));change('credential_id',id)}finally{setToken('')}})}>Store assistance credential and use its reference</Button>
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
