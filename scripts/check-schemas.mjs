// SPDX-License-Identifier: Apache-2.0
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import Ajv2020 from 'ajv/dist/2020.js';
import addFormats from 'ajv-formats';
import { loadSchemaCatalog, root } from './schema-catalog.mjs';

export function validateCatalog(catalog) {
  const ajv = new Ajv2020({ allErrors: true, strict: true, strictRequired: false });
  addFormats(ajv);
  // Keep every Insonic schema under strict linting. The immutable upstream
  // schema uses conditional properties without repeated type declarations.
  // Its isolated compiler relaxes only that authoring lint, retaining all
  // schema constraints and format validation. No network loader is enabled.
  const upstreamSchema = JSON.parse(readFileSync(resolve(root, 'internal/subtitles/schema/cueson.schema.json'), 'utf8'));
  const upstreamAjv = new Ajv2020({ allErrors: true, strict: true, strictTypes: false, strictRequired: false });
  addFormats(upstreamAjv);
  const upstreamValidate = upstreamAjv.compile(upstreamSchema);
  const validateUpstream = (_schema, data) => {
    const valid = upstreamValidate(data);
    validateUpstream.errors = upstreamValidate.errors;
    return valid;
  };
  ajv.addKeyword({ keyword: 'packagedCueson', schemaType: 'boolean', errors: true, validate: validateUpstream });
  // This compiler resource delegates the external reference to the exact
  // packaged upstream validator. Published JSON retains its authoritative URI.
  ajv.addSchema({ $id: upstreamSchema.$id, packagedCueson: true });
  for (const item of catalog.schemas) ajv.addSchema(item.schema);
  const master = ajv.getSchema(catalog.master.schema.$id);
  let examples = 0;
  function documentation(node, location) {
    if (!node || typeof node !== 'object') return;
    for (const [name, value] of Object.entries(node.properties ?? {})) {
      if (!value.description) throw new Error(`Undescribed property: ${location}.${name}`);
    }
    for (const [key, value] of Object.entries(node)) {
      if (['examples', 'default', 'const', 'enum'].includes(key)) continue;
      if (Array.isArray(value)) value.forEach((item, index) => documentation(item, `${location}.${key}[${index}]`));
      else if (value && typeof value === 'object') documentation(value, `${location}.${key}`);
    }
  }
  for (const item of catalog.schemas) {
    if (!ajv.validateSchema(item.schema)) throw new Error(`Invalid schema ${item.filename}: ${ajv.errorsText()}`);
    const validate = ajv.getSchema(item.schema.$id);
    documentation(item.schema, item.filename);
    if (catalog.contracts.includes(item) && !item.schema.examples?.length) throw new Error(`Missing contract example: ${item.filename}`);
    for (const example of item.schema.examples ?? []) {
      if (!validate(example)) throw new Error(`Invalid ${item.filename} example: ${ajv.errorsText(validate.errors)}`);
      if (catalog.contracts.includes(item) && !master(example)) throw new Error(`Example is outside master: ${item.filename}: ${ajv.errorsText(master.errors)}`);
      examples += 1;
    }
  }
  return { ajv, master, examples };
}

export function checkSchemas() {
  const catalog = loadSchemaCatalog();
  const manifest = JSON.parse(readFileSync(resolve(root, 'docs/versions.json'), 'utf8'));
  const packageVersion = JSON.parse(readFileSync(resolve(root, 'package.json'), 'utf8')).version;
  const siteVersion = JSON.parse(readFileSync(resolve(root, 'site/package.json'), 'utf8')).version;
  if (manifest.latest !== catalog.version || packageVersion !== catalog.version || siteVersion !== catalog.version) throw new Error('Software, documentation and master schema versions must match.');
  const result = validateCatalog(catalog);
  console.log(`Schema check passed: ${catalog.schemas.length} schemas, ${catalog.contracts.length} contracts, ${result.examples} validated examples; version ${catalog.version}.`);
  return { catalog, ...result };
}
if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) { try { checkSchemas(); } catch (error) { console.error(error.message); process.exitCode = 1; } }
