// SPDX-License-Identifier: Apache-2.0
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import MarkdownIt from 'markdown-it';
import taskLists from 'markdown-it-task-lists';
import { highlightSource } from './highlight.mjs';
import { loadSchemaCatalog, renderSchemaReference } from '../../scripts/schema-catalog.mjs';

export const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
export const docsRoot = path.join(root, 'docs');

export function getManifest() {
  const manifest = JSON.parse(fs.readFileSync(path.join(docsRoot, 'versions.json'), 'utf8'));
  if (!manifest.versions?.length || !manifest.versions.some((entry) => entry.version === manifest.latest)) throw new Error('docs/versions.json must identify an existing latest version.');
  const seen = new Set();
  for (const entry of manifest.versions) {
    if (!/^(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$/.test(entry.version) || seen.has(entry.version)) throw new Error(`Invalid or duplicate documentation version: ${entry.version}`);
    seen.add(entry.version);
    if (!entry.pages?.length) throw new Error(`Documentation v${entry.version} has no pages.`);
    const slugs = new Set();
    for (const page of entry.pages) {
      if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(page.slug) || slugs.has(page.slug) || !page.title) throw new Error(`Invalid page in documentation v${entry.version}: ${page.slug}`);
      slugs.add(page.slug);
    }
  }
  return manifest;
}

export function versionPath(version, slug) {
  return `/docs/v${version}/${slug}/`;
}

export function referencePath(version, slug) {
  return `/docs/v${version}/references/${slug}/`;
}

export function getRepositoryReferences() {
  const references = [];
  for (const entry of getManifest().versions) {
    const paths = new Set();
    for (const page of entry.pages) {
      if (page.slug === 'changelog') continue;
      const source = fs.readFileSync(path.join(docsRoot, `v${entry.version}`, `${page.slug}.md`), 'utf8');
      const tokens = new MarkdownIt().parse(source, {});
      for (const token of tokens) for (const child of token.children || []) {
        if (child.type !== 'link_open') continue;
        const href = child.attrGet('href');
        if (!href?.startsWith('../')) continue;
        const repositoryPath = path.posix.normalize(`docs/v${entry.version}/${href.split('#')[0]}`);
        if (repositoryPath.startsWith('../')) throw new Error(`Documentation link leaves the repository: ${href}`);
        paths.add(repositoryPath);
      }
    }
    for (const repositoryPath of paths) {
      const snapshot = path.join(docsRoot, `v${entry.version}`, 'references', repositoryPath);
      const filename = fs.existsSync(snapshot) ? snapshot : path.join(root, repositoryPath);
      if (entry.status !== 'specification' && !fs.existsSync(snapshot)) throw new Error(`Release documentation must freeze its reference: ${snapshot}`);
      const slug = repositoryPath.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
      if (references.some((item) => item.entry.version === entry.version && item.slug === slug)) throw new Error(`Duplicate repository reference route: ${repositoryPath}`);
      references.push({ entry, repositoryPath, slug, source: fs.readFileSync(filename, 'utf8') });
    }
  }
  return references;
}

export function readDocument(version, slug) {
  const entry = getManifest().versions.find((item) => `v${item.version}` === version);
  const page = entry?.pages.find((item) => item.slug === slug);
  if (!page) return null;
  let source = fs.readFileSync(path.join(docsRoot, `v${entry.version}`, `${slug}.md`), 'utf8');
  if (slug === 'contracts') source = source.replace('<!-- schema-reference -->', renderSchemaReference(loadSchemaCatalog(entry.version)));
  if (slug === 'changelog') {
    const changelog = entry.status === 'specification' ? path.join(root, 'CHANGELOG.md') : path.join(docsRoot, `v${entry.version}`, 'references/CHANGELOG.md');
    if (!fs.existsSync(changelog)) throw new Error(`Release documentation must freeze its changelog: ${changelog}`);
    source = fs.readFileSync(changelog, 'utf8');
  }
  return { entry, page, source };
}

export function renderDocument(source, entry, repositoryPath) {
  const headings = [];
  const identifiers = new Map();
  const md = new MarkdownIt({
    html: false,
    linkify: true,
    typographer: false,
    highlight(code, language) {
      return highlightSource(code, language) ?? md.utils.escapeHtml(code);
    },
  }).use(taskLists);
  const defaultFence = md.renderer.rules.fence;
  md.renderer.rules.fence = (tokens, index, options, environment, renderer) => {
    const token = tokens[index];
    if (token.info.trim() !== 'mermaid') return defaultFence(tokens, index, options, environment, renderer);
    return `<figure class="diagram"><div class="mermaid" role="img" aria-label="System diagram">${md.utils.escapeHtml(token.content)}</div><details><summary>Diagram source</summary><pre><code>${md.utils.escapeHtml(token.content)}</code></pre></details></figure>`;
  };
  md.core.ruler.push('documentation-links-and-headings', (state) => {
    for (let index = 0; index < state.tokens.length; index += 1) {
      const token = state.tokens[index];
      if (token.type === 'heading_open') {
        const title = state.tokens[index + 1].content;
        const base = title.toLowerCase().replace(/[^\p{L}\p{N}\s-]/gu, '').trim().replace(/\s+/g, '-') || 'section';
        const count = identifiers.get(base) || 0;
        identifiers.set(base, count + 1);
        const id = count ? `${base}-${count}` : base;
        token.attrSet('id', id);
        if (token.tag === 'h2') headings.push({ id, title });
      }
      for (const child of token.children || []) {
        if (child.type !== 'link_open') continue;
        const href = child.attrGet('href');
        if (!href || /^(?:[a-z]+:|\/\/|#)/i.test(href)) continue;
        const match = !repositoryPath && href.match(/^\.?\/?([^/#?]+)\.md(#.*)?$/);
        if (match) {
          if (!entry.pages.some((page) => page.slug === match[1])) throw new Error(`Unlisted documentation link: ${href}`);
          child.attrSet('href', `${versionPath(entry.version, match[1])}${match[2] || ''}`);
        } else if (href.startsWith('../') || repositoryPath) {
          const [pathname, fragment = ''] = href.split('#');
          const targetPath = path.posix.normalize(path.posix.join(repositoryPath ? path.posix.dirname(repositoryPath) : `docs/v${entry.version}`, pathname));
          if (targetPath.startsWith('../')) throw new Error(`Documentation link leaves the repository: ${href}`);
          const targetReference = getRepositoryReferences().find((item) => item.entry.version === entry.version && item.repositoryPath === targetPath);
          if (targetReference) {
            child.attrSet('href', `${referencePath(entry.version, targetReference.slug)}${fragment ? `#${fragment}` : ''}`);
            continue;
          }
          const docsLink = targetPath.match(/^docs\/v([^/]+)\/([^/]+)\.md$/);
          if (docsLink && getManifest().versions.some((version) => version.version === docsLink[1] && version.pages.some((page) => page.slug === docsLink[2]))) {
            child.attrSet('href', `${versionPath(docsLink[1], docsLink[2])}${fragment ? `#${fragment}` : ''}`);
            continue;
          }
          const reference = entry.status === 'specification' ? 'main' : `v${entry.version}`;
          child.attrSet('href', `https://github.com/shruggietech/insonic/blob/${reference}/${targetPath}${fragment ? `#${fragment}` : ''}`);
        }
      }
    }
  });
  return { html: md.render(source.replace(/^---\r?\n[\s\S]*?\r?\n---\r?\n/, '')), headings };
}
