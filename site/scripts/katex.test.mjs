// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict';
import test from 'node:test';
import { createRequire } from 'node:module';

// Resolve the actual renderer dependency, including nested override installs.
const katex = createRequire(import.meta.resolve('mermaid'))('katex');

test('KaTeX ignores inherited trust and preserves explicit own trust', () => {
  const expression = String.raw`\href{https://example.invalid/}{example}`;
  assert.doesNotMatch(katex.renderToString(expression, {}), /<a[ >]/);
  assert.doesNotMatch(katex.renderToString(expression, Object.create({ trust: true })), /<a[ >]/);
  assert.match(katex.renderToString(expression, { trust: true }), /<a[ >]/);
});
