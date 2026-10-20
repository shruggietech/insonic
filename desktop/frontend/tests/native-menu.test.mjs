// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import {JSDOM} from 'jsdom';
import {bindNativeFileMenu} from '../.test-build/native-menu.js';

test('native File mnemonic uses the focused document Alt context and leaves other input alone',async()=>{
 const dom=new JSDOM('<input>');const doc=dom.window.document;const input=doc.querySelector('input');let opened=0;
 const unbind=bindNativeFileMenu(doc,async()=>{opened++},()=>assert.fail('unexpected menu failure'));
 const key=options=>{const event=new dom.window.KeyboardEvent('keydown',{key:'f',bubbles:true,cancelable:true,...options});input.dispatchEvent(event);return event;};
 assert.equal(key({altKey:true}).defaultPrevented,true);await Promise.resolve();assert.equal(opened,1);
 for(const options of [{},{altKey:true,ctrlKey:true},{altKey:true,metaKey:true},{altKey:true,repeat:true},{altKey:true,key:'x'}])assert.equal(key(options).defaultPrevented,false);
 await Promise.resolve();assert.equal(opened,1);unbind();assert.equal(key({altKey:true}).defaultPrevented,false);dom.window.close();
});

test('unsupported hosts keep native keyboard handling and rejected menu opens remain visible',async()=>{
 const dom=new JSDOM('<input>');const doc=dom.window.document;const errors=[];
 const event=()=>new dom.window.KeyboardEvent('keydown',{key:'f',altKey:true,bubbles:true,cancelable:true});
 const unsupported=bindNativeFileMenu(doc,undefined,message=>errors.push(message));const untouched=event();doc.dispatchEvent(untouched);assert.equal(untouched.defaultPrevented,false);unsupported();
 const unbind=bindNativeFileMenu(doc,async()=>{throw new Error('failure')},message=>errors.push(message));doc.dispatchEvent(event());await new Promise(resolve=>setImmediate(resolve));assert.equal(errors.length,1);unbind();dom.window.close();
});
