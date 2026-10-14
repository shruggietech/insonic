// SPDX-License-Identifier: Apache-2.0
// Maintainer capture tooling. The production frontend and Go desktop bridge stay unchanged.
import {readFile,writeFile,readdir,mkdir,mkdtemp,rm,stat} from 'node:fs/promises';
import path from 'node:path';
import os from 'node:os';
import http from 'node:http';
import {spawn} from 'node:child_process';
import {createHash,randomBytes} from 'node:crypto';
import {fileURLToPath} from 'node:url';

export const ROOT=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'..');
export const VERSION=JSON.parse(await readFile(path.join(ROOT,'package.json'),'utf8')).version;
export const PLAYWRIGHT_VERSION='1.62.1';
const sha=bytes=>createHash('sha256').update(bytes).digest('hex');

export function fixtureEnvironment(base,fixture,cli) {
  // No owner profile, credential/provider variables or network routes are inherited.
  const out={};
  for(const name of ['SystemRoot','WINDIR','SystemDrive','COMSPEC'])if(base[name])out[name]=base[name];
  out.PATH=[path.dirname(cli),base.SystemRoot?path.join(base.SystemRoot,'System32'):'',...(base.INSONIC_CAPTURE_NATIVE_PATH??'').split(path.delimiter)].filter(Boolean).join(path.delimiter);
  Object.assign(out,{TEMP:fixture,TMP:fixture,HOME:path.join(fixture,'profile'),USERPROFILE:path.join(fixture,'profile'),APPDATA:path.join(fixture,'profile','Roaming'),LOCALAPPDATA:path.join(fixture,'profile','Local'),XDG_CONFIG_HOME:path.join(fixture,'profile','Config')});
  return out;
}

export function parseOptions(args) {
  const out={output:path.join(ROOT,'docs','screenshots','v'+VERSION)};
  for(let i=0;i<args.length;i++){
    const key={'--cli':'cli','--bridge':'bridge','--browser':'browser','--output':'output'}[args[i]];
    if(!key||!args[i+1]||args[i+1].startsWith('--')||out[key]&&key!=='output')throw Error('Use --cli FILE --bridge FILE [--browser FILE] [--output DIRECTORY].');
    out[key]=path.resolve(args[++i]);
  }
  if(!out.cli||!out.bridge)throw Error('A current CLI and current capture bridge build are required.');
  return out;
}

export function allowedAsset(url,assets) {
  return typeof url==='string'&&!url.includes('?')&&assets.has(url);
}

async function run(executable,args,options={}) {
  return await new Promise((resolve,reject)=>{
    const child=spawn(executable,args,{cwd:ROOT,windowsHide:true,stdio:['pipe','pipe','pipe'],...options});
    let stdout='',stderr='';
    child.stdout.on('data',d=>{stdout+=d;if(stdout.length>8<<20)child.kill()});
    child.stderr.on('data',d=>{stderr+=d;if(stderr.length>8<<20)child.kill()});
    const timer=setTimeout(()=>child.kill(),90000);
    child.once('error',reject);child.once('exit',code=>{clearTimeout(timer);code===0?resolve(stdout):reject(Error(`Capture command failed (${code}): ${stderr.slice(-1000)}`))});child.stdin.end();
  });
}

async function sourceSnapshot() {
  const revision=(await run('git',['rev-parse','HEAD'])).trim();
  const files=(await run('git',['ls-files','--cached','--others','--exclude-standard','-z','desktop','cmd/insonic','cmd/insonic-capture-bridge','internal','schemas','brand','go.mod','go.sum','package.json','package-lock.json','scripts/capture-application-ui.mjs'])).split('\0').filter(Boolean).sort();
  const identities=[];
  for(const name of files)identities.push({path:name.replaceAll('\\','/'),sha256:sha(await readFile(path.join(ROOT,name)))});
  const dirty=!!(await run('git',['status','--porcelain','--untracked-files=no'])).trim();
  return {revision,dirty,tree_sha256:sha(JSON.stringify(identities)),files:identities};
}

