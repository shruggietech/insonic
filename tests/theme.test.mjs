// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict';
import test from 'node:test';
import { applyTheme, readThemePreference, resolveTheme } from '../site/lib/theme.mjs';

test('explicit light and dark preferences override the opposite system theme', () => {
  assert.equal(resolveTheme('light', true), 'light');
  assert.equal(resolveTheme('dark', false), 'dark');
  assert.equal(resolveTheme('system', true), 'dark');
  assert.equal(resolveTheme('system', false), 'light');
});

test('missing, corrupt or denied preference storage falls back to the system', () => {
  assert.equal(readThemePreference({ getItem: () => null }), 'system');
  assert.equal(readThemePreference({ getItem: () => 'unexpected-value' }), 'system');
  assert.equal(readThemePreference({ getItem: () => { throw new Error('Storage unavailable'); } }), 'system');
  assert.equal(readThemePreference({ getItem: () => 'dark' }), 'dark');
});

test('resolved theme and saved intent remain distinct when the system changes', () => {
  const document = { documentElement: { dataset: {}, style: {} } };
  applyTheme(document, 'system', false);
  assert.equal(document.documentElement.dataset.theme, 'light');
  assert.equal(document.documentElement.dataset.themePreference, 'system');
  applyTheme(document, 'system', true);
  assert.equal(document.documentElement.dataset.theme, 'dark');
  assert.equal(document.documentElement.style.colorScheme, 'dark');
  applyTheme(document, 'light', true);
  assert.equal(document.documentElement.dataset.theme, 'light');
  assert.equal(document.documentElement.dataset.themePreference, 'light');
});
