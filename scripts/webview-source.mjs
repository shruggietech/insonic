// SPDX-License-Identifier: Apache-2.0
import { createHash } from 'node:crypto';
import { readFile, readdir, lstat } from 'node:fs/promises';
import { join } from 'node:path';

export const webviewModule = 'github.com/wailsapp/go-webview2';
export const webviewVersion = 'v1.0.23';
const directory = 'third_party/go-webview2';
const addedFiles = ['LICENSE.insonic', 'NOTICE.insonic', 'pkg/edge/callback_bridge_windows.go', 'pkg/edge/callback_bridge_windows_test.go', 'pkg/edge/callback_loader_go_windows_test.go', 'pkg/edge/window_lifetime_windows.go', 'pkg/edge/window_lifetime_windows_test.go'];

function names(files) {
  if (!files || typeof files !== 'object' || Array.isArray(files)) throw new Error('WebView source file authority is invalid');
  const values = Object.keys(files).sort();
  if (!values.length || values.some(name => !/^[A-Za-z0-9._/-]+$/.test(name) || name.split('/').some(part => !part || part === '.' || part === '..') || !/^[a-f0-9]{64}$/.test(files[name]))) throw new Error('WebView source file identity is invalid');
  return values;
}

export function webviewTreeHash(files) {
  const lines = names(files).map(name => `${files[name]}  ${webviewModule}@${webviewVersion}/${name}\n`).join('');
  return `h1:${createHash('sha256').update(lines).digest('base64')}`;
}

export async function verifyWebviewSource(root) {
  const authority = JSON.parse(await readFile(join(root, 'third_party/go-webview2.source.json'), 'utf8'));
  if (authority.module !== webviewModule || authority.version !== webviewVersion || authority.directory !== directory) throw new Error('WebView source authority differs from the sole maintained module');
  const upstream = names(authority.upstream_files);
  const current = names(authority.files);
  if (webviewTreeHash(authority.upstream_files) !== authority.upstream_sum || webviewTreeHash(authority.files) !== authority.patched_sum) throw new Error('WebView source tree checksum differs');
  const sum = await readFile(join(root, 'go.sum'), 'utf8');
  if (!sum.split('\n').includes(`${webviewModule} ${webviewVersion} ${authority.upstream_sum}`)) throw new Error('WebView upstream source is not bound to go.sum');
  const mod = await readFile(join(root, 'go.mod'), 'utf8');
  if (!new RegExp(`^\\s*${webviewModule.replaceAll('.', '\\.')} ${webviewVersion.replaceAll('.', '\\.')}([\\s]|$)`, 'm').test(mod)) throw new Error('WebView logical module pin differs');
  if (!new RegExp(`^replace ${webviewModule.replaceAll('.', '\\.')} => \\./${directory}$`, 'm').test(mod)) throw new Error('WebView maintained source replacement is absent');
  const changed = upstream.filter(name => authority.files[name] !== authority.upstream_files[name]);
  const added = current.filter(name => !Object.hasOwn(authority.upstream_files, name));
  if (upstream.some(name => !Object.hasOwn(authority.files, name)) || JSON.stringify(changed) !== JSON.stringify(['pkg/combridge/iunknown.go', 'pkg/edge/ICoreWebView2Controller.go', 'pkg/edge/ICoreWebView2Deferral.go', 'pkg/edge/ICoreWebView2WebResourceRequestedEventArgs.go', 'pkg/edge/ICoreWebView2WebResourceResponse.go', 'pkg/edge/chromium.go', 'pkg/edge/corewebview2.go', 'pkg/edge/create_env_go.go']) || JSON.stringify(added) !== JSON.stringify(addedFiles) || JSON.stringify([...authority.modified].sort()) !== JSON.stringify(changed) || JSON.stringify([...authority.added].sort()) !== JSON.stringify(added)) throw new Error('WebView maintained patch scope differs');
  const actual = [];
  async function inspect(prefix = '') {
    const path = join(root, directory, prefix);
    const stat = await lstat(path);
    if (stat.isSymbolicLink()) throw new Error('WebView maintained source contains a linked entry');
    if (stat.isDirectory()) {
      for (const entry of await readdir(path)) await inspect(prefix ? `${prefix}/${entry}` : entry);
    } else if (stat.isFile()) {
      actual.push(prefix);
      const digest = createHash('sha256').update(await readFile(path)).digest('hex');
      if (authority.files[prefix] !== digest) throw new Error(`WebView maintained source bytes differ: ${prefix}`);
    } else throw new Error('WebView maintained source contains an unsupported entry');
  }
  await inspect();
  if (JSON.stringify(actual.sort()) !== JSON.stringify(current)) throw new Error('WebView maintained source membership differs');
  return authority;
}
