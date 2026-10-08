// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import {JSDOM} from 'jsdom';
import {validateNativeRequest} from './contracts.mjs';
const dom=new JSDOM('<!doctype html><html><body><div id="root"></div></body></html>',{url:'http://wails.localhost'});
for(const key of ['window','document','HTMLElement','HTMLInputElement','HTMLTextAreaElement','HTMLSelectElement','Event','MouseEvent'])globalThis[key]=dom.window[key];
globalThis.IS_REACT_ACT_ENVIRONMENT=true;window.matchMedia=()=>({matches:false,addEventListener(){},removeEventListener(){}});
const {act,createElement}=await import('react'),{createRoot}=await import('react-dom/client'),{App}=await import('../.test-build/App.js');
const wid='aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',mid='bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb',qid='cccccccc-cccc-4ccc-8ccc-cccccccccccc';
let root;
const tick=async()=>{for(let i=0;i<4;i++)await act(()=>new Promise(r=>setTimeout(r,0)))};
function button(text){const b=[...document.querySelectorAll('button')].find(b=>b.textContent===text);assert.ok(b,'Missing '+text);return b}
async function click(text){await act(async()=>button(text).click());await tick()}
async function fill(label,value){const l=[...document.querySelectorAll('label')].find(l=>l.textContent===label);assert.ok(l,'Missing '+label);const f=document.getElementById(l.htmlFor);await act(async()=>{const proto=f.tagName==='SELECT'?HTMLSelectElement.prototype:f.tagName==='TEXTAREA'?HTMLTextAreaElement.prototype:HTMLInputElement.prototype;Object.getOwnPropertyDescriptor(proto,'value').set.call(f,value);f.dispatchEvent(new Event('input',{bubbles:true}));f.dispatchEvent(new Event('change',{bubbles:true}))});await tick()}
const row={id:'cue:'+mid,kind:'cue',label:'Source statement',text:'Source statement',media_id:mid,media_revision:4,recording_revision:9,document_digest:'5'.repeat(64),source_start_us:'2000000',source_end_us:'3000000',seek_seconds:2,local_speaker_ids:[mid],speaker_ids:[]};
function mock(){
 const calls=[],queries=[];
 const bridge={
  calls,Show:async()=>({workspace_id:wid,result:{display_name:'Explore fixture'}}),
  Operate:async(request)=>{const check=validateNativeRequest(request);assert.equal(check.valid,true,request.operation+': '+check.errors);calls.push(request);let value={items:[]};
   switch(request.operation){
    case 'settings.show':value={sections:{appearance:{value:{theme:'system',reduced_motion:true}}}};break;
    case 'timeline.calendar':value={items:request.data.pagination.cursor?[{...row,id:'media:undated',label:'Undated recording',recording_date:''}]:[{...row,id:'media:one',label:'First period',recording_date:'2026-10-01'},{...row,id:'media:two',label:'Another period',recording_date:'2026-10-02'}],total:3,next_cursor:request.data.pagination.cursor?'':'later'};break;
    case 'timeline.recording':value={items:[row],total:1};break;
    case 'query.run':value={items:[row,{id:'speaker:'+qid,kind:'speaker',label:'Mapped person'}],edges:[{id:'edge',from:row.id,to:'speaker:'+qid,kind:'maps-to'}],total:2};break;
    case 'query.list':value={items:[...new Map(queries.map(q=>[q.query_id,q])).values()]};break;
    case 'query.show':value={items:queries.filter(q=>q.query_id===request.item_id)};break;
    case 'query.save':value={...request.data.query,query_id:request.item_id,revision:request.data.expected_revision+1,parameters:{},validations:[]};queries.push(value);break;
    case 'views.save':value={id:request.item_id,query_id:request.data.query_id,revision:3,layout:request.data.layout};bridge.layout=value;break;
    case 'views.show':value=bridge.layout;break;
    case 'graph.rebuild':value={fresh:true};break;
    case 'evidence.extract':value={work_id:qid};break;
   }return {workspace_id:wid,result:value};
  },
  SelectWorkspace:async()=>({}),Choose:async()=>'',Playback:async(...args)=>{calls.push({operation:'Playback',args});return {result:{url:'/media/source',mime_type:'audio/wav',timeline_offset_seconds:0}}},
  ClosePlayback:async(url)=>{calls.push({operation:'ClosePlayback',url});return {result:{}}},
  VerifyPlayback:async()=>({result:{valid:true}}),Credential:async()=>({}),CompleteSmoke:async()=>{}
 };return bridge;
}
async function mount(b){document.body.innerHTML='<div id="root"></div>';root=createRoot(document.getElementById('root'));await act(async()=>root.render(createElement(App,{bridge:b})));await tick();await click('Explore')}
async function unmount(){await act(async()=>root.unmount())}

