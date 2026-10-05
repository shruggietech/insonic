// SPDX-License-Identifier: Apache-2.0
import { applyTheme, readThemePreference } from '../lib/theme.mjs';
let preference = 'system';
try { preference = readThemePreference(window.localStorage); } catch { /* Storage can be unavailable in an offline viewer. */ }
applyTheme(document, preference, window.matchMedia('(prefers-color-scheme: dark)').matches);
