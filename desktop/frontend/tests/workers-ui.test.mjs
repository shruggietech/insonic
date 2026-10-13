// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import {JSDOM} from 'jsdom';
import {validateNativeRequest} from './contracts.mjs';
const dom=new JSDOM('<!doctype html><html><body><div id="root"></div></body></html>',{url:'http://wails.localhost'});
for(const key of ['window','document','HTMLElement','HTMLInputElement','HTMLSelectElement','Event','MouseEvent'])globalThis[key]=dom.window[key];
globalThis.IS_REACT_ACT_ENVIRONMENT=true;
const {act,createElement}=await import('react');
const {createRoot}=await import('react-dom/client');
const {Workers}=await import('../.test-build/workers.js');
const {Client}=await import('../.test-build/client.js');
const wid='aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa';
function field(name){const label=[...document.querySelectorAll('label')].find(l=>l.textContent===name);assert.ok(label,name);return document.getElementById(label.htmlFor)}
async function fill(name,value){let control=field(name);await act(async()=>{Object.getOwnPropertyDescriptor(control.tagName==='SELECT'?HTMLSelectElement.prototype:HTMLInputElement.prototype,'value').set.call(control,value);control.dispatchEvent(new Event('input',{bubbles:true}));control.dispatchEvent(new Event('change',{bubbles:true}))})}
function button(name){const b=[...document.querySelectorAll('button')].find(b=>b.textContent===name);assert.ok(b,name);return b}
test('rendered worker/model setup uses protected credentials and durable work controls',async()=>{
 const calls=[],errors=[];let release;
 const client=new Client({
  Credential:async(args,input)=>{calls.push({operation:'protected',args,input});return {result:{}}},
  Operate:async(req)=>{
   assert.equal(validateNativeRequest(req).valid,true,req.operation);
   calls.push(req);
   let result={};
   if(req.operation==='workers.catalog')result={platform:'windows_amd64',catalog:{profiles:[]}};
   if(req.operation==='models.defaults')result=[];
   if(req.operation==='workers.plan'){await new Promise(resolve=>{release=resolve});result={destination:'owned',space_sufficient:true,election:{...req.data,expected_revision:'a'.repeat(64),profile_digest:'b'.repeat(64)}}}
   if(req.operation==='workers.setup'||req.operation==='work.cancel')result={work_id:wid,state:req.operation==='work.cancel'?'cancelled':'pending'};
   if(req.operation==='models.prepare')result={model_id:wid,scratch_space_sufficient:true,...(req.data.acquire?{work:{work_id:wid,state:'pending'}}:{})};
   return {workspace_id:wid,result};
  }
 });
 client.select(wid);
 const root=createRoot(document.getElementById('root'));
 const run=async(fn)=>{try{await fn()}catch(e){errors.push(e)}};
 try{
  await act(async()=>root.render(createElement(Workers,{client,run})));
  assert.equal(button('Prepare verified environment').disabled,true);
  await act(async()=>button('Preview environment setup').click());
  assert.equal(field('Processing role').disabled,true);
  await act(async()=>release());
  await act(async()=>button('Prepare verified environment').click());
  assert.equal(calls.find(c=>c.operation==='workers.setup').data.expected_revision,'a'.repeat(64));
  await act(async()=>button('Cancel setup work').click());
  assert.ok(calls.find(c=>c.operation==='work.cancel'));
  await fill('Exact model bundle','community-1');
  await fill('New Hugging Face read token (optional)','private-fixture-token');
  await act(async()=>button('Preview model acquisition').click());
  await act(async()=>button('Acquire or reuse exact model').click());
  const protectedCall=calls.find(c=>c.operation==='protected');
  assert.equal(JSON.parse(protectedCall.input).value,'private-fixture-token');
  const acquisition=calls.find(c=>c.operation==='models.prepare'&&c.data.acquire);
  assert.equal(acquisition.data.credential_id,protectedCall.args[1]);
  assert.equal(JSON.stringify(calls.filter(c=>c.operation!=='protected')).includes('private-fixture-token'),false);
  assert.equal(field('New Hugging Face read token (optional)').value,'');
  assert.deepEqual(errors,[]);
 }finally{await act(async()=>root.unmount())}
});
