// SPDX-License-Identifier: Apache-2.0
import { readFileSync } from 'node:fs';
import Ajv2020 from 'ajv/dist/2020.js';
import addFormats from 'ajv-formats';
import { loadSchemaCatalog } from '../../../scripts/schema-catalog.mjs';
const version = readFileSync(new URL('../../../VERSION', import.meta.url), 'utf8').trim();
const ajv = new Ajv2020({
  allErrors: true,
  strict: true,
  strictRequired: false,
});
addFormats(ajv);
const catalog = loadSchemaCatalog(version);
for (const { schema } of [...catalog.schemas, ...catalog.dependencies]) {
  ajv.addSchema(schema);
}
const requestID = catalog.schemas.find(
  ({ filename }) => filename === 'runtime-request.schema.json',
).schema.$id;
const validate = ajv.getSchema(requestID);
export function validateNativeRequest(request) {
  const envelope = {
    kind: 'runtime-request',
    schema_version: version,
    request_id: 'dddddddd-dddd-4ddd-8ddd-dddddddddddd',
    ...request,
  };
  const valid = validate(envelope);
  return { valid, errors: ajv.errorsText(validate.errors) };
}
