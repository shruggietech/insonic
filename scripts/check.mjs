#!/usr/bin/env node
// SPDX-License-Identifier: Apache-2.0
import { readFile, readdir, lstat } from 'node:fs/promises';
import { dirname, extname, join, relative, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';
import { verifyInstalled } from './sync-brand-kit.mjs';
import { parseDocument } from 'yaml';

const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const ignoredDirectories = new Set(['.git', '.local', 'node_modules', '.next', 'out', 'offline', 'build', 'dist', 'coverage', 'test-results', 'playwright-report', '.bootstrap-spec-kit', '.brandbuilder']);
const generatedPaths = new Set(['site/public/brand', 'site/public/mermaid', 'site/public/source', 'site/public/project', 'site/public/schemas', 'desktop/assets/help', 'desktop/assets/bundle']);
const textExtensions = new Set(['.md', '.mdx', '.mjs', '.cjs', '.js', '.jsx', '.ts', '.tsx', '.json', '.yml', '.yaml', '.css', '.html', '.txt', '.toml', '.ini', '.sh', '.ps1', '.svg', '.xml', '.sql', '.go']);
const textNames = new Set(['VERSION', 'LICENSE', 'NOTICE', '.editorconfig', '.gitattributes', '.gitignore', '.npmrc', '.node-version', 'go.mod', 'go.sum', 'Makefile']);
const location = (file, line, message) => `${file}${line ? `:${line}` : ''}: ${message}`;

function vendorStyle(file) {
  return file.startsWith('brand/kit/') || file.startsWith('.agents/') || (file.startsWith('.specify/') && file !== '.specify/memory/constitution.md');
}

export function validateText(file, bytes) {
  const problems = [];
  if (bytes.length >= 3 && bytes[0] === 0xef && bytes[1] === 0xbb && bytes[2] === 0xbf) problems.push(location(file, 0, 'UTF-8 BOM is forbidden'));
  let text;
  try { text = new TextDecoder('utf-8', { fatal: true }).decode(bytes); } catch { return { problems: [...problems, location(file, 0, 'Invalid UTF-8 bytes')], text: null }; }
  const lines = text.split('\n'); let reportedLineEndings = false;
  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index];
    if (line.includes('\r') && !reportedLineEndings) { problems.push(location(file, index + 1, 'Use LF line endings')); reportedLineEndings = true; }
    if (/\uFFFD|\u00C3[\u0080-\u00BF]|\u00C2[\u0080-\u00BF]|\u00E2(?:\u0080|\u20AC|\u2122)/u.test(line)) problems.push(location(file, index + 1, 'Replacement character or likely mojibake'));
    if (/^\s*(?:<{7}|={7}|>{7})(?:\s|$)/.test(line)) problems.push(location(file, index + 1, 'Unresolved merge conflict marker'));
  }
  return { problems, text };
}

export function validateYAML(file, text) {
  const document = parseDocument(text, { version: '1.2', prettyErrors: false, uniqueKeys: true });
  return document.errors.map(error => location(file, 0, `Invalid YAML (${error.message})`));
}

function markdownBlocks(text) {
  const lines = text.replace(/\r\n?/g, '\n').replace(/<!--[\s\S]*?-->/g, (comment) => comment.replace(/[^\n]/g, ' ')).split('\n');
  const prose = []; const headings = []; const fences = []; let fence = null;
  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index]; const match = line.match(/^ {0,3}(`{3,}|~{3,})(.*)$/);
    if (fence) {
      if (match && match[1][0] === fence.marker[0] && match[1].length >= fence.marker.length && !match[2].trim()) { fences.push(fence); fence = null; }
      else fence.lines.push(line);
      prose.push(''); continue;
    }
    if (match) { fence = { marker: match[1], language: match[2].trim().split(/\s+/)[0], line: index + 1, lines: [] }; prose.push(''); continue; }
    prose.push(line);
    const heading = line.match(/^ {0,3}(#{1,6})(?:\s+|$)(.*)$/);
    if (heading) headings.push({ level: heading[1].length, title: heading[2].replace(/\s+#+\s*$/, ''), line: index + 1 });
  }
  return { prose: prose.join('\n'), headings, fences, unclosed: fence };
}

export function markdownAnchors(text) {
  const { prose, headings } = markdownBlocks(text); const anchors = new Set(); const counts = new Map();
  for (const heading of headings) {
    const label = heading.title.replace(/\[([^\]]+)\]\([^)]*\)/g, '$1').replace(/<[^>]*>/g, '').replace(/[`*_~]/g, '');
    const base = label.toLowerCase().replace(/[^\p{L}\p{N}\s_-]/gu, '').replace(/\s/g, '-');
    const count = counts.get(base) ?? 0; counts.set(base, count + 1);
    anchors.add(count ? `${base}-${count}` : base);
  }
  for (const match of prose.matchAll(/\b(?:id|name)\s*=\s*["']([^"']+)["']/g)) anchors.add(match[1]);
  return anchors;
}

