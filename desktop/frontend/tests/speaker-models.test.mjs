// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import {JSDOM} from 'jsdom';
import {validateNativeRequest} from './contracts.mjs';
const dom=new JSDOM('<!doctype html><html><body><div id="root"></div></body></html>',{url:'http://wails.localhost'});
for(const name of ['window','document','HTMLElement','HTMLInputElement','HTMLTextAreaElement','HTMLSelectElement','Event','MouseEvent'])globalThis[name]=dom.window[name];
globalThis.IS_REACT_ACT_ENVIRONMENT=true;
const {act,createElement}=await import('react');
const {createRoot}=await import('react-dom/client');
const {Client}=await import('../.test-build/client.js');
const {SpeakerTraining,SpeakerModels,RosterMatching}=await import('../.test-build/speaker-models.js');
const {pipelinePayload,pipelineValues}=await import('../.test-build/forms.js');
const wid='aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',id='bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb',version='cccccccc-cccc-4ccc-8ccc-cccccccccccc';
let root;
const tick=async()=>{for(let i=0;i<3;i++)await act(()=>new Promise(resolve=>setTimeout(resolve,0)))};
function fixture(handler){const calls=[],errors=[];const client=new Client({Operate:async request=>{const check=validateNativeRequest(request);assert.equal(check.valid,true,check.errors);calls.push(request);return {workspace_id:wid,result:await handler(request)}}});client.select(wid);return {client,calls,errors,run:action=>{action().catch(error=>errors.push(error))}}}
async function mount(component,props){root=createRoot(document.getElementById('root'));await act(()=>root.render(createElement(component,props)));await tick()}
async function click(label){const button=[...document.querySelectorAll('button')].find(node=>node.textContent===label);assert.ok(button,label);await act(()=>button.dispatchEvent(new MouseEvent('click',{bubbles:true})));await tick()}
async function fill(label,value){const field=[...document.querySelectorAll('label')].find(node=>node.textContent.trim()===label);const node=field&&document.getElementById(field.htmlFor);assert.ok(node,label);const prototype=node.tagName==='TEXTAREA'?HTMLTextAreaElement.prototype:node.tagName==='SELECT'?HTMLSelectElement.prototype:HTMLInputElement.prototype;await act(()=>{Object.getOwnPropertyDescriptor(prototype,'value').set.call(node,value);node.dispatchEvent(new Event(node.tagName==='SELECT'?'change':'input',{bubbles:true}))});await tick()}
test.afterEach(async()=>{if(root){await act(()=>root.unmount());root=undefined}document.getElementById('root').innerHTML=''});

test('speaker training prepares corpus server-side and elects exact dataset',async()=>{
 const state=fixture(async request=>request.operation==='models.dataset.list'?{items:[]} : request.operation==='models.dataset.create'?{dataset:{id,state:'current'},summary:{references:201}}:{work_id:version,state:'pending'});
 await mount(SpeakerTraining,{...state,speakerID:id});await click('Prepare current speaker dataset');await click('Start elected training');
 const create=state.calls.find(call=>call.operation==='models.dataset.create');assert.equal(create.data.speaker_id,id);assert.deepEqual(create.data.recipe,{});
 const train=state.calls.find(call=>call.operation==='models.train');assert.equal(train.data.dataset_id,id);assert.equal(train.data.output_kind,'voice-embedding');assert.equal(train.data.adapter.mode,'local');assert.equal(state.calls.some(call=>call.operation==='speakers.select'),false);assert.deepEqual(state.errors,[]);
});

