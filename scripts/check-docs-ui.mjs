// SPDX-License-Identifier: Apache-2.0
// Browser checks own a fresh headless profile. They never attach to a desktop browser.
import { readFile, writeFile, readdir, mkdir, mkdtemp, rm, stat, lstat, realpath, open } from 'node:fs/promises';
import { constants } from 'node:fs';
import { createHash } from 'node:crypto';
import path from 'node:path';
import os from 'node:os';
import http from 'node:http';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { chromium } from 'playwright-core';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const mime = { '.html': 'text/html; charset=utf-8', '.css': 'text/css', '.js': 'text/javascript', '.json': 'application/json', '.svg': 'image/svg+xml', '.png': 'image/png', '.ico': 'image/x-icon', '.woff2': 'font/woff2', '.woff': 'font/woff' };

export function parseOptions(args) {
  const options = { out: path.join(root, 'site/out'), offline: path.join(root, 'site/offline'), output: path.join(root, 'build/s028-docs-ui'), browser: process.env.INSONIC_DOCS_BROWSER };
  const seen = new Set();
  for (let index = 0; index < args.length; index += 1) {
    const name = { '--out': 'out', '--offline': 'offline', '--output': 'output', '--browser': 'browser' }[args[index]];
    if (!name || seen.has(name) || !args[index + 1] || args[index + 1].startsWith('--')) throw new Error('Use [--out DIRECTORY] [--offline DIRECTORY] [--output DIRECTORY] [--browser FILE].');
    seen.add(name);
    options[name] = path.resolve(args[++index]);
  }
  return options;
}

export function boundedPath(directory, relative) {
  const target = path.resolve(directory, relative);
  if (target !== directory && !target.startsWith(directory + path.sep)) throw new Error('Asset leaves its documentation export.');
  return target;
}

function inside(directory, filename) {
  const relative = path.relative(directory, filename);
  return relative === '' || relative !== '..' && !relative.startsWith('..' + path.sep) && !path.isAbsolute(relative);
}

async function validateExportPath(directory, relative) {
  const requested = boundedPath(directory, relative);
  const base = await lstat(directory, { bigint: true });
  if (!base.isDirectory() || base.isSymbolicLink()) throw new Error('Documentation export must be a real directory.');
  const boundary = await realpath(directory);
  const segments = path.relative(directory, requested).split(path.sep).filter(Boolean);
  let filename = directory;
  let observed = base;
  for (const segment of segments) {
    if (!observed.isDirectory()) throw new Error('Documentation asset parent is not a directory.');
    filename = path.join(filename, segment);
    observed = await lstat(filename, { bigint: true });
    if (observed.isSymbolicLink()) throw new Error('Symbolic links and junctions are not documentation export assets.');
    if (!inside(boundary, await realpath(filename))) throw new Error('Asset real target leaves its documentation export.');
  }
  if (observed.isDirectory()) return validateExportPath(directory, path.join(relative, 'index.html'));
  if (!observed.isFile()) throw new Error('Documentation asset must be a regular file.');
  const canonical = await realpath(filename);
  if (!inside(boundary, canonical)) throw new Error('Asset real target leaves its documentation export.');
  return { filename: canonical, observed, boundary };
}

const sameFile = (first, second) => first.dev === second.dev && first.ino === second.ino;

// Component checks reject Windows junctions and platforms without O_NOFOLLOW.
// The opened regular handle must retain the identity validated before/after open;
// bytes and cache entries are created only after those checks succeed.
export async function readExportAsset(directory, relative, cache = new Map()) {
  const selected = await validateExportPath(directory, relative);
  const handle = await open(selected.filename, constants.O_RDONLY | (constants.O_NOFOLLOW || 0));
  try {
    const opened = await handle.stat({ bigint: true });
    const confirmed = await validateExportPath(directory, relative);
    if (!opened.isFile() || !sameFile(selected.observed, opened) || !sameFile(confirmed.observed, opened) || selected.boundary !== confirmed.boundary || selected.filename !== confirmed.filename) throw new Error('Documentation asset identity changed while opening.');
    const key = `${selected.filename}:${opened.dev}:${opened.ino}:${opened.size}:${opened.mtimeNs}`;
    if (!cache.has(key)) {
      const bytes = await handle.readFile();
      const afterRead = await handle.stat({ bigint: true });
      if (!sameFile(opened, afterRead) || opened.size !== afterRead.size || opened.mtimeNs !== afterRead.mtimeNs) throw new Error('Documentation asset changed while reading.');
      cache.set(key, bytes);
    }
    return { filename: selected.filename, bytes: cache.get(key) };
  } finally { await handle.close(); }
}

