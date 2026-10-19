// SPDX-License-Identifier: Apache-2.0
export function bindNativeFileMenu(doc:Document,open:(()=>Promise<void>)|undefined,onError:(message:string)=>void):()=>void {
 if(!open)return()=>{};
 let active=true;
 const handler=(event:KeyboardEvent)=>{
  if(!event.altKey||event.ctrlKey||event.metaKey||event.repeat||event.key.toLowerCase()!=='f')return;
  event.preventDefault();
  void Promise.resolve().then(open).catch(()=>{if(active)onError('The native File menu could not be opened.');});
 };
 doc.addEventListener('keydown',handler,true);
 return()=>{active=false;doc.removeEventListener('keydown',handler,true);};
}