test('speaker training can elect a later dataset page',async()=>{
 const state=fixture(async request=>request.operation==='models.dataset.list'?(request.data.after_id?{items:[{dataset:{id:version,state:'current'}}]}:{items:[{dataset:{id,state:'current'}}],next_id:id}):request.operation==='pipelines.list'?{items:[]}:{work_id:id,state:'pending'});
 await mount(SpeakerTraining,{...state,speakerID:id});await click('More speaker datasets');await fill('Training dataset',version);await click('Start elected training');
 assert.equal(state.calls.find(call=>call.operation==='models.train').data.dataset_id,version);assert.deepEqual(state.errors,[]);
});
test('matching controls preserve matching-only request and unknown result',async()=>{
 const state=fixture(async request=>request.operation==='recordings.match'?{work_id:version,state:'pending'}:{id:version,state:'succeeded',result:{decisions:[{local_speaker_id:'voice',state:'unknown'}]}});
 await mount(RosterMatching,{...state,recordingID:id,roster:{revision:2}});await fill('Matching model reference',version);await click('Match voices against current roster');await click('Refresh matching result');
 const match=state.calls.find(call=>call.operation==='recordings.match');assert.equal(match.item_id,id);assert.equal(match.data.model_id,version);assert.equal(match.data.recording_id,undefined);assert.equal(match.data.model_digest,undefined);assert.equal(state.calls.some(call=>['recordings.process','recordings.assemble'].includes(call.operation)),false);assert.match(document.body.textContent,/unknown/);assert.deepEqual(state.errors,[]);
});
test('model controls fetch the exact selected version and CAS current profile',async()=>{
 const state=fixture(async request=>request.operation==='models.speaker.list'?{items:[{id:version,name:'Voice',kind:'embedding-profile',speaker_id:id}]}:request.operation==='models.speaker.show'?{output:{id:version,speaker_id:id},profile:{id,revision:4}}:{revision:5});
 await mount(SpeakerModels,state);await click('Inspect version');await fill('Model fetch destination','A:/model-output');await click('Fetch exact speaker version');await fill('Profile speaker identity',id);await fill('Expected profile revision','4');await click('Use exact version as speaker profile');
 assert.equal(state.calls.find(call=>call.operation==='models.speaker.fetch').item_id,version);const profile=state.calls.find(call=>call.operation==='models.profile.set');assert.equal(profile.item_id,id);assert.equal(profile.data.version_id,version);assert.equal(profile.data.expected_revision,4);assert.deepEqual(state.errors,[]);
});

test('finding reassociated outputs refreshes the selected identity profile revision',async()=>{
 const origin='dddddddd-dddd-4ddd-8ddd-dddddddddddd';
 const state=fixture(async request=>request.operation==='models.speaker.list'?{items:[{id:version,name:'Moved output',speaker_id:origin}],...(request.data.speaker_id?{profile:{id,revision:9}}:{})}:request.operation==='models.speaker.show'?{output:{id:version,speaker_id:origin},profile:{id:origin,revision:2}}:{revision:10});
 await mount(SpeakerModels,state);await fill('Speaker model identity filter',id);await click('Find speaker model versions');await click('Inspect version');await click('Use exact version as speaker profile');
 const profile=state.calls.find(call=>call.operation==='models.profile.set');assert.equal(profile.item_id,id);assert.equal(profile.data.expected_revision,9);assert.equal(profile.data.version_id,version);assert.deepEqual(state.errors,[]);
});

test('a late matching response cannot populate another recording selection',async()=>{
 let release;
 const state=fixture(async request=>request.operation==='recordings.match'?await new Promise(resolve=>{release=resolve}):{});
 await mount(RosterMatching,{...state,recordingID:id,roster:{revision:2}});await fill('Matching model reference',version);await click('Match voices against current roster');
 await act(()=>root.render(createElement(RosterMatching,{...state,recordingID:version,roster:{revision:3}})));await tick();
 await act(()=>release({work_id:id,state:'pending',phase:'old-recording-result'}));await tick();
 assert.doesNotMatch(document.body.textContent,/old-recording-result/);assert.deepEqual(state.errors,[]);
});

test('saved training-only configuration remains reusable and preserves exact revision',async()=>{
 const training={adapter:{id:'pyannote-profile',contract_version:'1',mode:'local',architecture:'pyannote',output_kinds:['voice-embedding'],consumers:['voice-matching']},base_model_id:version,output_kind:'voice-embedding',parameters:{}};
 const saved={id:version,name:'Elected profile',preset:'local',revision:4,configuration:{speaker_training:training}};
 const values=pipelineValues(saved);assert.equal(values.processing_enabled,false);const payload=pipelinePayload(values,saved);assert.deepEqual(payload.pipeline.configuration.speaker_training,training);assert.equal(payload.pipeline.configuration.recognition,undefined);
 const check=validateNativeRequest({operation:'pipelines.set',workspace_id:wid,item_id:version,data:payload});assert.equal(check.valid,true,check.errors);
 const state=fixture(async request=>request.operation==='models.dataset.list'?{items:[]}:request.operation==='pipelines.list'?{items:[saved]}:request.operation==='models.dataset.create'?{dataset:{id,state:'current'}}:{work_id:version,state:'pending'});
 await mount(SpeakerTraining,{...state,speakerID:id});await click('Prepare current speaker dataset');await fill('Saved training profile',version);await click('Start elected training');
 const train=state.calls.find(call=>call.operation==='models.train');assert.equal(train.data.pipeline_id,version);assert.equal(train.data.pipeline_revision,4);assert.equal(train.data.adapter,undefined);assert.equal(train.data.output_kind,undefined);assert.deepEqual(state.errors,[]);
});

