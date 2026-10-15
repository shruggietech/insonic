// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import path from 'node:path';
import vm from 'node:vm';
import { readFile } from 'node:fs/promises';
import { boundedPath, parseOptions } from '../scripts/check-docs-ui.mjs';

test('documentation server confines public assets to its selected export', () => {
  const directory = path.resolve('site/out');
  assert.equal(boundedPath(directory, 'docs/latest/index.html'), path.join(directory, 'docs/latest/index.html'));
  for (const attempt of ['../README.md', '../../private.json', path.resolve('site/outside/file')]) assert.throws(() => boundedPath(directory, attempt), /leaves/);
});

test('browser verification rejects missing, repeated and unsupported arguments', () => {
  assert.equal(parseOptions(['--browser', './chromium', '--output', './build/ui']).browser, path.resolve('chromium'));
  for (const args of [['--browser'], ['--output', '--out'], ['--browser', 'a', '--browser', 'b'], ['--visible', 'true']]) assert.throws(() => parseOptions(args), /Use/);
});

test('responsive native navigation preserves reader choices and focus through width changes', async () => {
  const source = await readFile(new URL('../site/scripts/site-runtime.js', import.meta.url), 'utf8');
  const scoped = source.slice(source.indexOf('for (const disclosure of document.querySelectorAll'));
  const callbacks = {};
  const listeners = {};
  const summary = { focused: false, focus() { this.focused = true; } };
  const child = {};
  const disclosure = { dataset: { responsiveDisclosure: '960' }, open: false, addEventListener(name, callback) { listeners[name] = callback; }, contains(element) { return element === child; }, querySelector() { return summary; } };
  const media = { matches: true, addEventListener(name, callback) { callbacks[name] = callback; } };
  const document = { activeElement: null, querySelectorAll() { return [disclosure]; } };
  vm.runInNewContext(scoped, { document, window: { matchMedia() { return media; } } });
  assert.equal(disclosure.open, true);
  media.matches = false; callbacks.change(); assert.equal(disclosure.open, false);
  disclosure.open = true; listeners.toggle(); // Reader expands the narrow layout.
  media.matches = true; callbacks.change(); assert.equal(disclosure.open, true);
  disclosure.open = false; listeners.toggle(); // Independent wide-layout choice.
  media.matches = false; callbacks.change(); assert.equal(disclosure.open, true);
  document.activeElement = child;
  media.matches = true; callbacks.change();
  assert.equal(disclosure.open, false);
  assert.equal(summary.focused, true, 'collapsing must not strand focus in hidden navigation');
});