function embeddedRevision(bytes) {
  const marker=Buffer.from('build\tvcs.revision=');const at=bytes.indexOf(marker);
  if(at<0)throw Error('Capture binaries must include Go VCS build information.');
  const value=bytes.subarray(at+marker.length,at+marker.length+40).toString('ascii');
  if(!/^[a-f0-9]{40}$/.test(value))throw Error('Capture binary source revision is malformed.');
  return value;
}

export function assertReceipt(receipt) {
  if(receipt.kind!=='application-ui-capture'||receipt.schema_version!==VERSION||receipt.renderer.kind!=='chromium-headless'||receipt.renderer.playwright_version!==PLAYWRIGHT_VERSION||!receipt.fixture.isolated||receipt.fixture.private_inputs!==false||!/^[a-f0-9]{40}$/.test(receipt.source.revision)||!/^[a-f0-9]{64}$/.test(receipt.source.tree_sha256)||receipt.images.length!==4)throw Error('Incomplete capture provenance.');
  const expected=['setup-location.png','setup-backends.png','setup-processing.png','library-ready.png'];
  for(const [index,image] of receipt.images.entries())if(image.path!==expected[index]||!/^[a-f0-9]{64}$/.test(image.sha256)||!Number.isInteger(image.width)||image.width<1||image.width>1440||!Number.isInteger(image.height)||image.height<1||image.height>6000||!image.landmark||!image.caption)throw Error('Incomplete image receipt.');
  return receipt;
}

