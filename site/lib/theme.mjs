// SPDX-License-Identifier: Apache-2.0
export const themeStorageKey = 'insonic-theme';
export function normalizeThemePreference(value) { return ['light', 'dark', 'system'].includes(value) ? value : 'system'; }
export function resolveTheme(preference, systemDark) { return preference === 'light' || preference === 'dark' ? preference : systemDark ? 'dark' : 'light'; }
export function readThemePreference(storage) {
  try { return normalizeThemePreference(storage.getItem(themeStorageKey)); } catch { return 'system'; }
}
export function applyTheme(document, preference, systemDark) {
  const selected = normalizeThemePreference(preference);
  const resolved = resolveTheme(selected, systemDark);
  document.documentElement.dataset.themePreference = selected;
  document.documentElement.dataset.theme = resolved;
  document.documentElement.style.colorScheme = resolved;
  return resolved;
}