for(const action of ['Refresh matching result','More matching outcomes'])test(`a late prior-job ${action} cannot replace the latest matching job`,async()=>{
 let release,count=0;
 const state=fixture(async request=>request.operation==='recordings.match'?(++count===1?{work_id:version,state:'succeeded',result:{decisions:[{local_speaker_id:'current-voice',state:'unknown'}]},next_ordinal:1}:{work_id:id,state:'pending',phase:'new-job-owner'}):await new Promise(resolve=>{release=resolve}));
 await mount(RosterMatching,{...state,recordingID:id,roster:{revision:2}});await fill('Matching model reference',version);await click('Match voices against current roster');await click(action);await click('Match voices against current roster');
 await act(()=>release({work_id:version,state:'succeeded',phase:'old-job-owner',result:{decisions:[{local_speaker_id:'stale-voice',state:'unknown'}]}}));await tick();
 assert.match(document.body.textContent,/new-job-owner/);assert.doesNotMatch(document.body.textContent,/old-job-owner|stale-voice/);assert.deepEqual(state.errors,[]);
});

test('editing a matching election discards its pending response',async()=>{
 let release;
 const state=fixture(async()=>await new Promise(resolve=>{release=resolve}));
 await mount(RosterMatching,{...state,recordingID:id,roster:{revision:2}});await fill('Matching model reference',version);await click('Match voices against current roster');await fill('Matching score threshold','0.8');
 await act(()=>release({work_id:version,state:'pending',phase:'obsolete-threshold'}));await tick();assert.doesNotMatch(document.body.textContent,/obsolete-threshold/);assert.deepEqual(state.errors,[]);
});

test('a newer training election owns the displayed job',async()=>{
 let release,count=0;
 const state=fixture(async request=>request.operation==='models.dataset.list'?{items:[]}:request.operation==='pipelines.list'?{items:[]}:request.operation==='models.dataset.create'?{dataset:{id,state:'current'}}:++count===1?await new Promise(resolve=>{release=resolve}):{work_id:id,state:'pending',phase:'current-training-owner'});
 await mount(SpeakerTraining,{...state,speakerID:id});await click('Prepare current speaker dataset');await click('Start elected training');await fill('Speaker model name','New election');await click('Start elected training');
 await act(()=>release({work_id:version,state:'pending',phase:'obsolete-training-owner'}));await tick();assert.match(document.body.textContent,/current-training-owner/);assert.doesNotMatch(document.body.textContent,/obsolete-training-owner/);assert.deepEqual(state.errors,[]);
});

test('late profile mutation responses cannot restore an older CAS revision',async()=>{
 let release,count=0;
 const state=fixture(async request=>request.operation==='models.speaker.list'?{items:[{id:version,name:'Voice',speaker_id:id}]}:request.operation==='models.speaker.show'?{output:{id:version,speaker_id:id},profile:{id,revision:4}}:++count===1?await new Promise(resolve=>{release=resolve}):{id,revision:6,state:'cleared'});
 await mount(SpeakerModels,state);await click('Inspect version');await click('Use exact version as speaker profile');await fill('Expected profile revision','5');await click('Clear current speaker profile');
 await act(()=>release({id,revision:5,state:'active',version_id:version}));await tick();
 const label=[...document.querySelectorAll('label')].find(node=>node.textContent.trim()==='Expected profile revision');assert.equal(document.getElementById(label.htmlFor).value,'6');assert.deepEqual(state.errors,[]);
});