const advanced={title:'Assisted query',definition:{mode:'normalized',operation:'evidence-traverse',filters:{entity_ids:['cue:one'],concept_ids:[qid],text:'fixture'},traversal:{direction:'in',max_depth:4,relationship_types:['cites']},order_by:[{field:'id',direction:'desc'}],pagination:{limit:17}}};
const configuration={enabled:true,adapter:'insonic-http',contract_version:'1',route:'local',endpoint:'http://127.0.0.1:9001/query',model:'fixture',mode:'suggest',limits:{max_rows:100}};
function assisted(){const b=mock(),operate=b.Operate;b.Operate=async r=>{const check=validateNativeRequest(r);assert.equal(check.valid,true,r.operation+': '+check.errors);if(r.operation==='query.assistance-show')return {workspace_id:wid,result:{configuration,revision:4}};if(r.operation==='query.assistance-set'){b.calls.push(r);return {workspace_id:wid,result:{configuration:r.data.configuration,revision:5}}}if(r.operation==='query.assist'){b.calls.push(r);return {workspace_id:wid,result:{proposal:advanced,explanation:'Fixture suggestion',validation:{status:'validated'},provenance:{mode:r.data.mode},...(r.data.mode==='auto-run'?{execution:{items:[row],total:1}}:{})}}}return operate(r)};return b}
test('assistance suggests without execution and adopts a complete editable saveable query',async()=>{const b=assisted();await mount(b);await click('Query');await fill('Assistance prompt','Find source evidence');await click('Suggest query');assert.match(document.body.textContent,/Fixture suggestion/);assert.equal(b.calls.filter(r=>r.operation==='query.run').length,0);await click('Load proposal into editor');await click('Run current query');assert.deepEqual(b.calls.filter(r=>r.operation==='query.run').at(-1).data,advanced);await click('Save query version');assert.deepEqual(b.calls.filter(r=>r.operation==='query.save').at(-1).data.query,advanced);assert.equal(b.calls.filter(r=>r.operation==='query.save').at(-1).data.expected_revision,0);await unmount()});
test('assistance settings use CAS and auto-run election is explicit',async()=>{const b=assisted();await mount(b);await click('Query');await fill('Assistance model','changed');await click('Save assistance configuration');assert.equal(b.calls.find(r=>r.operation==='query.assistance-set').data.expected_revision,4);await fill('Assistance prompt','List current media');await click('Assist and run query');assert.equal(b.calls.find(r=>r.operation==='query.assist').data.mode,'auto-run');assert.match(document.body.textContent,/Source statement/);await unmount()});
test('late assistance cannot overwrite editor changes or a changed prompt',async()=>{const b=assisted(),operate=b.Operate;let finish;b.Operate=async r=>r.operation==='query.assist'?new Promise(resolve=>{finish=resolve}):operate(r);await mount(b);await click('Query');await fill('Assistance prompt','First');await click('Suggest query');await fill('Words or terms','edited');await act(async()=>finish({workspace_id:wid,result:{proposal:advanced,explanation:'Obsolete assistance',validation:{status:'validated'}}}));await tick();assert.doesNotMatch(document.body.textContent,/Obsolete assistance/);await click('Suggest query');await fill('Assistance prompt','Changed');await act(async()=>finish({workspace_id:wid,result:{proposal:advanced,explanation:'Obsolete prompt',validation:{status:'validated'}}}));await tick();assert.doesNotMatch(document.body.textContent,/Obsolete prompt/);await unmount()});
test('provider failure leaves direct querying available',async()=>{const b=assisted(),operate=b.Operate;b.Operate=async r=>r.operation==='query.assist'?{error:{code:'unavailable',message:'Provider unavailable.'}}:operate(r);await mount(b);await click('Query');await fill('Assistance prompt','Test');await click('Suggest query');assert.match(document.querySelector('[role=alert]').textContent,/Provider unavailable/);await click('Run current query');assert.ok(b.calls.some(r=>r.operation==='query.run'));await unmount()});

test('a direct query owns results over an earlier pending assistant',async()=>{const b=assisted(),operate=b.Operate;let finish;b.Operate=async r=>r.operation==='query.assist'?new Promise(resolve=>{finish=resolve}):operate(r);await mount(b);await click('Query');await fill('Assistance prompt','First');await click('Assist and run query');await click('Run current query');await act(async()=>finish({workspace_id:wid,result:{proposal:advanced,explanation:'Obsolete auto-run',validation:{status:'validated'},execution:{items:[{id:'obsolete',label:'Obsolete assisted row'}],total:1}}}));await tick();assert.doesNotMatch(document.body.textContent,/Obsolete assisted row|Obsolete auto-run/);assert.match(document.body.textContent,/Source statement/);await unmount()});

test('native integer parameter survives proposal adoption and direct execution as exact decimal text',async()=>{const b=assisted(),operate=b.Operate;const q={title:'Exact native parameter',definition:{mode:'native',dialect:'ladybug-cypher',text:'RETURN $clock AS clock'},parameters:{clock:{type:'integer',value:'9223372036854775807'}}};b.Operate=async r=>r.operation==='query.assist'?{workspace_id:wid,result:{proposal:q,explanation:'Exact parameter',validation:{status:'validated'}}}:operate(r);await mount(b);await click('Query');await fill('Assistance prompt','Keep clock');await click('Suggest query');await click('Load proposal into editor');await click('Run current query');assert.deepEqual(b.calls.filter(r=>r.operation==='query.run').at(-1).data,q);await unmount()});
