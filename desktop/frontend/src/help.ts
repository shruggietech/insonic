// SPDX-License-Identifier: Apache-2.0
// Offline documents own local navigation. Explicit web references use the native
// system-browser dispatcher, never an iframe or private-origin popup.
export function bindHelpNavigation(doc:Document,openReference:(url:string)=>void,rejected:()=>void):()=>void {
  const base=new URL(doc.baseURI);
  const activate=(event:Event)=>{
    const target=event.target as Element|null;
    const anchor=target?.closest?.('a[href]') as HTMLAnchorElement|null;
    if(!anchor)return;
    let url:URL;
    try{url=new URL(anchor.getAttribute('href')!,doc.baseURI);}
    catch{event.preventDefault();event.stopImmediatePropagation();rejected();return;}
    const local=url.protocol===base.protocol&&url.host===base.host&&url.pathname.startsWith('/assets/help/');
    if(local){anchor.target='_self';return;}
    event.preventDefault();event.stopImmediatePropagation();
    const mouse=event as MouseEvent;
    if(event.type==='auxclick'&&mouse.button!==1)return;
    if((url.protocol==='https:'||url.protocol==='http:')&&!['wails.localhost','wails'].includes(url.hostname.replace(/\.$/,'')))openReference(url.href);
    else rejected();
  };
  doc.addEventListener('click',activate,true);doc.addEventListener('auxclick',activate,true);
  return()=>{doc.removeEventListener('click',activate,true);doc.removeEventListener('auxclick',activate,true);};
}