export async function capture(options) {
  const source=await sourceSnapshot();
  const cliBytes=await readFile(options.cli),bridgeBytes=await readFile(options.bridge);
  if(embeddedRevision(cliBytes)!==source.revision||embeddedRevision(bridgeBytes)!==source.revision)throw Error('Capture binary source revision does not match the checkout. Rebuild both binaries.');
  await run(process.execPath,[path.join(ROOT,'desktop/frontend/build.mjs')]);
  const fixtureBase=os.platform()==='win32'?path.parse(os.tmpdir()).root:os.tmpdir();
  const fixture=await mkdtemp(path.join(fixtureBase,'insonic-capture-'));
  const library=path.join(fixture,'My voice archive');
  const fixtureID=randomBytes(16).toString('hex');
  await writeFile(path.join(fixture,'capture-owner.json'),JSON.stringify({fixture_id:fixtureID,kind:'insonic-ui-capture'}));
  const env=fixtureEnvironment(process.env,fixture,options.cli);
  for(const dir of [env.APPDATA,env.LOCALAPPDATA,env.XDG_CONFIG_HOME])await mkdir(dir,{recursive:true});
  let helper,browser,server,endpoint,helperToken,exitPromise,helperStderr='';
  try {
    helper=spawn(options.bridge,['--library',library,'--cli',options.cli],{cwd:fixture,env,windowsHide:true,stdio:['pipe','pipe','pipe']});helper.stdin.end();
    exitPromise=new Promise((resolve,reject)=>{helper.once('error',reject);helper.once('exit',code=>resolve(code))});
    helper.stderr.on('data',d=>helperStderr+=d);
    const ready=await new Promise((resolve,reject)=>{let text='';const timeout=setTimeout(()=>reject(Error('Capture bridge startup timeout.')),25000);helper.stdout.on('data',d=>{text+=d;if(text.includes('\n')){clearTimeout(timeout);try{resolve(JSON.parse(text.split('\n')[0]))}catch(e){reject(e)}}});helper.once('exit',()=>{clearTimeout(timeout);reject(Error('Capture bridge failed before readiness.'))})});
    ({endpoint,token:helperToken}=ready);if(ready.version!==VERSION||!/^http:\/\/127\.0\.0\.1:\d+$/.test(endpoint))throw Error('Capture bridge identity mismatch.');
    const backend=async(route,body)=>{const response=await fetch(endpoint+route,{method:body===undefined?'GET':'POST',headers:{'X-Capture-Token':helperToken,Origin:endpoint,'Content-Type':'application/json'},...(body===undefined?{}:{body:JSON.stringify(body)}),signal:AbortSignal.timeout(70000)});if(!response.ok)throw Error(`Capture bridge rejected request (${response.status}).`);return response.status===204?null:response.json()};
    const assets=new Map();
    for(const name of ['app.js','app.css','product.css'])assets.set('/assets/bundle/'+name,await readFile(path.join(ROOT,'desktop/assets/bundle',name)));
    const fonts=await readdir(path.join(ROOT,'desktop/assets/bundle/fonts')).catch(error=>{if(error.code==='ENOENT')return [];throw error});
    for(const name of fonts)assets.set('/assets/bundle/fonts/'+name,await readFile(path.join(ROOT,'desktop/assets/bundle/fonts',name)));
    for(const name of await readdir(path.join(ROOT,'desktop/assets/fonts')))assets.set('/assets/fonts/'+name,await readFile(path.join(ROOT,'desktop/assets/fonts',name)));
    const token=randomBytes(32).toString('hex');let origin;const transportErrors=[];
    server=http.createServer(async(req,res)=>{try{
      if(req.headers['x-capture-token']!==token){res.writeHead(403);res.end();return}
      if(req.url==='/bridge'&&req.method==='POST'){
        if(req.headers.origin!==origin){res.writeHead(403);res.end();return}
        let body='';for await(const chunk of req){body+=chunk;if(Buffer.byteLength(body)>128<<10)throw Error('capture request limit')}
        res.setHeader('Content-Type','application/json');res.end(JSON.stringify(await backend('/bridge',JSON.parse(body))));return;
      }
      if(req.method!=='GET'){res.writeHead(405);res.end();return}
      if(req.url==='/'){
        const native=['Show','SetupDraft','BackendSetup','Operate','SelectWorkspace','Choose'];
        const script=`<script>const rpc=(method,args)=>fetch('/bridge',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({method,args})}).then(async r=>{if(!r.ok)throw Error('Capture bridge unavailable');return r.json()});window.go={desktop:{Bridge:Object.fromEntries(${JSON.stringify(native)}.map(method=>[method,(...args)=>rpc(method,args)]))}};</script>`;
        const html=(await readFile(path.join(ROOT,'desktop/index.html'),'utf8')).replace('<script defer',script+'<script defer');res.setHeader('Content-Type','text/html');res.end(html);return;
      }
      if(req.url==='/favicon.ico'){res.writeHead(204);res.end();return}
      if(!allowedAsset(req.url,assets)){res.writeHead(404);res.end();return}
      res.setHeader('Content-Type',req.url.endsWith('.css')?'text/css':req.url.endsWith('.js')?'application/javascript':'font/woff2');res.end(assets.get(req.url));
    }catch(e){transportErrors.push(String(e));res.writeHead(500);res.end('Capture transport failed.')}});
    await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));origin='http://127.0.0.1:'+server.address().port;
    const {chromium}=await import('playwright-core');
    const actualVersion=JSON.parse(await readFile(path.join(ROOT,'node_modules/playwright-core/package.json'),'utf8')).version;
    if(actualVersion!==PLAYWRIGHT_VERSION)throw Error('Playwright toolchain version mismatch.');
    browser=await chromium.launch({headless:true,env,...(options.browser?{executablePath:options.browser}:{}),args:['--disable-background-networking','--disable-extensions','--no-first-run','--no-default-browser-check']});
    const viewport={width:1440,height:1100,device_scale_factor:1};
    const context=await browser.newContext({viewport:{width:viewport.width,height:viewport.height},deviceScaleFactor:1,locale:'en-US',colorScheme:'light',reducedMotion:'reduce',serviceWorkers:'block',extraHTTPHeaders:{'X-Capture-Token':token}});
    const page=await context.newPage();page.setDefaultTimeout(25000);
    const errors=[],blocked=[];page.on('pageerror',e=>errors.push(String(e)));
    page.on('response',response=>{if(response.status()>=400)errors.push('Resource '+new URL(response.url()).pathname+' failed ('+response.status()+').')});
    page.on('requestfailed',request=>errors.push('Resource '+new URL(request.url()).pathname+' did not load.'));
    await context.route('**/*',route=>{if(new URL(route.request().url()).origin!==origin){blocked.push(route.request().url());return route.abort()}return route.continue()});
    const pending=path.join(fixture,'images');await mkdir(pending);
    const images=[];
    const screenshot=async(name,landmark,caption)=>{
      try{await page.getByRole('heading',{name:landmark,exact:true,...(landmark==='Library'?{level:1}:{})}).waitFor();}
      catch{throw Error(`Expected UI landmark was absent: ${landmark}. ${await page.getByRole('alert').allTextContents()}. ${helperStderr.slice(-1000)}`)}
      await page.waitForFunction(()=>![...document.querySelectorAll('[role="status"]')].some(n=>n.textContent.includes('Working')));
      if(await page.getByRole('alert').count())throw Error('UI reports a runtime error.');
      await page.evaluate(()=>document.fonts.ready);
      if(!await page.evaluate(()=>document.fonts.check('16px Geist')&&document.fonts.check('16px "Space Grotesk"')&&document.fonts.check('16px "Geist Mono"')))throw Error('Production fonts did not load.');
      const text=await page.locator('body').innerText();
      const owner=os.homedir();if(text.includes(owner)||/password|secret_key|access_key|passphrase|bearer token/i.test(await page.locator('input').evaluateAll(nodes=>nodes.map(n=>n.value).join('\n'))))throw Error('Private input appeared in the capture.');
      const surface=page.locator(landmark==='Library'?'section[aria-label="Library controls"]':'section[aria-label="First-run setup"]');
      const bounds=await surface.boundingBox();if(!bounds||bounds.height>6000)throw Error('Capture surface exceeds its bounded screenshot size.');
      const geometry=await surface.evaluate(node=>{const heading=node.querySelector('h2'),paragraph=heading?.nextElementSibling;return {heading_to_text_gap:heading&&paragraph?paragraph.getBoundingClientRect().top-heading.getBoundingClientRect().bottom:null}});
      if(geometry.heading_to_text_gap!==null&&geometry.heading_to_text_gap>100)throw Error('Production grid stretched the setup text away from its heading.');
      // Chromium clips offscreen descendants of the application's scroll pane.
      // Expand only this headless viewport, preserving width and the rendered
      // section's natural geometry; no DOM or stylesheet override is applied.
      const captureViewport={width:viewport.width,height:Math.max(viewport.height,Math.ceil(bounds.height+900))};
      if(captureViewport.height>7000)throw Error('Offscreen capture viewport exceeds its bound.');
      await page.setViewportSize(captureViewport);
      const expanded=await surface.boundingBox();if(!expanded||Math.abs(expanded.height-bounds.height)>20)throw Error('Viewport expansion reshaped the captured controls.');
      const fullyVisible=await surface.evaluate(node=>{const r=node.getBoundingClientRect(),parent=document.querySelector('.desktop-content').getBoundingClientRect();return r.height<=parent.height&&r.bottom<=parent.bottom+1});
      if(!fullyVisible)throw Error('The screenshot surface remains clipped by the app scroll pane.');
      const file=path.join(pending,name);const bytes=await surface.screenshot({path:file,animations:'disabled',caret:'hide'});
      images.push({path:name,sha256:sha(bytes),width:bytes.readUInt32BE(16),height:bytes.readUInt32BE(20),landmark,caption,rendered_geometry:geometry,capture_viewport:captureViewport});
      await page.setViewportSize({width:viewport.width,height:viewport.height});
    };
    await page.goto(origin,{waitUntil:'networkidle'});
    await page.getByLabel('Library location',{exact:true}).fill(library);
    await page.getByRole('button',{name:'Preview location and space',exact:true}).click();
    await screenshot('setup-location.png','Workspace and location (Step 1 of 5)','Figure 1. Optional setup previews an editable library location outside program files.');
    await page.getByRole('button',{name:'Save location and continue',exact:true}).click();
    await screenshot('setup-backends.png','Backend profiles (Step 2 of 5)','Figure 2. Keep local backend defaults or explicitly choose the three storage roles before creating a library.');
    await page.getByRole('button',{name:'Create configured workspace',exact:true}).click();
    await screenshot('setup-processing.png','Optional local processing (Step 3 of 5)','Figure 3. Optional processing follows successful library creation; import and supplied subtitles need no model download.');
    await page.getByRole('button',{name:'Set up later',exact:true}).click();
    await screenshot('library-ready.png','Library','Figure 4. Deferring setup leaves the ordinary Library controls available.');
    await page.getByRole('button',{name:'Set up library',exact:true}).click();
    await page.getByRole('heading',{name:'Optional local processing (Step 3 of 5)',exact:true}).waitFor();
    if(errors.length||blocked.length)throw Error('Capture had browser errors or attempted an external request: '+errors.slice(0,8).join('; ')+' '+transportErrors.join('; ')+' '+helperStderr.slice(-1000));
    const transport=await backend('/receipt');
    const after=await sourceSnapshot();if(after.revision!==source.revision||after.tree_sha256!==source.tree_sha256)throw Error('Source changed during capture; no images published.');
    const bundle=[...assets].map(([name,bytes])=>({path:name.slice(1),sha256:sha(bytes)}));
    const receipt=assertReceipt({kind:'application-ui-capture',schema_version:VERSION,source,renderer:{kind:'chromium-headless',browser_version:browser.version(),playwright_version:PLAYWRIGHT_VERSION,platform:os.platform(),architecture:os.arch(),windows_build:os.platform()==='win32'?os.release():null},fixture:{id:fixtureID,isolated:true,private_inputs:false},viewport,locale:'en-US',signing:'unsigned',binaries:{cli_sha256:sha(cliBytes),bridge_sha256:sha(bridgeBytes)},bundle,transport,verification:{production_frontend:true,shared_desktop_bridge:true,shared_runtime:true,deferred_setup_resumed:true,external_requests:0,browser_errors:0,native_webview:false},images});
    await mkdir(options.output,{recursive:true});for(const image of images)await writeFile(path.join(options.output,image.path),await readFile(path.join(pending,image.path)));
    await writeFile(path.join(options.output,'application-capture.json'),JSON.stringify(receipt,null,2)+'\n','utf8');
    return receipt;
  } finally {
    await browser?.close();if(server)await new Promise(resolve=>server.close(resolve));
    if(endpoint&&helperToken)try{await fetch(endpoint+'/shutdown',{method:'POST',headers:{'X-Capture-Token':helperToken,Origin:endpoint},signal:AbortSignal.timeout(10000)})}catch{}
    if(helper){let timer;const code=await Promise.race([exitPromise,new Promise(resolve=>{timer=setTimeout(()=>resolve(null),12000)})]);clearTimeout(timer);if(code===null){helper.kill();await exitPromise}if(code!==0&&code!==null)throw Error('Capture bridge cleanup failed: '+helperStderr.slice(-1000))}
    const identity=JSON.parse(await readFile(path.join(fixture,'capture-owner.json'),'utf8'));if(identity.fixture_id!==fixtureID||!path.basename(fixture).startsWith('insonic-capture-'))throw Error('Fixture cleanup identity mismatch.');await rm(fixture,{recursive:true,force:true});
  }
}

if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url)){
  try{const receipt=await capture(parseOptions(process.argv.slice(2)));console.log(JSON.stringify({kind:receipt.kind,source:receipt.source.revision,images:receipt.images.map(x=>x.path),renderer:receipt.renderer}));}
  catch(error){console.error(error.message);process.exitCode=1}
}
