// SPDX-License-Identifier: Apache-2.0
import { readFileSync } from 'node:fs';
import Ajv2020 from 'ajv/dist/2020.js';
import addFormats from 'ajv-formats';
const version = readFileSync(new URL('../../../VERSION', import.meta.url), 'utf8').trim();
const ajv = new Ajv2020({
  allErrors: true,
  strict: true,
  strictRequired: false,
});
addFormats(ajv);
let requestID;
for (const name of [
  'common',
  'import-manifest',
  'base-model-manifest',
  'pipeline-config',
  'media-tools',
  'processing-tools',
  'query-assistance',
  'runtime-request',
]) {
  const schema = JSON.parse(
    readFileSync(
      new URL(`../../../schemas/v${version}/${name}.schema.json`, import.meta.url),
      'utf8',
    ),
  );
  ajv.addSchema(schema);
  if (name === 'runtime-request') requestID = schema.$id;
}
const validate = ajv.getSchema(requestID);
export function validateNativeRequest(request) {
  const envelope = {
    kind: 'runtime-request',
    schema_version: '1.0.0',
    request_id: 'dddddddd-dddd-4ddd-8ddd-dddddddddddd',
    ...request,
  };
  const valid = validate(envelope);
  return { valid, errors: ajv.errorsText(validate.errors) };
}