async function htmlFiles(directory) {
  const files = [];
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    const filename = path.join(directory, entry.name);
    if (entry.isDirectory()) files.push(...await htmlFiles(filename));
    else if (entry.name.endsWith('.html')) files.push(filename);
  }
  return files.sort();
}

function browserEnvironment(fixture, executable) {
  const env = {};
  for (const name of ['SystemRoot', 'WINDIR', 'SystemDrive', 'COMSPEC']) if (process.env[name]) env[name] = process.env[name];
  Object.assign(env, {
    PATH: [executable ? path.dirname(executable) : '', process.env.SystemRoot ? path.join(process.env.SystemRoot, 'System32') : '/usr/bin:/bin'].filter(Boolean).join(path.delimiter),
    TEMP: fixture, TMP: fixture, TMPDIR: fixture, HOME: path.join(fixture, 'home'), USERPROFILE: path.join(fixture, 'home'),
    APPDATA: path.join(fixture, 'home/AppData/Roaming'), LOCALAPPDATA: path.join(fixture, 'home/AppData/Local'),
    XDG_CONFIG_HOME: path.join(fixture, 'home/config'), XDG_CACHE_HOME: path.join(fixture, 'home/cache'),
  });
  return env;
}

export async function serve(exports) {
  const cache = new Map();
  const server = http.createServer(async (request, response) => {
    try {
      const url = new URL(request.url, 'http://127.0.0.1');
      const [, mount, ...segments] = decodeURIComponent(url.pathname).split('/');
      const directory = exports[mount];
      if (!directory) { response.writeHead(404).end(); return; }
      const { filename, bytes } = await readExportAsset(directory, segments.join('/'), cache);
      response.writeHead(200, { 'Content-Type': mime[path.extname(filename)] || 'application/octet-stream', 'Cache-Control': 'no-cache' }).end(bytes);
    } catch { response.writeHead(404).end(); }
  });
  await new Promise((resolve, reject) => { server.once('error', reject); server.listen(0, '127.0.0.1', resolve); });
  return { server, origin: `http://127.0.0.1:${server.address().port}`, cache };
}

export function sanitizedRequestURL(value) {
  try { const url = new URL(value); url.username = ''; url.password = ''; url.search = ''; url.hash = ''; return url.href; }
  catch { return 'Unparseable blocked URL'; }
}

export async function configureBrowserNetwork(context, { origin, out, offline, externalRequests, webSocketAttempts }) {
  // HTTP routing does not intercept WebSocket upgrades. Documentation uses no
  // owned WebSocket protocol, including on loopback: never connect to a server.
  await context.routeWebSocket('**/*', async socket => {
    webSocketAttempts.push(sanitizedRequestURL(socket.url()));
    await socket.close({ code: 1008, reason: 'Documentation verification does not permit WebSockets.' });
  });
  await context.route('**/*', route => {
    const url = route.request().url();
    if (url.startsWith(origin + '/') || url.startsWith('data:') || url.startsWith('blob:') || [out, offline].some(directory => url.startsWith(pathToFileURL(directory + path.sep).href))) return route.continue();
    externalRequests.push(sanitizedRequestURL(url));
    return route.abort();
  });
}

