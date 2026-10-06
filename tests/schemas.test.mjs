// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict';
import { cp, mkdtemp, mkdir, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';
import { validateCatalog } from '../scripts/check-schemas.mjs';
import { loadSchemaCatalog, renderSchemaReference } from '../scripts/schema-catalog.mjs';

const catalog = loadSchemaCatalog();
const { master, examples } = validateCatalog(catalog);
const example = kind => structuredClone(catalog.contracts.find(item => item.schema.properties.kind.const === kind).schema.examples[0]);

test('all documented contracts and local references validate through the release master', () => {
  assert.equal(catalog.contracts.length, 13);
  assert.equal(examples, 14);
  for (const item of catalog.contracts) for (const value of item.schema.examples) assert.equal(master(value), true);
});

test('release master rejects unknown kinds, version drift and undeclared core fields', () => {
  const value = example('import-manifest');
  value.kind = 'unregistered'; assert.equal(master(value), false);
  value.kind = 'import-manifest'; value.schema_version = '0.0.1'; assert.equal(master(value), false);
  value.schema_version = '0.0.0'; value.unrecognised_core_field = true; assert.equal(master(value), false);
});

test('an import cannot silently supply both a timestamp and a date at either scope', () => {
  const value = example('import-manifest');
  value.defaults.originated_at = '2026-10-04T14:30:00';
  value.defaults.originated_on = '2026-10-04'; assert.equal(master(value), false);
  delete value.defaults.originated_on; assert.equal(master(value), true);
  value.items[0].originated_on = '2026-10-04'; assert.equal(master(value), false);
});

test('speaker model identity and content digests reject malformed lineage identifiers', () => {
  const value = example('speaker-model');
  value.originating_speaker_id = 'speaker-name'; assert.equal(master(value), false);
  value.originating_speaker_id = example('speaker-model').originating_speaker_id;
  value.manifest_sha256 = 'short-digest'; assert.equal(master(value), false);
});

test('a completed model descriptor requires downloadable output or a hosted identity', () => {
  const value = example('speaker-model');
  value.outputs = { artifacts: [] }; assert.equal(master(value), false);
  const hosted = catalog.contracts.find(item => item.schema.properties.kind.const === 'speaker-model').schema.examples[1];
  assert.equal(master(hosted), true);
});

test('namespaced extensions survive serialization without weakening core validation', () => {
  const value = example('import-manifest');
  value.extensions = { 'example.operator': { batch_note: 'A custom annotation', revision: 2 } };
  const copy = JSON.parse(JSON.stringify(value));
  assert.equal(master(copy), true);
  assert.deepEqual(copy.extensions, value.extensions);
});

test('workspace backend selections accept their own configuration and reject another backend configuration', () => {
  const alternatives = [
    ['storage', 's3', { endpoint: 'http://127.0.0.1:9000', bucket: 'library', region: 'local', addressing_style: 'path', authentication: 'anonymous' }],
    ['catalog', 'postgresql', { host: 'db.example.com', port: 5432, database: 'library', schema: 'insonic', tls_mode: 'verify-full' }],
    ['graph', 'arcadedb', { endpoint: 'https://graph.example.com', database: 'library', preferred_dialect: 'arcade-opencypher' }],
    ['graph', 'community', { plugin_id: 'example.graph', contract_version: '1', options: {} }]
  ];
  for (const [role, adapter, configuration] of alternatives) {
    const value = example('workspace-config');
    const originalAdapter = value.profiles[role].adapter_id;
    value.profiles[role].adapter_id = adapter;
    value.profiles[role].configuration = configuration;
    assert.equal(master(value), true, `${adapter} must accept its valid configuration`);
    value.profiles[role].adapter_id = originalAdapter;
    assert.equal(master(value), false, `${originalAdapter} must reject ${adapter} configuration`);
  }
});

test('backend validation retains credential and service constraints', () => {
  const value = example('workspace-config');
  value.profiles.storage.adapter_id = 's3';
  value.profiles.storage.configuration = { endpoint: 'https://objects.example.com', bucket: 'library', region: 'local', addressing_style: 'path', authentication: 'credential' };
  assert.equal(master(value), false, 'credential authentication needs a credential reference');
  value.profiles.storage.configuration.credential_id = '11111111-1111-4111-8111-111111111111';
  assert.equal(master(value), true);
  value.profiles.catalog.adapter_id = 'postgresql';
  value.profiles.catalog.configuration = { host: 'db.example.com', port: 65536, database: 'library', schema: 'insonic', tls_mode: 'verify-full' };
  assert.equal(master(value), false, 'a PostgreSQL service port must be valid');
  value.profiles.catalog.configuration.port = 5432;
  assert.equal(master(value), true);
  value.profiles.graph.adapter_id = 'arcadedb';
  value.profiles.graph.configuration = { endpoint: 'https://graph.example.com', database: 'library', preferred_dialect: 'unknown' };
  assert.equal(master(value), false, 'ArcadeDB requires a declared supported dialect');
});

test('audit timestamps use the exact Cueson iso and unix_ns object contract', () => {
  const value = example('job-event');
  value.occurred_at = { iso: '1970-01-01T00:00:00Z', unix_ns: 0 };
  assert.equal(master(value), true);
  for (const timestamp of [
    '1970-01-01T00:00:00Z',
    { iso: '1970-01-01T00:00:00Z' },
    { unix_ns: 0 },
    { iso: 'invalid', unix_ns: 0 },
    { iso: '1970-01-01T00:00:00Z', unix_ns: 0.5 },
    { iso: '1970-01-01T00:00:00Z', unix_ns: 0, timezone: 'UTC' }
  ]) {
    value.occurred_at = timestamp;
    assert.equal(master(value), false, 'legacy, incomplete or malformed audit timestamps must fail');
  }
});

test('an unregistered schema cannot enter a release unnoticed', async () => {
  const fixture = await mkdtemp(join(tmpdir(), 'insonic-schema-test-'));
  try {
    await mkdir(join(fixture, 'schemas'));
    await cp(catalog.directory, join(fixture, 'schemas', 'v0.0.0'), { recursive: true });
    const orphan = structuredClone(catalog.contracts[0].schema);
    orphan.$id = 'https://raw.githubusercontent.com/shruggietech/insonic/v0.0.0/schemas/v0.0.0/orphan.schema.json';
    await writeFile(join(fixture, 'schemas', 'v0.0.0', 'orphan.schema.json'), JSON.stringify(orphan));
    assert.throws(() => loadSchemaCatalog('0.0.0', fixture), /absent from the master registry/);
  } finally { await rm(fixture, { recursive: true, force: true }); }
});

test('published reference derives field descriptions and complete examples from schemas', () => {
  const reference = renderSchemaReference(catalog);
  const contract = catalog.contracts.find(item => item.schema.properties.kind.const === 'import-manifest');
  assert.ok(reference.includes(contract.schema.properties.items.description));
  assert.ok(reference.includes(JSON.stringify(contract.schema.examples[0], null, 2)));
  assert.ok(reference.includes('/schemas/v0.0.0/master.schema.json'));
});

test('published reference explains alternative backend and query fields, typed parameters and shared timestamps', () => {
  const reference = renderSchemaReference(catalog);
  const workspace = catalog.contracts.find(item => item.schema.properties.kind.const === 'workspace-config').schema;
  const query = catalog.contracts.find(item => item.schema.properties.kind.const === 'graph-query').schema;
  const timestamp = catalog.schemas.find(item => item.filename === 'common.schema.json').schema.$defs.utcInstant;
  const fields = [
    workspace.$defs.s3Configuration.properties.addressing_style,
    workspace.$defs.postgresqlConfiguration.properties.host,
    query.properties.definition.oneOf[0].properties.filters.properties.speaker_ids,
    query.properties.definition.oneOf[1].properties.text,
    query.properties.parameters.additionalProperties.properties.value,
    timestamp.properties.iso,
    timestamp.properties.unix_ns
  ];
  for (const field of fields) assert.ok(reference.includes(field.description), `Missing reference description: ${field.description}`);
  assert.ok(reference.includes('db.example.com'), 'field examples must appear even when absent from complete document examples');
  assert.ok(reference.includes('Required fields: `credential_id`'), 'conditional credential requirements must be explained');
  assert.equal(reference.split(workspace.$defs.s3Configuration.properties.addressing_style.description).length - 1, 1, 'shared backend definitions should be documented once');
});
