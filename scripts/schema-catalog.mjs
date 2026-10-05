// SPDX-License-Identifier: Apache-2.0
import { readFileSync, readdirSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

export const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
export function loadSchemaCatalog(version = readFileSync(resolve(root, 'VERSION'), 'utf8').trim(), repositoryRoot = root) {
  if (!/^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/.test(version)) throw new Error('Invalid schema release version.');
  const directory = resolve(repositoryRoot, 'schemas', `v${version}`);
  const schemas = readdirSync(directory).filter(name => name.endsWith('.schema.json')).sort().map(filename => ({ filename, schema: JSON.parse(readFileSync(resolve(directory, filename), 'utf8')) }));
  const master = schemas.find(item => item.filename === 'master.schema.json');
  if (!master) throw new Error(`Missing master schema for ${version}.`);
  const definitions = master.schema.$defs ?? {};
  const references = new Set(Object.values(definitions).map(value => value.$ref));
  for (const item of schemas) {
    const expectedId = `https://raw.githubusercontent.com/shruggietech/insonic/v${version}/schemas/v${version}/${item.filename}`;
    if (item.schema.$id !== expectedId) throw new Error(`Schema identity/version mismatch: ${item.filename}`);
    if (item.schema.$schema !== 'https://json-schema.org/draft/2020-12/schema') throw new Error(`Unsupported schema dialect: ${item.filename}`);
    if (!item.schema.title || !item.schema.description) throw new Error(`Missing schema documentation: ${item.filename}`);
    if (item !== master && !references.has(item.filename)) throw new Error(`Schema is absent from the master registry: ${item.filename}`);
    const kind = item.schema.properties?.kind?.const;
    if (kind && (item.schema.properties?.schema_version?.const !== version || !item.schema.required?.includes('kind') || !item.schema.required?.includes('schema_version'))) throw new Error(`Missing release envelope: ${item.filename}`);
  }
  for (const ref of references) if (!schemas.some(item => item.filename === ref)) throw new Error(`Missing registered schema: ${ref}`);
  const contracts = schemas.filter(item => item.schema.properties?.kind?.const);
  const branches = new Set((master.schema.oneOf ?? []).map(item => item.$ref));
  for (const item of contracts) if (!branches.has(`#/$defs/${item.schema.properties.kind.const}`)) throw new Error(`Missing master document branch: ${item.filename}`);
  return { version, directory, schemas, contracts, master };
}

const clean = value => String(value ?? '').replaceAll('|', '\\|').replace(/\r?\n/g, ' ');
function shape(node) {
  if (node.$ref) return `Reference: ${node.$ref}`;
  if (node.const !== undefined) return `Constant ${JSON.stringify(node.const)}`;
  if (node.enum) return node.enum.map(value => JSON.stringify(value)).join(', ');
  if (node.type) return Array.isArray(node.type) ? node.type.join(' or ') : node.type;
  return node.oneOf ? 'Exactly one alternative' : node.anyOf ? 'At least one alternative' : node.allOf ? 'All listed constraints' : 'Structured value';
}
const exampleValues = node => (node.examples ?? []).map(value => clean(JSON.stringify(value))).join('; ');
function condition(node) {
  const values = Object.entries(node.properties ?? {}).map(([name, value]) => `\`${clean(name)}\` ${value.const !== undefined ? `is ${clean(JSON.stringify(value.const))}` : `matches ${clean(shape(value))}`}`);
  const required = (node.required ?? []).filter(name => !Object.hasOwn(node.properties ?? {}, name));
  if (required.length) values.push(`has ${required.map(name => `\`${clean(name)}\``).join(', ')}`);
  return values.join(' and ') || 'the condition matches';
}
function referenceTables(lines, schema) {
  const walk = (node, path = '', root = false) => {
    if (!node || typeof node !== 'object') return;
    if (path) lines.push(`### ${path}`, '');
    if (node.description && !root) lines.push(node.description, '');
    if (node.properties) {
      lines.push('| Field | Required | Value | Description | Examples |', '| --- | --- | --- | --- | --- |');
      for (const [name, property] of Object.entries(node.properties)) lines.push(`| \`${clean(name)}\` | ${node.required?.includes(name) ? 'Yes' : 'No'} | ${clean(shape(property))} | ${clean(property.description)} | ${exampleValues(property)} |`);
      lines.push('');
    } else {
      if (node.$ref || node.type || node.const !== undefined || node.enum) lines.push(`Value: ${clean(shape(node))}.`, '');
      if (node.required?.length) lines.push(`Required fields: ${node.required.map(name => `\`${clean(name)}\``).join(', ')}.`, '');
    }
    if (!root && node.examples?.length) for (const example of node.examples) lines.push('Example:', '', '```json', JSON.stringify(example, null, 2), '```', '');
    if (node.not) lines.push(`Disallowed combination: ${condition(node.not)}.`, '');
    for (const [name, property] of Object.entries(node.properties ?? {})) {
      if (property.properties || property.items || property.oneOf || property.anyOf || property.allOf || typeof property.additionalProperties === 'object') walk(property, path ? `${path}.${name}` : name);
    }
    if (node.items) walk(node.items, `${path}[]`);
    if (typeof node.additionalProperties === 'object') walk(node.additionalProperties, `${path}{name}`);
    for (const keyword of ['oneOf', 'anyOf', 'allOf']) {
      const alternatives = node[keyword] ?? [];
      if (alternatives.length) lines.push(`${keyword === 'oneOf' ? 'Exactly one' : keyword === 'anyOf' ? 'At least one' : 'All'} of the following ${keyword === 'allOf' ? 'constraints apply' : 'alternatives must match'}:`, '');
      for (const [index, alternative] of alternatives.entries()) {
        const discriminator = Object.values(alternative.properties ?? {}).find(property => typeof property.const === 'string')?.const;
        const label = alternative.title ?? discriminator ?? `Alternative ${index + 1}`;
        if (alternative.if) {
          lines.push(`When ${condition(alternative.if)}:`, '');
          if (alternative.then) walk(alternative.then);
          if (alternative.else) { lines.push('Otherwise:', ''); walk(alternative.else); }
        } else walk(alternative, path ? `${path}: ${label}` : label);
      }
    }
    for (const [name, definition] of Object.entries(node.$defs ?? {})) walk(definition, definition.title ?? `Definition: ${name}`);
  };
  walk(schema, '', true);
}
export function renderSchemaReference(catalog) {
  const lines = ['## Contract reference', '', catalog.master.schema.description, '', `[Download the master schema](/schemas/v${catalog.version}/master.schema.json) · [Download shared definitions](/schemas/v${catalog.version}/common.schema.json)`, '', 'The following definitions, descriptions and examples are generated from the versioned JSON Schema files during the documentation build.', ''];
  for (const item of catalog.contracts) {
    const schema = item.schema;
    lines.push(`## ${schema.title}`, '', schema.description, '', `[Download ${item.filename}](/schemas/v${catalog.version}/${item.filename})`, '');
    referenceTables(lines, schema);
    for (const example of schema.examples ?? []) lines.push('### Example', '', '```json', JSON.stringify(example, null, 2), '```', '');
  }
  const shared = catalog.schemas.find(item => item.filename === 'common.schema.json')?.schema;
  lines.push('## Shared definitions', '', shared?.description ?? '', '');
  referenceTables(lines, shared);
  return `${lines.join('\n')}\n`;
}