async function verifyWebSocketIsolation(browser, contexts, network, deadline) {
  const result = { attempts: [], connections: 0, upgrades: 0, received_bytes: 0, passed: false };
  const sockets = new Set();
  const server = http.createServer((_, response) => response.writeHead(404).end());
  server.on('connection', socket => {
    result.connections += 1; sockets.add(socket);
    socket.on('data', bytes => { result.received_bytes += bytes.length; });
    socket.once('close', () => sockets.delete(socket));
  });
  server.on('upgrade', (request, socket) => {
    result.upgrades += 1;
    const accept = createHash('sha1').update(request.headers['sec-websocket-key'] + '258EAFA5-E914-47DA-95CA-C5AB0DC85B11').digest('base64');
    socket.write(`HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: ${accept}\r\n\r\n`);
  });
  let context;
  try {
    await new Promise((resolve, reject) => { server.once('error', reject); server.listen(0, '127.0.0.1', resolve); });
    context = await browser.newContext({ serviceWorkers: 'block' }); contexts.push(context);
    await configureBrowserNetwork(context, { ...network, externalRequests: [], webSocketAttempts: result.attempts });
    const page = await context.newPage();
    const url = `ws://127.0.0.1:${server.address().port}/owned-isolation-probe?credential=fixture-only`;
    result.browser = await bounded('owned WebSocket isolation probe', () => page.evaluate(url => new Promise(resolve => {
      const socket = new WebSocket(url);
      const timer = setTimeout(() => { socket.close(); resolve({ closed: false }); }, 1500);
      socket.onopen = () => socket.send('owned-isolation-probe');
      socket.onclose = event => { clearTimeout(timer); resolve({ closed: true, code: event.code }); };
      socket.onerror = () => {}; // The explicit close result is checked below.
    }), url), Math.min(5000, deadline - Date.now()));
    await bounded('owned WebSocket probe completion', () => context.close(), Math.min(8000, deadline - Date.now()));
    assert(result.attempts.length === 1 && result.attempts[0] === sanitizedRequestURL(url) && result.browser.closed && result.browser.code === 1008, 'WebSocket isolation probe must record and close its attempted connection.');
    assert(result.connections === 0 && result.upgrades === 0 && result.received_bytes === 0, 'WebSocket isolation probe reached its owned loopback server.');
    result.passed = true;
    return result;
  } finally {
    if (context) await bounded('owned WebSocket probe context cleanup', () => context.close(), 8000);
    for (const socket of sockets) socket.destroy();
    await new Promise(resolve => server.close(resolve));
  }
}

function assert(condition, message) { if (!condition) throw new Error(message); }

async function inspectPage(page) {
  return page.evaluate(() => {
    const visible = element => Boolean(element.getClientRects().length) && getComputedStyle(element).visibility !== 'hidden';
    const name = element => (element.getAttribute('aria-label') || element.textContent || [...element.querySelectorAll('img[alt]')].map(image => image.alt).join(' ')).trim();
    const ids = [...document.querySelectorAll('[id]')].map(element => element.id);
    const headings = [...document.querySelectorAll('h1,h2,h3,h4,h5,h6')].filter(visible).map(element => ({ level: Number(element.tagName[1]), text: element.textContent.trim() }));
    const sidebar = document.querySelector('.docs-sidebar');
    const disclosure = sidebar?.querySelector('details');
    return {
      lang: document.documentElement.lang,
      main: document.querySelectorAll('main').length,
      h1: document.querySelectorAll('h1').length,
      width: document.documentElement.clientWidth,
      scrollWidth: document.documentElement.scrollWidth,
      unnamedLinks: [...document.querySelectorAll('a[href]')].filter(element => visible(element) && !name(element)).length,
      unnamedButtons: [...document.querySelectorAll('button')].filter(element => visible(element) && !name(element)).length,
      missingAlt: document.querySelectorAll('img:not([alt])').length,
      duplicateIds: ids.filter((id, index) => ids.indexOf(id) !== index),
      headings,
      sidebar: sidebar ? { height: sidebar.getBoundingClientRect().height, nativeDisclosure: Boolean(disclosure?.querySelector('summary')), open: disclosure?.open, current: sidebar.querySelectorAll('a[aria-current="page"]').length } : null,
      overflow: [...document.querySelectorAll('body *')].filter(element => {
        if (!visible(element) || element.closest('pre,.table-scroll,.diagram')) return false;
        const box = element.getBoundingClientRect();
        return box.width > 0 && (box.right > document.documentElement.clientWidth + 1 || box.left < -1) && !element.matches('.skip-link');
      }).slice(0, 8).map(element => ({ tag: element.tagName, class: element.className?.baseVal ?? element.className, text: element.textContent.slice(0, 80) })),
    };
  });
}

