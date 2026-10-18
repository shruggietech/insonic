// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import {JSDOM} from 'jsdom';
const dom=new JSDOM('<!doctype html><html><body><div id="root"></div></body></html>',{url:'http://wails.localhost'});
for(const key of ['window','document','HTMLElement','HTMLInputElement','Event','MouseEvent'])globalThis[key]=dom.window[key];
globalThis.IS_REACT_ACT_ENVIRONMENT=true;
const {act,createElement,useState}=await import('react');
const {createRoot}=await import('react-dom/client');
const {Check}=await import('../.test-build/components.js');
test('selection is one associated native label target and preserves controlled state',async()=>{
 const root=createRoot(document.getElementById('root'));
 function Fixture(){const [value,setValue]=useState(false);return createElement(Check,{label:'Retain current recording',value,onChange:setValue});}
 try{
  await act(()=>root.render(createElement(Fixture)));
  const input=document.querySelector('input'),label=document.querySelector('label');
  assert.equal(input.type,'checkbox');assert.equal(label.control,input);
  assert.ok(label.contains(input),'the complete input/label target must be clickable');
  await act(()=>label.click());assert.equal(input.checked,true);
  await act(()=>label.click());assert.equal(input.checked,false);
  assert.equal(document.querySelectorAll('input').length,1);
 }finally{await act(()=>root.unmount());}
});
