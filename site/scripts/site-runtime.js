// SPDX-License-Identifier: Apache-2.0
import { applyTheme, normalizeThemePreference, readThemePreference, themeStorageKey } from '../lib/theme.mjs';

// Explicit index files keep offline help portable; HTTP hosts use directory URLs.
if (['http:', 'https:'].includes(window.location.protocol)) {
  for (const anchor of document.querySelectorAll('a[href]')) {
    const destination = new URL(anchor.href);
    if (destination.origin === window.location.origin && destination.pathname.endsWith('/index.html')) {
      destination.pathname = destination.pathname.slice(0, -'index.html'.length);
      anchor.href = destination.href;
    }
  }
}

const theme = window.matchMedia('(prefers-color-scheme: dark)');
const themeToggle = document.querySelector('#theme-toggle');
let preference = normalizeThemePreference(document.documentElement.dataset.themePreference);
function updateTheme(next, persist = false) {
  preference = normalizeThemePreference(next);
  const previous = document.documentElement.dataset.theme;
  const resolved = applyTheme(document, preference, theme.matches);
  if (themeToggle) {
    themeToggle.setAttribute('aria-checked', String(resolved === 'dark'));
    themeToggle.title = `Switch to ${resolved === 'dark' ? 'light' : 'dark'} theme`;
  }
  if (persist) { try { window.localStorage.setItem(themeStorageKey, preference); } catch { /* The current page still changes when storage is unavailable. */ } }
  if (previous !== resolved) window.dispatchEvent(new Event('insonic-themechange'));
}
if (themeToggle) themeToggle.addEventListener('click', () => updateTheme(document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark', true));
theme.addEventListener('change', () => { if (preference === 'system') updateTheme('system'); });
window.addEventListener('storage', event => {
  if (event.key !== themeStorageKey && event.key !== null) return;
  let next = 'system';
  try { next = readThemePreference(window.localStorage); } catch { /* Default to the system preference. */ }
  updateTheme(next);
});

const previewTabs = [...document.querySelectorAll('[data-preview-tab]')];
function selectPreview(tab, focus = false) {
  for (const item of previewTabs) { item.setAttribute('aria-selected', String(item === tab)); item.tabIndex = item === tab ? 0 : -1; }
  for (const panel of document.querySelectorAll('[data-preview-panel]')) {
    panel.hidden = panel.dataset.previewPanel !== tab.dataset.previewTab;
    panel.inert = panel.hidden;
  }
  if (focus) tab.focus();
}
for (const [index, tab] of previewTabs.entries()) {
  tab.addEventListener('click', () => selectPreview(tab));
  tab.addEventListener('keydown', event => {
    const destination = event.key === 'ArrowRight' ? (index + 1) % previewTabs.length : event.key === 'ArrowLeft' ? (index + previewTabs.length - 1) % previewTabs.length : event.key === 'Home' ? 0 : event.key === 'End' ? previewTabs.length - 1 : null;
    if (destination === null) return;
    event.preventDefault();
    selectPreview(previewTabs[destination], true);
  });
}

// Reconcile an OS change between the blocking initializer and this deferred script.
updateTheme(preference);