function assertPage(result, label, { compact = true, bare = false } = {}) {
  assert(result.lang === 'en', `${label}: document language is missing.`);
  assert(result.main === 1 && result.h1 === 1, `${label}: expected one main landmark and one h1.`);
  assert(!result.unnamedLinks && !result.unnamedButtons && !result.missingAlt && !result.duplicateIds.length, `${label}: missing accessible names/alternatives or duplicate IDs: ${JSON.stringify(result)}.`);
  assert(result.scrollWidth <= result.width + 1 && !result.overflow.length, `${label}: unintended page overflow: ${JSON.stringify(result.overflow)} (${result.scrollWidth}/${result.width}).`);
  if (result.sidebar && !bare) {
    assert(result.sidebar.nativeDisclosure, `${label}: documentation navigation must use a native disclosure.`);
    assert(result.sidebar.current === 1, `${label}: documentation navigation must identify its current article or reference.`);
    if (compact) assert(result.sidebar.open === false && result.sidebar.height <= 120, `${label}: mobile navigation must begin compact and collapsed.`);
  }
}

async function bounded(phase, operation, timeout) {
  assert(timeout > 0, `Documentation browser deadline exhausted before ${phase}.`);
  let timer;
  try { return await Promise.race([operation(), new Promise((_, reject) => { timer = setTimeout(() => reject(new Error(`Documentation browser deadline exhausted during ${phase}.`)), timeout); })]); }
  finally { clearTimeout(timer); }
}

async function ready(page, url, deadline, { scripts = true } = {}) {
  assert(Date.now() < deadline, 'Documentation browser qualification exceeded its180second limit.');
  await page.goto(url, { waitUntil: 'load', timeout: Math.min(20000, deadline - Date.now()) });
  await bounded('font readiness', () => page.evaluate(() => document.fonts.ready), Math.min(5000, deadline - Date.now()));
  if (scripts) await page.waitForFunction(() => [...document.querySelectorAll('.mermaid')].every(element => element.dataset.processed === 'true' || element.parentElement.querySelector('.diagram-error')), null, { timeout: Math.min(10000, deadline - Date.now()) });
}

