// SPDX-License-Identifier: Apache-2.0
import fs from 'node:fs/promises';
import path from 'node:path';
import crypto from 'node:crypto';
import { spawnSync } from 'node:child_process';

const hash = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
function capturedSources(repositoryRoot, receipt, changed) {
  const revision = receipt.source.revision;
  const env = Object.fromEntries(Object.entries(process.env).filter(([name]) => !/^GIT_|TOKEN|SECRET|PASSWORD|CREDENTIAL|ASKPASS/i.test(name)));
  Object.assign(env, {GIT_TERMINAL_PROMPT:'0',GIT_CONFIG_NOSYSTEM:'1'});
  const options = {cwd:repositoryRoot,env,windowsHide:true,shell:false,maxBuffer:64 << 20};
  // All history observations share one deadline, rather than permitting each
  // of up to 64 candidates to consume a separate command timeout.
  const deadline=Date.now()+15000;
  function observe(args,input) {
    const timeout=deadline-Date.now();
    if(timeout<1)throw new Error('Screenshot source differs; captured-history verification exceeded its time bound.');
    const result=spawnSync('git',args,{...options,timeout,input});
    if(result.error?.code==='ETIMEDOUT')throw new Error('Screenshot source differs; captured-history verification exceeded its time bound.');
    return result;
  }
  function blobs(references) {
    const result=observe(['cat-file','--batch'],references.join('\n')+'\n');
    if(result.status!==0||!result.stdout)throw new Error('Screenshot source differs; captured Git objects are unavailable.');
    let offset=0;const files=[];
    for(const reference of references){
      const end=result.stdout.indexOf(10,offset);
      if(end<0)throw new Error('Screenshot source differs; incomplete captured source.');
      const header=result.stdout.subarray(offset,end).toString('ascii');offset=end+1;
      if(header===reference+' missing'){files.push(null);continue;}
      const match=header.match(/^[a-f0-9]{40} blob ([0-9]+)$/);
      const size=match?Number(match[1]):-1;
      if(size<0||size>16<<20||offset+size>=result.stdout.length||result.stdout[offset+size]!==10)throw new Error('Screenshot source differs; invalid captured object.');
      files.push(result.stdout.subarray(offset,offset+size));offset+=size+1;
    }
    if(offset!==result.stdout.length)throw new Error('Screenshot source differs; extra captured objects.');
    return files;
  }
  const paths=['VERSION',...receipt.source.files.map(item=>item.path)];
  const matches=files=>files[0]?.toString('utf8').trim()===receipt.schema_version&&receipt.source.files.every((item,index)=>files[index+1]&&hash(files[index+1])===item.sha256);
  const ancestry=observe(['merge-base','--is-ancestor',revision,'HEAD']);
  if(ancestry.status===0){
    const files=blobs(paths.map(name=>revision+':'+name));
    if(files[0]?.toString('utf8').trim()!==receipt.schema_version)throw new Error('Screenshot source differs from its captured version.');
    if(!matches(files))throw new Error('Screenshot source differs from its recorded capture.');
    return;
  }
  // Squash merges do not retain the CI test-merge commit in main's ancestry.
  // Accept only one coherent ancestral snapshot with every recorded byte hash
  // and version. Preserve the original capture revision and receipt unchanged.
  // A coherent capture can be newer than the last change to the anchor file.
  // Search actual ancestors, including commits that changed only other files.
  const history=observe(['log','--format=%H','--max-count=64','HEAD']);
  if(history.status!==0)throw new Error('Screenshot source differs; captured history is unavailable. Fetch complete history.');
  const revisions=history.stdout.toString('ascii').trim().split(/\r?\n/).filter(value=>/^[a-f0-9]{40}$/.test(value));
  if(revisions.length){
    const anchors=blobs(revisions.map(value=>value+':'+changed.path));
    const candidates=revisions.filter((value,index)=>anchors[index]&&hash(anchors[index])===changed.sha256);
    // One candidate per observation keeps complete snapshots within the byte
    // bound even when several individually valid inventories total over it.
    for(const candidate of candidates){
      const files=blobs(paths.map(name=>candidate+':'+name));
      if(matches(files))return;
    }
  }
  throw new Error('Screenshot source differs; no matching complete snapshot exists in bounded repository history. Refresh the capture or fetch complete history.');
}

