// SPDX-License-Identifier: Apache-2.0
import {useEffect,useRef,useState} from 'react';
import {Actions,Button,Check,Input,Table} from './components';
import type {Obj} from './client';
export type Positions=Record<string,{x:number;y:number}>;
// Deterministic initial positions, bounded forces and finite cooling prevent
// backend row order or reduced-motion preferences from changing evidence.
export function forceStep(ids:string[],edges:Obj[],positions:Positions,iteration:number):Positions {
 const out:Positions={};const n=Math.max(ids.length,1),cool=Math.max(.02,1-iteration/160);
 for(let i=0;i<ids.length;i++){
  const id=ids[i],p=positions[id]??{x:400+240*Math.cos(i*2*Math.PI/n),y:240+180*Math.sin(i*2*Math.PI/n)};
  let dx=(400-p.x)*.002,dy=(240-p.y)*.002;
  for(const other of ids){if(other===id)continue;const q=positions[other];if(!q)continue;const x=p.x-q.x,y=p.y-q.y,d=Math.max(x*x+y*y,25);dx+=x*80/d;dy+=y*80/d}
  for(const e of edges){const other=e.from===id?e.to:e.to===id?e.from:'';const q=positions[other];if(!q)continue;dx+=(q.x-p.x)*.004;dy+=(q.y-p.y)*.004}
  out[id]={x:Math.max(30,Math.min(770,p.x+Math.max(-6,Math.min(6,dx))*cool)),y:Math.max(25,Math.min(455,p.y+Math.max(-6,Math.min(6,dy))*cool))};
 }return out;
}
export function GraphView({rows,edges,select,expand,positions,setPositions,paused,setPaused,filter,setFilter}:{
 filter:string;setFilter:(v:string)=>void;rows:Obj[];edges:Obj[];select:(r:Obj)=>void;expand:(r:Obj)=>void;positions:Positions;setPositions:(p:Positions)=>void;paused:boolean;setPaused:(v:boolean)=>void;
}){
 const [zoom,setZoom]=useState(1),[pan,setPan]=useState({x:0,y:0});
 const current=useRef(positions),iteration=useRef(0),drag=useRef<{id:string;x:number;y:number}|undefined>(undefined);
 current.current=positions;
 const visible=rows.filter(r=>String(r.label).toLowerCase().includes(filter.toLowerCase())).slice(0,200),ids=visible.map(r=>String(r.id)),idKey=ids.join('|');
 const links=edges.filter(e=>ids.includes(e.from)&&ids.includes(e.to));
 useEffect(()=>{iteration.current=0;setPositions(forceStep(ids,links,current.current,0))},[idKey]);
 useEffect(()=>{
  const reduced=document.documentElement.dataset.reducedMotion==='true'||(window.matchMedia?.('(prefers-reduced-motion: reduce)').matches??false);
  if(paused||reduced||ids.length===0)return;
  const timer=setInterval(()=>{if(iteration.current>=160){clearInterval(timer);return;}setPositions(forceStep(ids,links,current.current,++iteration.current))},40);
  return()=>clearInterval(timer);
 },[idKey,paused]);
 return <>
  <Actions><Input label="Graph label filter" value={filter} onChange={setFilter}/><Check label="Pause graph motion" value={paused} onChange={setPaused}/>
   <Button onClick={()=>setZoom(z=>Math.min(4,z*1.2))}>Zoom graph in</Button><Button onClick={()=>setZoom(z=>Math.max(.25,z/1.2))}>Zoom graph out</Button><Button onClick={()=>{setZoom(1);setPan({x:0,y:0})}}>Reset graph view</Button></Actions>
  <p>{visible.length} visible nodes, {rows.length-visible.length} hidden by label filtering or the 200-node drawing bound. The result table includes every returned node.</p>
  <svg className="evidence-graph" viewBox="0 0 800 480" tabIndex={0} role="group" aria-label="Evidence graph. Arrow keys pan, plus and minus zoom."
   onKeyDown={e=>{const delta=25;switch(e.key){case 'ArrowLeft':setPan(p=>({...p,x:p.x+delta}));break;case 'ArrowRight':setPan(p=>({...p,x:p.x-delta}));break;case 'ArrowUp':setPan(p=>({...p,y:p.y+delta}));break;case 'ArrowDown':setPan(p=>({...p,y:p.y-delta}));break;case '+':setZoom(z=>Math.min(4,z*1.2));break;case '-':setZoom(z=>Math.max(.25,z/1.2));break;default:return}e.preventDefault()}}
   onPointerMove={e=>{const d=drag.current;if(!d)return;const box=e.currentTarget.getBoundingClientRect();if(!box.width)return;const dx=(e.clientX-d.x)*800/box.width/zoom,dy=(e.clientY-d.y)*480/box.height/zoom;drag.current={...d,x:e.clientX,y:e.clientY};if(d.id==='pan'){setPan(p=>({x:p.x+dx*zoom,y:p.y+dy*zoom}))}else{const p=current.current[d.id];if(p)setPositions({...current.current,[d.id]:{x:p.x+dx,y:p.y+dy}})}}}
   onPointerDown={e=>{if(e.target!==e.currentTarget)return;e.currentTarget.setPointerCapture(e.pointerId);drag.current={id:'pan',x:e.clientX,y:e.clientY}}}
   onPointerUp={()=>{drag.current=undefined}} onPointerCancel={()=>{drag.current=undefined}}>
   <g transform={`translate(${pan.x} ${pan.y}) scale(${zoom})`}>
    {links.map(e=>{const a=positions[e.from],b=positions[e.to];if(!a||!b)return null;return <g key={e.id}><line x1={a.x} y1={a.y} x2={b.x} y2={b.y}/><text x={(a.x+b.x)/2} y={(a.y+b.y)/2} className="edge-label">{e.kind}</text></g>})}
    {visible.map(r=>{const p=positions[r.id];if(!p)return null;return <g key={r.id} transform={`translate(${p.x} ${p.y})`} tabIndex={0} role="button" aria-label={`${r.kind}: ${r.label}`} onClick={()=>select(r)} onKeyDown={e=>{if(e.key==='Enter'||e.key===' '){e.preventDefault();select(r)}}} onPointerDown={e=>{e.stopPropagation();e.currentTarget.setPointerCapture(e.pointerId);drag.current={id:r.id,x:e.clientX,y:e.clientY};setPaused(true)}}><title>{r.label}</title><circle r="9"/><text x="13" y="4">{String(r.label).slice(0,55)}</text></g>})}
   </g>
  </svg>
  <Table caption="Accessible graph nodes" heads={['Kind','Source label','Actions']}>{rows.map(r=><tr key={r.id}><td>{r.kind}</td><td>{r.label}</td><td><Button onClick={()=>select(r)}>Inspect source</Button><Button onClick={()=>expand(r)}>Expand relationships</Button></td></tr>)}</Table>
 </>;
}
