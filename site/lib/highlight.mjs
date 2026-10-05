// SPDX-License-Identifier: Apache-2.0
import hljs from 'highlight.js';
import bash from 'highlight.js/lib/languages/bash';

// Shell syntax remains intact; add the executable, flags and placeholders used
// by the CLI contract so examples are readable beyond quoted string literals.
hljs.registerLanguage('insonic-shell', engine => {
  const shell = bash(engine);
  return { ...shell, name: 'insonic shell', contains: [
    ...shell.contains,
    { scope: 'title', match: /\binsonic\b/ },
    { scope: 'attr', match: /--[a-z][a-z0-9-]*/ },
    { scope: 'variable', match: /\b[A-Z][A-Z0-9_]+\b/ },
  ] };
});

export function highlightSource(code, language) {
  const grammar = ['sh', 'bash', 'shell'].includes(language) && /\binsonic\b/.test(code) ? 'insonic-shell' : language;
  if (!grammar || !hljs.getLanguage(grammar)) return null;
  return hljs.highlight(code, { language: grammar, ignoreIllegals: true }).value;
}
