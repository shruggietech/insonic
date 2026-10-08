// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import {JSDOM} from 'jsdom';
import {validateNativeRequest} from './contracts.mjs';
const dom=new JSDOM('<!doctype html><html><body><div id="root"></div></body></html>',{url:'http://wails.localhost'});
for(const key of ['window','document','HTMLElement','HTMLInputElement','HTMLTextAreaElement','HTMLSelectElement','Event','MouseEvent'])globalThis[key]=dom.window[key];
globalThis.IS_REACT_ACT_ENVIRONMENT=true;window.matchMedia=()=>({matches:false,addEventListener(){},removeEventListener(){}});
const {act,createElement}=await import('react'),{createRoot}=await import('react-dom/client'),{App}=await import('../.test-build/App.js'),{forceStep}=await import('../.test-build/graph-view.js');
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
test('Explore reaches later calendar rows with keyboard viewport navigation and source-bound timeline playback',async()=>{
 const b=mock();await mount(b);await click('Load more calendar recordings');assert.match(document.body.textContent,/Undated recording/);
 const group=document.querySelector('[aria-label^="Recording calendar"]');await act(async()=>group.dispatchEvent(new dom.window.KeyboardEvent('keydown',{key:'ArrowRight',bubbles:true})));await tick();const cal=b.calls.filter(r=>r.operation==='timeline.calendar');assert.notEqual(cal.at(-1).data.filter.from,cal[0].data.filter.from);
 const entry=[...document.querySelectorAll('button')].find(x=>x.textContent.includes('First period'));await act(async()=>entry.click());await tick();assert.match(document.body.textContent,/Source-clock recording timeline/);await click('Play from source');assert.deepEqual(b.calls.find(r=>r.operation==='Playback').args,[mid,4,9,'5'.repeat(64)]);assert.equal(document.querySelector('audio').getAttribute('src'),'/media/source');await unmount();assert.ok(b.calls.some(r=>r.operation==='ClosePlayback'));assert.ok(!b.calls.some(r=>r.operation==='recordings.process'));
});
test('Explore graph and accessible table share nodes, save immutable definitions and separate layout with CAS',async()=>{
 const b=mock();await mount(b);await click('Graph');await click('Run current query');assert.ok(document.querySelector('svg[aria-label^="Evidence graph"]'));assert.match(document.body.textContent,/Accessible graph nodes/);await click('Save query version');const first=b.calls.find(r=>r.operation==='query.save');assert.equal(first.data.expected_revision,0);assert.equal(first.data.query.definition.operation,'graph-view');
 await click('Save query version');assert.equal(b.calls.filter(r=>r.operation==='query.save').at(-1).data.expected_revision,1);await click('Save graph layout');const layout=b.calls.find(r=>r.operation==='views.save');assert.equal(layout.data.query_id,first.item_id);assert.equal(layout.data.expected_revision,0);assert.ok(Object.keys(layout.data.layout.positions).length>0);assert.ok(!JSON.stringify(layout.data).includes('Source statement'));await click('Expand relationships');assert.deepEqual(b.calls.filter(r=>r.operation==='query.run').at(-1).data.definition.filters.entity_ids,[row.id]);await unmount();
});
test('editing a query discards delayed results and native incompatibility leaves saved definitions available',async()=>{
 const b=mock(),operate=b.Operate;let finish; b.Operate=async r=>{if(r.operation==='query.run')return new Promise(resolve=>{finish=resolve});return operate(r)};
 await mount(b);await click('Query');await click('Run current query');await fill('Words or terms','changed');await act(async()=>finish({workspace_id:wid,result:{items:[{id:'late',label:'Obsolete response'}],total:1}}));await tick();assert.doesNotMatch(document.body.textContent,/Obsolete response/);
 b.Operate=async r=>{if(r.operation==='query.run')return {error:{code:'unsupported_capability',message:'The selected backend does not support this dialect.'}};return operate(r)};
 await fill('Query mode','native');await fill('Native query dialect','arcade-sql');await fill('Native read query','SELECT entity_id FROM Entity LIMIT :limit');await fill('Typed parameters JSON','{"limit":{"type":"integer","value":10}}');await click('Save query version');await click('Run current query');assert.match(document.querySelector('[role=alert]').textContent,/does not support this dialect/);assert.ok(button('Save query version'));await unmount();
});
test('force layout remains finite and deterministic for isolated, linked and coincident nodes',()=>{
 const ids=['a','b','c'],edges=[{from:'a',to:'b'}],initial={a:{x:400,y:240},b:{x:400,y:240}};
 assert.deepEqual(forceStep(ids,edges,initial,0),forceStep(ids,edges,initial,0));let p=initial;for(let i=0;i<180;i++)p=forceStep(ids,edges,p,i);for(const value of Object.values(p)){assert.ok(Number.isFinite(value.x)&&Number.isFinite(value.y));assert.ok(value.x>=30&&value.x<=770)}
});

test('late source ticket is closed after selection changes',async()=>{
 const b=mock();let resolve;b.Playback=async()=>new Promise(r=>{resolve=r});await mount(b);
 const first=[...document.querySelectorAll('button')].find(x=>x.textContent.includes('First period'));await act(async()=>first.click());await tick();
 await act(async()=>button('Play from source').click());await tick();
 const second=[...document.querySelectorAll('button')].find(x=>x.textContent.includes('Another period'));await act(async()=>second.click());await tick();
 await act(async()=>resolve({result:{url:'/media/stale',mime_type:'audio/wav'}}));await tick();
 assert.equal(document.querySelector('audio'),null);assert.ok(b.calls.some(c=>c.operation==='ClosePlayback'&&c.url==='/media/stale'));await unmount();
});
