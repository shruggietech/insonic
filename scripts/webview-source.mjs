// SPDX-License-Identifier: Apache-2.0
import { createHash } from 'node:crypto';
import { readFile, readdir, lstat } from 'node:fs/promises';
import { join } from 'node:path';

export const webviewModule = 'github.com/wailsapp/go-webview2';
export const webviewVersion = 'v1.0.23';
export const wailsModule = 'github.com/wailsapp/wails/v2';
export const wailsVersion = 'v2.16.0';
const policies = {
  [webviewModule]: {
    module: webviewModule, version: webviewVersion, directory: 'third_party/go-webview2',
    modified: ['pkg/combridge/iunknown.go', 'pkg/edge/ICoreWebView2Controller.go', 'pkg/edge/ICoreWebView2Deferral.go', 'pkg/edge/ICoreWebView2WebResourceRequestedEventArgs.go', 'pkg/edge/ICoreWebView2WebResourceResponse.go', 'pkg/edge/chromium.go', 'pkg/edge/corewebview2.go', 'pkg/edge/create_env_go.go'],
    added: ['LICENSE.insonic', 'NOTICE.insonic', 'pkg/edge/browser_bootstrap_windows.go', 'pkg/edge/browser_bootstrap_windows_test.go', 'pkg/edge/callback_bridge_windows.go', 'pkg/edge/callback_bridge_windows_test.go', 'pkg/edge/callback_loader_go_windows_test.go', 'pkg/edge/startup_windows.go', 'pkg/edge/startup_windows_test.go', 'pkg/edge/window_lifetime_windows.go', 'pkg/edge/window_lifetime_windows_test.go'],
  },
  [wailsModule]: {
    module: wailsModule, version: wailsVersion, directory: 'third_party/wails',
    modified: ['internal/app/app_dev.go', 'internal/app/app_production.go', 'internal/frontend/desktop/windows/frontend.go'],
    added: ['LICENSE.insonic', 'NOTICE.insonic', 'internal/app/app_lifecycle_test.go'],
  },
};

function names(files) {
  if (!files || typeof files !== 'object' || Array.isArray(files)) throw new Error('Maintained source file authority is invalid');
  const values = Object.keys(files).sort();
  if (!values.length || values.some(name => !/^[A-Za-z0-9._/+-]+$/.test(name) || name.split('/').some(part => !part || part === '.' || part === '..') || !/^[a-f0-9]{64}$/.test(files[name]))) throw new Error('Maintained source file identity is invalid');
  return values;
}
function treeHash(files, module, version) {
  const lines = names(files).map(name => files[name] + '  ' + module + '@' + version + '/' + name + '\n').join('');
  return 'h1:' + createHash('sha256').update(lines).digest('base64');
}
export function webviewTreeHash(files) { return treeHash(files, webviewModule, webviewVersion); }

async function verifySource(root, policy) {
  const { module, version, directory } = policy;
  const authority = JSON.parse(await readFile(join(root, directory + '.source.json'), 'utf8'));
  if (authority.module !== module || authority.version !== version || authority.directory !== directory) throw new Error('Source authority differs from the selected maintained module');
  const upstream = names(authority.upstream_files), current = names(authority.files);
  if (treeHash(authority.upstream_files, module, version) !== authority.upstream_sum || treeHash(authority.files, module, version) !== authority.patched_sum) throw new Error('Maintained source tree checksum differs');
  // Go's goModSum hashes one summary entry named go.mod (without a module prefix).
  // Its content digest is already bound by the verified pristine tree authority.
  if (!Object.hasOwn(authority.upstream_files, 'go.mod')) throw new Error('Maintained upstream go.mod identity is absent');
  const goModSum = 'h1:' + createHash('sha256').update(authority.upstream_files['go.mod'] + '  go.mod\n').digest('base64');
  const sums = (await readFile(join(root, 'go.sum'), 'utf8')).split('\n');
  for (const [selectedVersion, checksum] of [[version, authority.upstream_sum], [version + '/go.mod', goModSum]]) {
    const prefix = module + ' ' + selectedVersion + ' ', entries = sums.filter(line => line.startsWith(prefix));
    if (entries.length !== 1 || entries[0] !== prefix + checksum) throw new Error('Upstream source is not bound to go.sum: ' + module + ' ' + selectedVersion);
  }
  const mod = await readFile(join(root, 'go.mod'), 'utf8');
  const declarations = mod.split(String.fromCharCode(10)).map(line => line.trim().replaceAll(String.fromCharCode(9), ' ').split(' ').filter(Boolean));
  if (!declarations.some(fields => fields[0] === module && fields[1] === version)) throw new Error('Maintained logical module pin differs');
  if (!mod.split(String.fromCharCode(10)).includes('replace ' + module + ' => ./' + directory)) throw new Error('Maintained source replacement is absent');
  const changed = upstream.filter(name => authority.files[name] !== authority.upstream_files[name]);
  const added = current.filter(name => !Object.hasOwn(authority.upstream_files, name));
  if (upstream.some(name => !Object.hasOwn(authority.files, name)) || JSON.stringify(changed) !== JSON.stringify(policy.modified) || JSON.stringify(added) !== JSON.stringify(policy.added) || JSON.stringify([...authority.modified].sort()) !== JSON.stringify(changed) || JSON.stringify([...authority.added].sort()) !== JSON.stringify(added)) throw new Error('Maintained patch scope differs');
  const actual = [];
  async function inspect(prefix = '') {
    const path = join(root, directory, prefix), stat = await lstat(path);
    if (stat.isSymbolicLink()) throw new Error('Maintained source contains a linked entry');
    if (stat.isDirectory()) {
      for (const entry of await readdir(path)) await inspect(prefix ? prefix + '/' + entry : entry);
    } else if (stat.isFile()) {
      actual.push(prefix);
      if (authority.files[prefix] !== createHash('sha256').update(await readFile(path)).digest('hex')) throw new Error('Maintained source bytes differ: ' + prefix);
    } else throw new Error('Maintained source contains an unsupported entry');
  }
  await inspect();
  if (JSON.stringify(actual.sort()) !== JSON.stringify(current)) throw new Error('Maintained source membership differs');
  return authority;
}
export function verifyWebviewSource(root) { return verifySource(root, policies[webviewModule]); }
export function verifyWailsSource(root) { return verifySource(root, policies[wailsModule]); }