async function inspectReadability(page) {
  return page.evaluate(() => {
    const visible = element => Boolean(element.getClientRects().length) && getComputedStyle(element).visibility !== 'hidden';
    const color = value => { const values = value.match(/[\d.]+/g)?.map(Number) || []; return values.length >= 3 ? { rgb: values.slice(0, 3), alpha: values[3] ?? 1 } : { rgb: [0, 0, 0], alpha: 0 }; };
    const composite = (front, back) => front.rgb.map((channel, index) => channel * front.alpha + back[index] * (1 - front.alpha));
    const luminance = rgb => rgb.map(channel => channel / 255).map(channel => channel <= .04045 ? channel / 12.92 : ((channel + .055) / 1.055) ** 2.4).reduce((sum, channel, index) => sum + channel * [.2126, .7152, .0722][index], 0);
    const samples = [];
    const elements = [...new Set([
      ...[...document.querySelectorAll('body, .prose p, .prose a, .prose blockquote, .docs-version-context p, .docs-sidebar a, .table-of-contents a, summary')].filter(visible).slice(0, 200),
      ...[...document.querySelectorAll('pre code, pre code span')].filter(visible).slice(0, 500),
      ...[...document.querySelectorAll('.mermaid svg text, .mermaid svg foreignObject div')].filter(visible).slice(0, 200),
    ])];
    for (const element of elements) {
      const ancestors = []; for (let node = element; node; node = node.parentElement) ancestors.unshift(node);
      let background = [255, 255, 255];
      for (const node of ancestors) background = composite(color(getComputedStyle(node).backgroundColor), background);
      const style = getComputedStyle(element);
      const diagramSurface = element.closest('.node,.cluster,.edgeLabel')?.querySelector('rect,polygon,circle');
      if (diagramSurface) background = composite(color(getComputedStyle(diagramSurface).fill), background);
      const foreground = composite(color(element.matches('svg text') ? style.fill : style.color), background);
      const first = luminance(foreground), second = luminance(background);
      const ratio = (Math.max(first, second) + .05) / (Math.min(first, second) + .05);
      const required = parseFloat(style.fontSize) >= 24 || parseFloat(style.fontSize) >= 18.66 && parseInt(style.fontWeight, 10) >= 700 ? 3 : 4.5;
      samples.push({ selector: element.tagName.toLowerCase() + (element.className ? '.' + String(element.className).trim().replaceAll(' ', '.') : ''), ratio: Math.round(ratio * 100) / 100, required });
    }
    const targets = [...document.querySelectorAll('.brand-link, .site-header nav a, .theme-toggle, .docs-sidebar a, .table-of-contents a, summary, .page-navigation a, .docs-version-context a')].filter(visible).map(element => ({ text: (element.getAttribute('aria-label') || element.textContent).trim().slice(0, 60), width: element.getBoundingClientRect().width, height: element.getBoundingClientRect().height }));
    const themeGlyphs = [...document.querySelectorAll('#theme-toggle svg')].filter(visible).map(element => ({ width: element.getBoundingClientRect().width, height: element.getBoundingClientRect().height }));
    return { minimum_contrast: Math.min(...samples.map(sample => sample.ratio)), contrast_samples: samples.length, contrast_failures: samples.filter(sample => sample.ratio + .01 < sample.required), minimum_target_width: Math.min(...targets.map(target => target.width)), minimum_target_height: Math.min(...targets.map(target => target.height)), target_failures: targets.filter(target => target.width < 44 || target.height < 44), theme_glyph_failures: themeGlyphs.filter(glyph => glyph.width < 20 || glyph.height < 20) };
  });
}