export function validateMarkdown(file, text) {
  const problems = []; const { headings, fences, unclosed } = markdownBlocks(text);
  const titles = headings.filter((heading) => heading.level === 1);
  if (titles.length !== 1) problems.push(location(file, 0, `Markdown needs exactly one H1 (found ${titles.length})`));
  if (unclosed) problems.push(location(file, unclosed.line, 'Unclosed code fence'));
  for (const fence of fences) {
    if (!/^[A-Za-z0-9][A-Za-z0-9_+.-]*$/.test(fence.language)) problems.push(location(file, fence.line, 'Code fence needs a language'));
    if (fence.language.toLowerCase() === 'mermaid') {
      const chart = fence.lines.join('\n');
      const graph = chart.match(/^\s*(?:flowchart|graph)\b([^\n]*)/m);
      if (graph && !/^\s+(?:TB|TD)\b/.test(graph[1])) problems.push(location(file, fence.line, 'Mermaid flowcharts must use TB or TD'));
      if (/^\s*direction\s+(?!(?:TB|TD)\b)/m.test(chart)) problems.push(location(file, fence.line, 'Mermaid subgraphs must flow top to bottom'));
    }
  }
  return problems;
}

export function markdownTargets(text) {
  const { prose } = markdownBlocks(text);
  const source = prose.replace(/`+[^`\n]*`+/g, (code) => ' '.repeat(code.length));
  const targets = [];
  for (const match of source.matchAll(/!?\[[^\]\n]*\]\(\s*/g)) {
    let offset = match.index + match[0].length; let target = ''; let depth = 0;
    if (source[offset] === '<') { const end = source.indexOf('>', offset + 1); if (end >= 0) target = source.slice(offset + 1, end); }
    else {
      for (; offset < source.length; offset += 1) {
        const char = source[offset];
        if (char === '\\' && offset + 1 < source.length) { target += source[++offset]; continue; }
        if (char === '(') depth += 1;
        if (char === ')') { if (!depth) break; depth -= 1; }
        if (/\s/.test(char) && !depth) break;
        target += char;
      }
    }
    if (target) targets.push({ target, line: source.slice(0, match.index).split('\n').length });
  }
  for (const match of source.matchAll(/^ {0,3}\[[^\]\n]+\]:\s*(?:<([^>]+)>|(\S+))/gm)) targets.push({ target: match[1] ?? match[2], line: source.slice(0, match.index).split('\n').length });
  for (const match of source.matchAll(/\b(?:href|src|srcset)\s*=\s*["']([^"']+)["']/g)) {
    const target = match[1].trim().split(/\s+/)[0];
    if (target) targets.push({ target, line: source.slice(0, match.index).split('\n').length });
  }
  return targets;
}

export async function validateLinks(root, file, text) {
  const problems = [];
  // Frozen repository references retain their original link base, matching
  // the documentation renderer rather than their physical snapshot directory.
  const reference = file.match(/^docs\/v[^/]+\/references\/(.+)$/);
  const linkBase = reference ? reference[1] : file;
  for (const { target, line } of markdownTargets(text)) {
    if (/^(?:[a-z][a-z0-9+.-]*:|\/)/i.test(target)) continue;
    let path; let anchor;
    try { const hash = target.indexOf('#'); path = decodeURIComponent((hash >= 0 ? target.slice(0, hash) : target).split('?')[0]); anchor = hash >= 0 ? decodeURIComponent(target.slice(hash + 1)) : ''; } catch { problems.push(location(file, line, `Malformed link: ${target}`)); continue; }
    let destination = path ? resolve(root, dirname(linkBase), path) : resolve(root, file);
    const within = relative(resolve(root), destination);
    if (within === '..' || within.startsWith(`..${sep}`)) { problems.push(location(file, line, `Relative link leaves the repository: ${target}`)); continue; }
    try {
      const stat = await lstat(destination);
      if (stat.isSymbolicLink()) { problems.push(location(file, line, `Relative link points to a symlink: ${target}`)); continue; }
      if (stat.isDirectory() && anchor) {
        let found = false;
        for (const name of ['README.md', 'index.md']) { try { await lstat(join(destination, name)); destination = join(destination, name); found = true; break; } catch (error) { if (error.code !== 'ENOENT') throw error; } }
        if (!found) { problems.push(location(file, line, `Directory link has no Markdown anchor source: ${target}`)); continue; }
      }
      if (anchor && ['.md', '.markdown'].includes(extname(destination).toLowerCase())) {
        const destinationText = new TextDecoder('utf-8', { fatal: true }).decode(await readFile(destination));
        if (!markdownAnchors(destinationText).has(anchor)) problems.push(location(file, line, `Missing Markdown anchor: ${target}`));
      }
    } catch (error) {
      problems.push(location(file, line, error.code === 'ENOENT' ? `Missing relative link target: ${target}` : `Cannot read relative link ${target}: ${error.message}`));
    }
  }
  return problems;
}

export async function validateVersions(root) {
  const problems = [];
  let version; let navigation;
  try { version = (await readFile(join(root, 'VERSION'), 'utf8')).trim(); navigation = JSON.parse(await readFile(join(root, 'docs', 'versions.json'), 'utf8')); } catch (error) { return [`Documentation version inputs cannot be read: ${error.message}`]; }
  if (!navigation || typeof navigation !== 'object' || Array.isArray(navigation)) return ['docs/versions.json: Expected a navigation object'];
  if (!/^\d+\.\d+\.\d+$/.test(version)) problems.push('VERSION: Expected a semantic version');
  if (navigation.latest !== version) problems.push(`docs/versions.json: latest ${navigation.latest} does not match VERSION ${version}`);
  if (!Array.isArray(navigation.versions) || !navigation.versions.length) return [...problems, 'docs/versions.json: versions must be a nonempty array'];
  const versions = new Set();
  for (const entry of navigation.versions) {
    if (!entry || typeof entry !== 'object') { problems.push('docs/versions.json: Invalid version entry'); continue; }
    if (!/^\d+\.\d+\.\d+$/.test(entry.version ?? '') || versions.has(entry.version)) { problems.push('docs/versions.json: Invalid or duplicate version entry'); continue; }
    versions.add(entry.version);
    if (!Array.isArray(entry.pages) || !entry.pages.length) { problems.push(`docs/versions.json: ${entry.version} needs navigation pages`); continue; }
    const pages = new Set();
    for (const page of entry.pages) {
      if (!page || typeof page !== 'object') { problems.push(`docs/versions.json: ${entry.version} has an invalid navigation page`); continue; }
      if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(page.slug ?? '') || pages.has(page.slug) || typeof page.title !== 'string' || !page.title.trim()) { problems.push(`docs/versions.json: ${entry.version} has an invalid or duplicate navigation page`); continue; }
      pages.add(page.slug);
      const file = `docs/v${entry.version}/${page.slug}.md`;
      try { const stat = await lstat(join(root, file)); if (!stat.isFile() || stat.isSymbolicLink()) problems.push(`${file}: Navigation page must be a regular file`); } catch (error) { problems.push(`${file}: Navigation page is missing (${error.code ?? error.message})`); }
    }
  }
  if (!versions.has(navigation.latest)) problems.push('docs/versions.json: latest has no version entry');
  return problems;
}

async function listProjectFiles(root, prefix = '', result = []) {
  for (const entry of await readdir(join(root, prefix), { withFileTypes: true })) {
    const file = prefix ? `${prefix}/${entry.name}` : entry.name;
    if (entry.name.startsWith('.preview-') || generatedPaths.has(file)) continue;
    if (entry.isDirectory()) { if (!ignoredDirectories.has(entry.name) && file !== 'brand/kit' && file !== '.agents') await listProjectFiles(root, file, result); }
    else if (entry.isSymbolicLink()) result.push({ file, symlink: true });
    else if (entry.isFile()) result.push({ file, symlink: false });
  }
  return result;
}

export async function checkRepository(root = repositoryRoot, { verifyBrand = true } = {}) {
  const problems = []; let checked = 0;
  if (Number(process.versions.node.split('.')[0]) < 22) problems.push('Node.js 22 or newer is required');
  for (const { file, symlink } of await listProjectFiles(root)) {
    if (symlink) { if (!vendorStyle(file)) problems.push(`${file}: Project files must not be symlinks`); continue; }
    const extension = extname(file).toLowerCase(); const json = extension === '.json';
    if ((!textExtensions.has(extension) && !textNames.has(file.split('/').at(-1))) || (vendorStyle(file) && !json)) continue;
    const bytes = await readFile(join(root, file));
    let text;
    if (vendorStyle(file)) { try { text = new TextDecoder('utf-8', { fatal: true }).decode(bytes); } catch { problems.push(`${file}: Invalid JSON text encoding`); continue; } }
    else { const validation = validateText(file, bytes); problems.push(...validation.problems); text = validation.text; checked += 1; }
    if (text === null) continue;
    if (json) { try { JSON.parse(text); } catch (error) { problems.push(`${file}: Invalid JSON (${error.message})`); } }
    if (extension === '.yml' || extension === '.yaml') problems.push(...validateYAML(file, text));
    if (extension === '.md' && !vendorStyle(file)) { problems.push(...validateMarkdown(file, text)); problems.push(...await validateLinks(root, file, text)); }
  }
  problems.push(...await validateVersions(root));
  if (verifyBrand) { try { if (!await verifyInstalled(join(root, 'brand'))) problems.push('brand/: No integrated upstream kit'); } catch (error) { problems.push(`brand/: ${error.message}`); } }
  return { checked, problems };
}

async function main() {
  if (process.argv.length > 2) throw new Error('Usage: node scripts/check.mjs');
  const { checked, problems } = await checkRepository();
  if (problems.length) { console.error(problems.join('\n')); console.error(`Foundation check failed with ${problems.length} problem${problems.length === 1 ? '' : 's'}.`); process.exitCode = 1; }
  else console.log(`Foundation check passed: ${checked} project text files, Markdown links, version navigation, JSON/YAML, and upstream kit integrity.`);
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) main().catch((error) => { console.error(error.message); process.exitCode = 1; });
