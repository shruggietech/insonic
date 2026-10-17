// SPDX-License-Identifier: Apache-2.0
import {readFile,mkdir,mkdtemp,rm} from 'node:fs/promises';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import assert from 'node:assert/strict';
import {chromium} from 'playwright-core';
const ROOT=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'..');

// This small renderer probe uses the real shared components and delivered CSS.
// Full application capture additionally exercises actual Go bridge workflows.
export async function probeControls(browserPath){
 const buildRoot=path.join(ROOT,'build');await mkdir(buildRoot,{recursive:true});
 const temporary=await mkdtemp(path.join(buildRoot,'density-probe-'));
 if(path.dirname(path.resolve(temporary))!==path.resolve(buildRoot))throw Error('Probe directory escaped build root');
 let browser;
 try{
  const {build}=await import('../desktop/frontend/node_modules/esbuild/lib/main.js');
  const bundle=await build({stdin:{contents:`import React,{useState} from 'react';import{createRoot}from'react-dom/client';import{Button,Check,Input,Table}from'./src/components';import{AppFrame}from'../../brand/kit/web/react/server';function Fixture(){const[value,setValue]=useState(false);return <AppFrame host="wails" layout="full-bleed" header={<header className="app-header"><strong>Desktop controls</strong></header>}><div className="desktop-content"><section><Button>Apply</Button><Check label="Compact selection" value={value} onChange={setValue}/><Input label="Readable field" value="Current selection" onChange={()=>{}}/><Table caption="Controlled records" heads={['Name']}><tr><td>Fixture</td></tr></Table></section></div></AppFrame>}createRoot(document.getElementById('root')).render(<Fixture/>);`,resolveDir:path.join(ROOT,'desktop/frontend'),sourcefile:'density-probe.tsx',loader:'tsx'},bundle:true,write:false,format:'iife',jsx:'automatic',nodePaths:[path.join(ROOT,'desktop/frontend/node_modules')],define:{'process.env.NODE_ENV':'"production"'}});
  let css=(await Promise.all(['brand/kit/tokens/interface.css','brand/kit/tokens/typography.css','brand/kit/tokens/spacing.css','brand/kit/web/components.css','desktop/frontend/src/app.css'].map(p=>readFile(path.join(ROOT,p),'utf8')))).join('\n');
  for(const name of ['Geist-Regular.woff2','SpaceGrotesk-Medium.woff2','GeistMono-Regular.woff2'])css=css.replaceAll('/assets/fonts/'+name,'data:font/woff2;base64,'+(await readFile(path.join(ROOT,'desktop/assets/fonts',name))).toString('base64'));
  browser=await chromium.launch({executablePath:browserPath,headless:true,timeout:20000,args:['--disable-background-networking','--disable-extensions','--no-first-run']});
  const context=await browser.newContext({viewport:{width:1200,height:800},reducedMotion:'reduce'});
  await context.route('**/*',route=>route.abort());
  const page=await context.newPage();await page.setContent('<!doctype html><html><head><meta charset="utf-8"><style>'+css+'</style></head><body><div id="root"></div><script>'+bundle.outputFiles[0].text+'</script></body></html>');
  await page.getByRole('button',{name:'Apply',exact:true}).waitFor();await page.evaluate(()=>document.fonts.ready);
  const input=page.getByLabel('Compact selection',{exact:true});const glyph=await input.boundingBox();
  assert.ok(glyph&&Math.abs(glyph.width-glyph.height)<=1,'selection glyph must be square: '+JSON.stringify(glyph));
  assert.ok(glyph.width<44,'visual glyph must be smaller than its activation target');
  const label=page.locator('label').filter({has:input});const target=await label.boundingBox();
  assert.ok(target&&target.width>=44&&target.height>=44,'compound label retains minimum activation target');
  await label.click({position:{x:target.width-3,y:target.height-3}});assert.equal(await input.isChecked(),true,'label padding activates the selection');
  await input.focus();await page.keyboard.press('Space');assert.equal(await input.isChecked(),false);
  assert.equal(await input.evaluate(node=>node.matches(':focus-visible')),true);
  const outline=await input.evaluate(node=>getComputedStyle(node).outlineWidth);assert.ok(parseFloat(outline)>=2);
  const {desktopDensityGeometry}=await import('../desktop/frontend/.test-build/density.js');
  await page.evaluate(desktopDensityGeometry);
  await input.evaluate(node=>{node.style.inlineSize='100%'});
  await assert.rejects(page.evaluate(desktopDensityGeometry),/glyph|target/);
  await input.evaluate(node=>{node.style.inlineSize=''});
  const action=await page.getByRole('button',{name:'Apply',exact:true}).boundingBox();assert.ok(action.width<300,'ordinary action must not stretch to its grid container');
  await page.setViewportSize({width:720,height:600});
  const narrow=await input.boundingBox();assert.ok(Math.abs(narrow.width-glyph.width)<=1&&Math.abs(narrow.height-glyph.height)<=1,'glyph does not expand with the container');
  return {state:'passed',glyph:{width:glyph.width,height:glyph.height},target:{width:target.width,height:target.height},action_width:action.width,keyboard:'passed',label_padding:'passed'};
 }finally{
  if(browser)await browser.close();
  if(path.dirname(path.resolve(temporary))!==path.resolve(buildRoot)||!path.basename(temporary).startsWith('density-probe-'))throw Error('Probe cleanup identity mismatch');
  await rm(temporary,{recursive:true,force:true});
 }
}
// This matrix observes the real mounted application through its normal controls.
export async function runDensityMatrix(page, context, {checkpoint, image}) {
  const {desktopDensityGeometry} = await import('../desktop/frontend/.test-build/density.js');
  const session = await context.newCDPSession(page);
  const oldFont = await page.evaluate(() => document.documentElement.style.fontSize);
  const observations = [];
  const normalText = await page.evaluate(() => parseFloat(getComputedStyle(document.documentElement).fontSize));
  const idle = async () => {
    await page.waitForFunction(() => ![...document.querySelectorAll('[role="status"]')].some(n => n.textContent.includes('Working')));
    await page.evaluate(() => document.fonts.ready);
    await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
    if (await page.getByRole('alert').count()) throw new Error('Density workflow reports an operation failure.');
  };
  try {
    for (const width of [1200, 720]) for (const scale of [1, 1.25, 1.5, 2]) for (const text of [1, 2]) {
      const height = width === 1200 ? 800 : 600;
      await page.setViewportSize({width, height});
      await session.send('Emulation.setDeviceMetricsOverride', {width, height, deviceScaleFactor: scale, mobile: false});
      await page.evaluate(value => { document.documentElement.style.fontSize = value + '%'; }, text * 100);
      for (const view of ['Setup', 'Library', 'Jobs', 'Pipelines', 'Models', 'Settings']) {
        await checkpoint('density.' + view, async () => {
          if (view === 'Setup') await page.getByRole('button', {name: 'Set up library', exact: true}).click();
          else if (view === 'Library') await page.getByRole('button', {name: 'Set up later', exact: true}).click();
          else await page.getByRole('navigation', {name: 'Workspace views'}).getByRole('button', {name: view, exact: true}).click();
          await page.getByRole('heading', {name: view, level: 1, exact: true}).waitFor();
          await idle();
          const observed = await page.evaluate(() => ({device_scale_factor: devicePixelRatio, width: innerWidth, height: innerHeight, text_pixels: parseFloat(getComputedStyle(document.documentElement).fontSize)}));
          if (observed.width !== width || observed.height !== height || Math.abs(observed.device_scale_factor - scale) > 0.01)
            throw new Error('Renderer did not apply the declared density scenario.');
          if (Math.abs(observed.text_pixels - normalText * text) > 0.1) throw new Error('Renderer text enlargement did not apply.');
          const geometry = await page.evaluate(desktopDensityGeometry);
          observations.push({view, width, height, device_scale_factor: scale, text_scale: text, observed, ...geometry});
          if (scale === 1 && ((width === 1200 && text === 1) || (width === 720 && text === 2 && view === 'Settings')))
            await image('desktop-' + view.toLowerCase() + (width === 720 ? '-enlarged' : '') + '.png', view,
              'Compact ' + view + ' controls at ' + width + ' by ' + height + (text === 2 ? ' with 200% text.' : '.'));
        }, 20000);
      }
    }
  } finally {
    await page.evaluate(value => { document.documentElement.style.fontSize = value; }, oldFont);
    await session.send('Emulation.clearDeviceMetricsOverride'); await session.detach();
  }
  if (observations.length !== 96) throw new Error('Density matrix is incomplete.');
  return {state: 'passed', scenarios: observations, os_display_scaling: 'not-inferred-from-renderer-scale'};
}

if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url)){
 if(process.argv[2]!=='--probe'||process.argv[3]!=='--browser'||!path.isAbsolute(process.argv[4]??'')||process.argv.length!==5)throw Error('Usage: desktop-density-capture.mjs --probe --browser ABSOLUTE_BROWSER');
 console.log(JSON.stringify(await probeControls(process.argv[4])));
}