async function verifySources(repositoryRoot, receipt) {
  if(receipt.source.files.length>4096)throw new Error('Screenshot source inventory exceeds its bound.');
  let changed=null;
  for (const item of receipt.source.files) {
    if (typeof item.path !== 'string' || !item.path || path.isAbsolute(item.path) || item.path.includes('..') || /[\\:\x00-\x1f]/.test(item.path)
        || !/^[a-f0-9]{64}$/.test(item.sha256)) throw new Error('Invalid capture source path or digest.');
    try {if(hash(await fs.readFile(path.join(repositoryRoot,item.path)))!==item.sha256)changed??=item;}
    catch(error){if(error.code==='ENOENT')changed??=item;else throw error;}
  }
  // Versioned images describe their recorded clean source. Later backend edits
  // cannot invalidate that historical evidence or rewrite its receipt.
  if(changed)capturedSources(repositoryRoot,receipt,changed);
}
export async function prepareScreenshots(repositoryRoot, publicRoot) {
  const source = path.join(repositoryRoot, 'docs/screenshots');
  const publicDirectory = path.resolve(publicRoot);
  const destinationRoot = path.resolve(publicDirectory, 'screenshots');
  if (path.dirname(destinationRoot) !== publicDirectory) throw new Error('Invalid screenshot output directory.');
  await fs.rm(destinationRoot, { recursive: true, force: true });
  try { await fs.access(source); } catch { return []; }
  const copied = [];
  for (const version of await fs.readdir(source, { withFileTypes: true })) {
    if (!version.isDirectory() || !/^v[0-9]+\.[0-9]+\.[0-9]+$/.test(version.name)) throw new Error('Invalid screenshot version directory.');
    const directory = path.join(source, version.name);
    const names = await fs.readdir(directory);
    for (const name of names) if (!(await fs.lstat(path.join(directory, name))).isFile()) throw new Error('Screenshot inputs must be regular files.');
    const witnessed = new Set();
    const receipts = names.filter(name => /^[a-z0-9-]+-capture\.json$/.test(name));
    for (const name of receipts) {
      const receipt = JSON.parse(await fs.readFile(path.join(directory, name), 'utf8'));
      if (receipt.schema_version !== version.name.slice(1) || !Array.isArray(receipt.images) || !receipt.images.length
          || !receipt.fixture?.isolated || receipt.fixture.private_inputs !== false || !receipt.source?.revision
          || receipt.source.dirty !== false || !Array.isArray(receipt.source.files) || !receipt.source.files.length
          || !/^[a-f0-9]{40}$/.test(receipt.source.revision) || !receipt.renderer?.kind || !receipt.locale) throw new Error('Screenshot provenance is incomplete.');
      await verifySources(repositoryRoot,receipt);
      for (const image of receipt.images) {
        const basename = path.posix.basename(image.path || '');
        if (![basename, `docs/screenshots/${version.name}/${basename}`].includes(image.path) || !/^[a-z0-9-]+\.png$/.test(basename) || witnessed.has(basename) || !image.landmark || !image.caption) throw new Error('Screenshot identity or accessible caption is incomplete.');
        const target = path.join(directory, basename);
        if (!(await fs.lstat(target)).isFile()) throw new Error('Screenshot must be a regular file.');
        const bytes = await fs.readFile(target);
        if (hash(bytes) !== image.sha256 || !bytes.subarray(0, 8).equals(Buffer.from([137,80,78,71,13,10,26,10]))) throw new Error('Screenshot bytes differ from capture receipt.');
        if (bytes.length < 24 || bytes.readUInt32BE(16) !== image.width || bytes.readUInt32BE(20) !== image.height) throw new Error('Screenshot dimensions differ from capture receipt.');
        witnessed.add(basename);
      }
    }
    for (const name of names) {
      if (!receipts.includes(name) && !witnessed.has(name)) throw new Error('Screenshot asset lacks verified capture provenance.');
      const destination = path.join(publicRoot, 'screenshots', version.name, name);
      await fs.mkdir(path.dirname(destination), { recursive: true });
      await fs.copyFile(path.join(directory, name), destination);
      copied.push(path.posix.join('screenshots', version.name, name));
    }
  }
  return copied;
}
