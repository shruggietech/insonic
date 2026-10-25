// SPDX-License-Identifier: Apache-2.0
// Disposable OS-association fixture. Never shipped in application packages.
import {readFile,writeFile,rename} from 'node:fs/promises';
import path from 'node:path';
import {pathToFileURL} from 'node:url';
import {createHash} from 'node:crypto';
import {chromium} from 'playwright-core';
import {milliseconds,withinAllowance} from './qualification-timing.mjs';

if(process.env.GITHUB_ACTIONS!=='true'||process.env.RUNNER_ENVIRONMENT!=='github-hosted'||process.env.INSONIC_DISPOSABLE_INSTALLER_TEST!=='1')throw new Error('Requires acknowledged disposable hosted CI.');
const [configuration,argument]=process.argv.slice(2);
const config=JSON.parse(await readFile(configuration,'utf8'));
const target=path.resolve(argument),relative=path.relative(config.directory,target);
if(!relative||relative==='..'||relative.startsWith('..'+path.sep)||path.isAbsolute(relative))throw new Error('Association target escaped its elected installation.');
const manifest=JSON.parse(await readFile(path.join(config.directory,'help/offline-manifest.json'),'utf8'));
if(manifest.version!==config.version||target!==path.resolve(config.directory,'help',manifest.entrypoint))throw new Error('Association opened different help identity.');
const browser=await chromium.launch({executablePath:config.browser,headless:true,timeout:milliseconds('frontend_seconds'),args:['--no-first-run','--disable-background-networking']});
try{
  const context=await browser.newContext({offline:true,serviceWorkers:'block'});
  const external=[],failed=[],errors=[];
  await context.route('**/*',route=>{if(/^https?:/i.test(route.request().url())){external.push(route.request().url());return route.abort();}return route.continue();});
  const page=await context.newPage();page.setDefaultTimeout(milliseconds('frontend_seconds'));page.setDefaultNavigationTimeout(milliseconds('frontend_seconds'));
  page.on('requestfailed',request=>failed.push(request.url()));page.on('pageerror',error=>errors.push(error.message));
  await page.goto(pathToFileURL(target).href,{waitUntil:'load'});
  await page.getByRole('heading',{level:1,name:'insonic v'+config.version,exact:true}).waitFor();
  await withinAllowance(()=>page.evaluate(()=>document.fonts.ready),milliseconds('frontend_seconds'));
  const rendered=await page.evaluate(()=>({styles:document.styleSheets.length,images:[...document.images].every(image=>image.complete&&image.naturalWidth>0),font:getComputedStyle(document.body).fontFamily}));
  if(!rendered.styles||!rendered.images||failed.length||errors.length||external.length)throw new Error('Installed help resources failed under ordinary offline file handling.');
  await page.getByRole('link',{name:'CLI invocation and results',exact:true}).first().click();
  await page.getByRole('heading',{level:1,name:'CLI invocation and results',exact:true}).waitFor();
  if(!page.url().startsWith(pathToFileURL(path.join(config.directory,'help')).href))throw new Error('Local help navigation escaped installed help.');
  await withinAllowance(()=>context.close(),milliseconds('cleanup_seconds'));
  config.result={state:'passed',version:manifest.version,entry_sha256:createHash('sha256').update(await readFile(target)).digest('hex'),offline:true,local_navigation:true,local_styles:rendered.styles,images:rendered.images,external_requests:external.length};
}finally{await withinAllowance(()=>browser.close(),milliseconds('natural_release_seconds'));}
// Publish only complete JSON. The ordinary UI actor observes existence concurrently.
const pending=config.output+'.pending-'+config.nonce;
await writeFile(pending,JSON.stringify({nonce:config.nonce,...config.result})+'\n',{encoding:'utf8',flag:'wx'});
await rename(pending,config.output);
