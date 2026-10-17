// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import {JSDOM} from 'jsdom';
import {bindHelpNavigation} from '../.test-build/help.js';

test('offline frame keeps local navigation and delegates external references without navigating itself',()=>{
  const dom=new JSDOM('<a href="cli.html" target="_blank">Local</a><a href="https://example.org/reference">Reference</a><a href="file:///outside.html">Invalid</a>',{url:'http://wails.localhost/assets/help/docs/index.html'});
  const opened=[],rejected=[];
  const unbind=bindHelpNavigation(dom.window.document,url=>opened.push(url),()=>rejected.push(true));
  const [local,external,invalid]=dom.window.document.querySelectorAll('a');
  const click=element=>{const event=new dom.window.MouseEvent('click',{bubbles:true,cancelable:true});element.dispatchEvent(event);return event;};
  assert.equal(click(local).defaultPrevented,false);assert.equal(local.target,'_self');
  assert.equal(click(external).defaultPrevented,true);assert.deepEqual(opened,['https://example.org/reference']);
  assert.equal(click(invalid).defaultPrevented,true);assert.equal(rejected.length,1);
  assert.equal(dom.window.location.href,'http://wails.localhost/assets/help/docs/index.html');
  unbind();dom.window.close();
});
