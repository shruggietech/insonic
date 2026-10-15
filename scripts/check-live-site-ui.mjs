// SPDX-License-Identifier: Apache-2.0
import { createHash } from 'node:crypto';
import { request } from 'node:https';
import { mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright-core';
import { assertPage, inspectPage, inspectReadability, landingGeometry, landingInventory, assertLandingGeometry, browserEnvironment } from './check-docs-ui.mjs';

const assert = (ok, message) => { if (!ok) throw new Error(message); };
export async function closeOwnedBrowser(server, timeout = 8000) {
  let timer;
  try {
    await Promise.race([server.close(), new Promise((_, reject) => { timer = setTimeout(() => reject(new Error('Owned browser close deadline exceeded.')), timeout); })]);
  } catch {
    // The server handle owns this exact browser process group, never an image-name search.
    await server.kill();
  } finally { clearTimeout(timer); }
}
export function liveInventory(receipt) {
  assert(Array.isArray(receipt?.paths) && receipt.paths.length > 0 && receipt.paths.length <= 10000, 'Invalid live inventory size.');
  const selected = new Map();
  for (const item of receipt.paths) {
    const url = new URL(item.url);
    assert(url.origin === 'https://insonic.io' && !url.username && !url.password && !url.search && !url.hash && url.href === item.url
      && /^[a-f0-9]{64}$/.test(item.sha256) && !selected.has(url.href), 'Invalid or duplicate live resource identity.');
    selected.set(url.href, item.sha256);
  }
  assert(selected.has('https://insonic.io/'), 'Live landing identity is missing.');
  // Hydrated navigation uses the directory alias; require the same exported bytes.
  if (selected.has('https://insonic.io/docs/latest/index.html') && !selected.has('https://insonic.io/docs/latest/')) {
    selected.set('https://insonic.io/docs/latest/', selected.get('https://insonic.io/docs/latest/index.html'));
  }
  return selected;
}

export function readLive(url, { signal, redirects = 0 } = {}) {
  const selected = new URL(url);
  assert(selected.origin === 'https://insonic.io' && !selected.search && !selected.hash && !selected.username && !selected.password, 'Live origin differs.');
  return new Promise((resolve, reject) => {
    const req = request(selected, { signal, headers: { 'Accept-Encoding': 'identity', 'User-Agent': 'insonic-live-browser' } }, response => {
      if ([301, 302, 303, 307, 308].includes(response.statusCode)) {
        response.resume();
        if (redirects >= 4) { reject(new Error('Live redirect bound exceeded.')); return; }
        try { resolve(readLive(new URL(response.headers.location, selected).href, { signal, redirects: redirects + 1 })); }
        catch { reject(new Error('Live redirect left selected origin.')); }
        return;
      }
      if (response.statusCode !== 200 || (response.headers['content-encoding'] && response.headers['content-encoding'] !== 'identity')) {
        response.resume(); reject(new Error('Live response status/encoding differs.')); return;
      }
      let size = 0; const chunks = [];
      response.on('data', chunk => {
        size += chunk.length;
        if (size > 16 << 20) { response.destroy(new Error('Live resource bound exceeded.')); return; }
        chunks.push(chunk);
      });
      response.on('error', reject);
      response.on('end', () => resolve({ body: Buffer.concat(chunks), type: response.headers['content-type'] || 'application/octet-stream' }));
    });
    req.on('error', () => reject(new Error('Bounded live request failed.')));
    req.end();
  });
}

export async function checkLiveSite({ receipt, output, executable = '/usr/bin/google-chrome', reader = readLive }) {
  const selected = liveInventory(receipt), cache = new Map();
  const started = Date.now(), controller = new AbortController();
  let bytes = 0, context, server, fixture, failure;
  const guard = setTimeout(() => { controller.abort(); if (server) void server.kill().catch(() => {}); }, 180000);
  const report = { kind: 'live-website-browser-verification', method: 'bounded-live-response-rendering',
    site_revision: receipt.revision, release_revision: receipt.release_revision, pages_commit: receipt.pages_commit,
    version: receipt.version, cases: [], resources: [], blocked_requests: 0, cleanup: {}, passed: false };
  const fetch = async url => {
    assert(selected.has(url), 'Browser resource is absent from the verified live inventory.');
    if (!cache.has(url)) {
      assert(cache.size < 200, 'Browser resource count bound exceeded.');
      const pending = (async () => {
        const value = await reader(url, { signal: controller.signal });
        const digest = createHash('sha256').update(value.body).digest('hex');
        assert(digest === selected.get(url), 'Live browser resource digest differs.');
        bytes += value.body.length; assert(bytes <= 64 << 20, 'Browser response total bound exceeded.');
        report.resources.push({ url, sha256: digest, size_bytes: value.body.length });
        return value;
      })();
      cache.set(url, pending);
    }
    return cache.get(url);
  };
  try {
    // Reject stale landing before launching any browser.
    await fetch('https://insonic.io/');
    await mkdir(output, { recursive: true });
    fixture = await mkdtemp(path.join(os.tmpdir(), 'insonic-live-ui-'));
    server = await chromium.launchServer({ executablePath: executable, headless: true, host: '127.0.0.1',
      env: browserEnvironment(fixture, executable), downloadsPath: path.join(fixture, 'downloads'), timeout: 20000,
      args: ['--disable-background-networking', '--disable-component-update', '--disable-sync', '--no-first-run'] });
    const browser = await chromium.connect(server.wsEndpoint(), { timeout: 10000 });
    context = await browser.newContext({ acceptDownloads: false, serviceWorkers: 'block' });
    context.setDefaultTimeout(15000);
    await context.addInitScript(() => { localStorage.setItem('insonic-theme', matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'); });
    await context.routeWebSocket('**', route => { report.blocked_requests++; route.close(); });
    await context.route('**/*', async route => {
      try {
        const req = route.request();
        if (req.method() !== 'GET' || !selected.has(req.url())) { report.blocked_requests++; await route.abort(); return; }
        const value = await fetch(req.url());
        await route.fulfill({ status: 200, contentType: value.type, body: value.body });
      } catch (error) { failure ??= error; await route.abort().catch(() => {}); }
    });
    const page = context.pages()[0] || await context.newPage();
    for (const width of [320, 390, 768, 1440]) for (const theme of ['light', 'dark']) {
      assert(!controller.signal.aborted, 'Live browser aggregate deadline exceeded.');
      await page.setViewportSize({ width, height: 1000 });
      await page.emulateMedia({ colorScheme: theme, reducedMotion: 'reduce' });
      await page.goto('https://insonic.io/', { waitUntil: 'networkidle', timeout: Math.min(20000, Math.max(1, 180000 - (Date.now() - started))) });
      await page.evaluate(() => document.fonts.ready);
      const item = { width, theme, image: `${width}-${theme}.png`, passed: false };
      report.cases.push(item);
      try {
        item.actual_theme = await page.evaluate(() => document.documentElement.dataset.theme);
        item.page = await inspectPage(page); assertPage(item.page, `${width}/${theme}`);
        item.geometry = await landingGeometry(page); assertLandingGeometry(item.geometry, `${width}/${theme}`);
        item.readability = await inspectReadability(page);
        assert(!item.readability.contrast_failures.length && !item.readability.target_failures.length, 'Live contrast/target checks failed.');
        item.links = await landingInventory(page, receipt.version);
        const hrefs = new Set(item.links.map(link => link.resolved_href));
        const latest = ['https://insonic.io/docs/latest/', 'https://insonic.io/docs/latest/index.html'].find(url => hrefs.has(url) && selected.has(url));
        assert(latest, 'Live latest documentation link differs.');
        await fetch(latest);
        for (const pack of receipt.packages ?? []) assert(hrefs.has(pack.url), 'Live package link differs.');
        assert(item.actual_theme === theme, 'Actual theme differs.');
        item.passed = true;
      } finally { await page.screenshot({ path: path.join(output, item.image), fullPage: true, timeout: 10000 }); }
    }
    if (failure) throw failure;
    assert(report.cases.length === 8 && report.cases.every(item => item.passed), 'Incomplete live browser matrix.');
    report.passed = true;
  } catch (error) { failure = error; report.failure = error.message; }
  finally {
    clearTimeout(guard); controller.abort();
    if (server) {
      try { await closeOwnedBrowser(server); report.cleanup.browser_closed = true; }
      catch { report.cleanup.browser_closed = false; report.passed = false; }
    }
    if (fixture && (!context || report.cleanup.browser_closed)) {
      await rm(fixture, { recursive: true, force: true, maxRetries: 5 }); report.cleanup.owned_profile_removed = true;
    }
    report.elapsed_seconds = (Date.now() - started) / 1000;
    await mkdir(output, { recursive: true });
    await writeFile(path.join(output, 'report.json'), JSON.stringify(report, null, 2) + '\n', 'utf8');
  }
  if (failure) throw failure;
  assert(report.passed, 'Live browser cleanup failed.');
  return report;
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const args = process.argv.slice(2), options = {};
  for (let i = 0; i < args.length; i += 2) options[args[i].replace(/^--/, '')] = args[i + 1];
  const receipt = JSON.parse(await readFile(options.receipt, 'utf8'));
  await checkLiveSite({ receipt, output: options.output, executable: options.browser });
}