export async function checkDocumentationUI(options = parseOptions([])) {
  const started = Date.now();
  const deadline = started + 180000;
  const navigate = (page, url, settings) => ready(page, url, deadline, settings);
  const report = { kind: 'documentation-ui-verification', qualification_limit_seconds: 180, viewport_widths: [320, 360, 390, 768, 1440], themes: ['light', 'dark'], browser: null, isolated_profile: true, public_pages: 0, offline_pages: 0, cases: [], readability: [], screenshots: [], external_requests: [], external_websockets: [], browser_errors: [], cleanup: {}, limitations: ['Automated Chromium checks and captured visual inspection do not certify screen-reader behavior, other browser engines, every assistive technology, or complete WCAG conformance. Contrast checks sample body, link, muted, navigation, syntax and diagram-label text on computed solid surfaces; arbitrary raster-image text and gradients are not certified.'] };
  await mkdir(options.output, { recursive: true });
  const fixtureBase = os.platform() === 'win32' ? path.parse(os.tmpdir()).root : os.tmpdir();
  const fixture = await mkdtemp(path.join(fixtureBase, 'insonic-docs-ui-'));
  let context;
  const contexts = [];
  let server;
  let failure;
  let deadlineExpired = false;
  const qualificationGuard = setTimeout(() => {
    deadlineExpired = true;
    // Only these freshly created contexts belong to this check. Closing them
    // cancels an in-flight browser operation without touching another browser.
    for (const owned of contexts) owned.close().catch(() => {});
  }, Math.max(1, deadline - Date.now()));
  try {
    const publicFiles = await htmlFiles(options.out);
    const offlineFiles = await htmlFiles(options.offline);
    ({ server, origin: report.origin } = await serve({ public: options.out, offline: options.offline }));
    const env = browserEnvironment(fixture, options.browser);
    for (const directory of [env.APPDATA, env.LOCALAPPDATA, env.XDG_CONFIG_HOME, env.XDG_CACHE_HOME]) await mkdir(directory, { recursive: true });
    context = await chromium.launchPersistentContext(path.join(fixture, 'browser-profile'), {
      headless: true, timeout: 25000, ...(options.browser ? { executablePath: options.browser } : {}),
      env, args: ['--disable-background-networking', '--disable-extensions', '--no-first-run', '--no-default-browser-check'],
      viewport: { width: 320, height: 900 }, deviceScaleFactor: 1, locale: 'en-US', colorScheme: 'light', reducedMotion: 'reduce', serviceWorkers: 'block',
    });
    contexts.push(context);
    const browser = context.browser();
    report.browser = browser.version();
    const network = { origin: report.origin, out: options.out, offline: options.offline };
    const configure = async current => {
      await configureBrowserNetwork(current, { ...network, externalRequests: report.external_requests, webSocketAttempts: report.external_websockets });
      current.on('page', page => page.on('pageerror', error => report.browser_errors.push(error.message)));
    };
    await configure(context);
    report.websocket_isolation_probe = await verifyWebSocketIsolation(browser, contexts, network, deadline);
    report.cases.push({ name: 'Owned loopback WebSocket probe is recorded and blocked before any server connection, upgrade or traffic', passed: true });
    const page = await context.newPage();
    const urlFor = (filename, directory, mount) => `${report.origin}/${mount}/${path.relative(directory, filename).split(path.sep).join('/')}`;
    for (const [mount, directory, files] of [['public', options.out, publicFiles], ['offline', options.offline, offlineFiles]]) {
      for (const filename of files) {
        await navigate(page, urlFor(filename, directory, mount));
        const result = await inspectPage(page);
        assertPage(result, `${mount}/${path.relative(directory, filename)}`);
        report[`${mount}_pages`] += 1;
      }
    }
    report.cases.push({ name: 'All exported public and offline pages: 320px reflow, compact navigation, landmarks, names, alternatives and unique IDs', passed: true });
    const manifest = JSON.parse(await readFile(path.join(options.out, 'documentation-manifest.json'), 'utf8'));
    const version = manifest.latest;
    const representatives = ['index', 'architecture', 'contracts', 'changelog', 'windows-installation'].map(slug => `docs/v${version}/${slug}/index.html`);
    const references = [publicFiles.find(filename => filename.includes(`${path.sep}v${version}${path.sep}references${path.sep}`)), publicFiles.find(filename => /[\\/]docs[\\/]v[^\\/]+[\\/]references[\\/]/.test(filename) && !filename.includes(`${path.sep}v${version}${path.sep}`))];
    for (const reference of new Set(references.filter(Boolean))) representatives.push(path.relative(options.out, reference).split(path.sep).join('/'));
    for (const width of report.viewport_widths) for (const theme of report.themes) {
      await page.setViewportSize({ width, height: 900 });
      await page.emulateMedia({ colorScheme: theme, reducedMotion: 'reduce' });
      for (const relative of representatives) {
        await navigate(page, `${report.origin}/public/${relative}`);
        const result = await inspectPage(page);
        assertPage(result, `${relative}: ${width}px/${theme}`, { compact: width < 960 });
        const readability = await inspectReadability(page);
        assert(!readability.contrast_failures.length && !readability.target_failures.length && !readability.theme_glyph_failures.length, `${relative}: ${width}px/${theme}: contrast, control size or theme glyph failure: ${JSON.stringify(readability)}`);
        report.readability.push({ page: relative, width, theme, ...readability });
      }
      report.cases.push({ name: `Article, diagram, contracts, changelog, installation and reference templates: ${width}px/${theme}`, passed: true });
      await navigate(page, `${report.origin}/public/docs/v${version}/windows-installation/index.html`);
      const image = `installation-${width}-${theme}.png`;
      await page.screenshot({ path: path.join(options.output, image) });
      report.screenshots.push(image);
    }
    await page.setViewportSize({ width: 390, height: 900 });
    await navigate(page, `${report.origin}/public/docs/v${version}/index/index.html`);
    await page.keyboard.press('Tab');
    assert(await page.locator('.skip-link').evaluate(element => element === document.activeElement), 'Keyboard: skip link must be the first focus target.');
    await page.keyboard.press('Enter');
    assert(await page.evaluate(() => document.activeElement.id === 'main'), 'Keyboard: skip link must transfer focus to the main landmark.');
    const summary = page.locator('.docs-sidebar > details > summary');
    await summary.focus();
    await page.keyboard.press('Enter');
    assert(await page.locator('.docs-sidebar > details').evaluate(element => element.open), 'Keyboard: Enter must expand documentation navigation.');
    await page.keyboard.press('Tab');
    assert(await page.evaluate(() => document.activeElement.matches('.docs-sidebar a')), 'Keyboard: expanded documentation links must be reachable.');
    const focus = await page.evaluate(() => { const style = getComputedStyle(document.activeElement); return { width: parseFloat(style.outlineWidth), offset: parseFloat(style.outlineOffset), style: style.outlineStyle }; });
    assert(focus.width >= 2 && focus.offset >= 2 && focus.style !== 'none', 'Keyboard: focused links require a visible 2px outline and offset.');
    await summary.focus();
    await page.keyboard.press('Space');
    assert(!await page.locator('.docs-sidebar > details').evaluate(element => element.open), 'Keyboard: Space must collapse documentation navigation.');
    await page.keyboard.press('Tab');
    assert(!await page.evaluate(() => document.activeElement.matches('.docs-sidebar a')), 'Keyboard: collapsed links must leave the focus order.');
    report.cases.push({ name: 'Keyboard: skip focus, Enter/Space disclosure, expanded/collapsed tab order and visible focus', passed: true });
    for (const relative of ['docs/versions/index.html', 'docs/latest/topic-not-found/index.html', '404.html']) {
      try { await stat(path.join(options.out, relative)); } catch { continue; }
      await navigate(page, `${report.origin}/public/${relative}`);
      await page.keyboard.press('Tab'); await page.keyboard.press('Enter');
      await page.waitForFunction(() => document.activeElement.id === 'main', null, { timeout: Math.min(5000, deadline - Date.now()) });
    }
    report.cases.push({ name: 'Archive and unavailable-page skip links reach their main landmark', passed: true });

    const noScript = await browser.newContext({ javaScriptEnabled: false, viewport: { width: 320, height: 900 }, colorScheme: 'dark', reducedMotion: 'reduce', serviceWorkers: 'block' });
    contexts.push(noScript); await configure(noScript);
    const fallback = await noScript.newPage();
    for (const relative of representatives) {
      await navigate(fallback, pathToFileURL(path.join(options.offline, relative)).href, { scripts: false });
      assertPage(await inspectPage(fallback), `${relative}: offline file/no JavaScript`);
    }
    assert(!await fallback.locator('#theme-toggle').isVisible(), 'No JavaScript: an inert theme switch must not be exposed.');
    const fallbackImage = 'offline-no-javascript-320.png';
    await fallback.screenshot({ path: path.join(options.output, fallbackImage) }); report.screenshots.push(fallbackImage);
    const fallbackSummary = fallback.locator('.docs-sidebar > details > summary');
    await fallbackSummary.focus(); await fallback.keyboard.press('Enter');
    assert(await fallback.locator('.docs-sidebar > details').evaluate(element => element.open), 'No JavaScript: native navigation must expand.');
    report.cases.push({ name: 'Offline file URLs with JavaScript disabled: all representative templates and native disclosure', passed: true });
    const expandedImage = 'offline-navigation-expanded-320.png';
    await fallback.screenshot({ path: path.join(options.output, expandedImage) }); report.screenshots.push(expandedImage);

    const denied = await browser.newContext({ viewport: { width: 390, height: 900 }, colorScheme: 'light', reducedMotion: 'reduce', serviceWorkers: 'block' });
    contexts.push(denied); await configure(denied);
    await denied.addInitScript(() => Object.defineProperty(window, 'localStorage', { configurable: true, get() { throw new DOMException('Storage is disabled', 'SecurityError'); } }));
    const preferences = await denied.newPage();
    await navigate(preferences, `${report.origin}/public/docs/v${version}/index/index.html`);
    await preferences.locator('#theme-toggle').click();
    assert(await preferences.locator('#theme-toggle').getAttribute('aria-checked') === 'true', 'Storage denied: theme toggle must still change the current page.');
    assertPage(await inspectPage(preferences), 'Storage denied/theme change');
    report.cases.push({ name: 'Unavailable preference storage preserves current-page theme controls', passed: true });

    await page.setViewportSize({ width: 768, height: 900 });
    await navigate(page, `${report.origin}/public/docs/v${version}/contracts/index.html`);
    await page.evaluate(() => { document.documentElement.style.fontSize = '200%'; });
    assertPage(await inspectPage(page), 'Contracts: 200% text at768px');
    const enlargedReadability = await inspectReadability(page);
    assert(!enlargedReadability.target_failures.length && !enlargedReadability.theme_glyph_failures.length, '200% text: navigation controls and theme glyph must retain their usable size.');
    const enlargedImage = 'contracts-enlarged-text-768.png';
    await page.screenshot({ path: path.join(options.output, enlargedImage) }); report.screenshots.push(enlargedImage);
    await page.evaluate(() => { document.documentElement.style.fontSize = ''; const style = document.createElement('style'); style.textContent = 'body { letter-spacing: .12em !important; word-spacing: .16em !important; line-height: 1.5 !important; } p { margin-bottom: 2em !important; }'; document.head.append(style); });
    assertPage(await inspectPage(page), 'Contracts: user text-spacing overrides at768px');
    report.cases.push({ name: '200% text and WCAG text-spacing overrides retain reading/navigation reflow', passed: true });
    await page.emulateMedia({ forcedColors: 'active', reducedMotion: 'reduce' });
    await navigate(page, `${report.origin}/public/docs/v${version}/index/index.html`);
    assertPage(await inspectPage(page), 'Forced colors and reduced motion');
    report.cases.push({ name: 'Forced colors and reduced motion retain reachable navigation', passed: true });
    assert(!deadlineExpired && Date.now() <= deadline, 'Documentation browser qualification exceeded its180second limit.');
    assert(report.external_requests.length === 0 && report.external_websockets.length === 0 && report.browser_errors.length === 0, `Browser errors or external asset requests: ${JSON.stringify({ requests: report.external_requests, websockets: report.external_websockets, errors: report.browser_errors })}`);
  } catch (error) { failure = error; report.failure = error.message; }
  finally {
    clearTimeout(qualificationGuard);
    report.qualification_elapsed_seconds = Math.round((Date.now() - started) / 100) / 10;
    for (const current of contexts.reverse()) {
      try { await bounded('owned browser cleanup', () => current.close(), 8000); } catch (error) { report.cleanup.browser_error = error.message; }
    }
    report.cleanup.browser_contexts_closed = contexts.length > 0 && !report.cleanup.browser_error;
    if (server) await new Promise(resolve => { server.closeAllConnections(); server.close(resolve); });
    report.cleanup.server_closed = !server?.listening;
    if (report.cleanup.browser_contexts_closed || !context) {
      assert(path.dirname(fixture) === fixtureBase && path.basename(fixture).startsWith('insonic-docs-ui-'), 'Unexpected owned fixture path.');
      try {
        await rm(fixture, { recursive: true, force: true, maxRetries: 10, retryDelay: 200 });
        report.cleanup.owned_fixture_removed = true;
      } catch (error) { report.cleanup.fixture_error = error.message; }
    }
    report.elapsed_seconds = Math.round((Date.now() - started) / 100) / 10;
    delete report.origin;
    report.passed = !failure && report.cleanup.browser_contexts_closed && report.cleanup.server_closed && report.cleanup.owned_fixture_removed;
    await writeFile(path.join(options.output, 'report.json'), JSON.stringify(report, null, 2) + '\n', 'utf8');
  }
  if (failure) throw failure;
  assert(report.passed, 'Documentation browser cleanup did not complete.');
  console.log(`Documentation UI checked: ${report.public_pages} public and ${report.offline_pages} offline pages, ${report.cases.length} cases, ${report.elapsed_seconds}s; ${options.output}`);
  return report;
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) await checkDocumentationUI(parseOptions(process.argv.slice(2)));
