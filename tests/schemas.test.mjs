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
  assert.equal(catalog.contracts.length, 21);
  assert.equal(examples, 28);
  for (const item of catalog.contracts) for (const value of item.schema.examples) assert.equal(master(value), true);
});

test('backup source and portable authority proofs and configured signing have explicit contracts', () => {
  const backup = example('workspace-backup');
  assert.equal(master(backup), true);
  delete backup.source_catalog_digest;
  assert.equal(master(backup), false, 'source capture proof cannot be omitted');
  const release = example('release-candidate');
  release.packages[0].signing.configured = true;
  assert.equal(master(release), false, 'configured signing cannot claim unverified success');
  release.packages[0].signing.verified = true;
  release.packages[0].signing.status = 'signed';
  assert.equal(master(release), true, JSON.stringify(master.errors));
  release.packages[0].signing.configured = false;
  assert.equal(master(release), false, 'unconfigured signing cannot claim publisher verification');
  release.packages[0].signing.verified = false;
  release.packages[0].signing.status = 'unconfigured';
  assert.equal(master(release), true, JSON.stringify(master.errors));
  release.packages.pop();
  assert.equal(master(release), false, 'complete native matrix requires six variants');
});

test('speaker output snapshots preserve invalidated hosted lineage without inventing retained publications',()=>{
 const model=example('speaker-model');
 const snapshot=example('catalog-snapshot');snapshot.catalog_schema=9;
 const output={id:model.model_version_id,model_id:model.model_family_id,speaker_id:model.originating_speaker_id,dataset_id:model.training.dataset.dataset_snapshot_id,work_id:model.training.job_id,kind:model.compatibility.model_kind,name:model.display_name,metadata:model,publication_ids:[]};
 snapshot.records.speaker_outputs=[output];
 assert.equal(master(snapshot),false,'current output needs its accepted manifest');
 output.publication_ids=[model.model_version_id];assert.equal(master(snapshot),true);
 model.state='invalidated';model.manifest_sha256=null;model.training.preparation=null;model.training.dataset.state='invalidated';model.training.dataset.manifest_sha256=null;
 output.publication_ids=[];
 assert.equal(master(snapshot),true,JSON.stringify(master.errors));
 model.training.dataset.manifest_sha256='a'.repeat(64);assert.equal(master(snapshot),false,'obsolete dataset digest is not current authority');
});

test('speaker preparation supports verified transient clips without fabricated durable audio',()=>{
 const model=example('speaker-model');
 model.training.preparation.input_artifact_ids=[];
 assert.equal(master(model),true,JSON.stringify(master.errors));
 model.training.preparation.artifact.byte_length=0;
 assert.equal(master(model),false,'noncontent preparation receipt still needs verified bytes');
});

test('roster add/remove require selectors while replace/clear allow empty declarations',()=>{
 const base={...example('runtime-request'),item_id:'44444444-4444-4444-8444-444444444444',data:{expected_revision:0,speakers:[]}};
 for(const mode of ['add','remove']) {
  assert.equal(master({...base,operation:'recordings.roster.'+mode}),false);
  assert.equal(master({...base,operation:'recordings.roster.'+mode,data:{...base.data,speakers:['Known']}}),true);
 }
 for(const mode of ['replace','clear'])assert.equal(master({...base,operation:'recordings.roster.'+mode}),true);
});

test('model references share bounded syntax across processing, pipelines and import', () => {
  const request = {...example('runtime-request'), operation:'recordings.process', item_id:'22222222-2222-4222-8222-222222222222'};
  for (const reference of ['speech', 'speech-v2', 'source:catalog/speech', 'base:33333333-3333-4333-8333-333333333333', 'speaker:33333333-3333-4333-8333-333333333333']) {
    assert.equal(master({...request,data:{transcription:'generate',diarization:'reuse',recognition_model_id:reference}}),true,reference);
    const pipeline=example('pipeline-config');
    pipeline.configuration.recognition.model_id=reference;
    assert.equal(master(pipeline),true,reference);
    const input=example('import-manifest');
    input.defaults.attribution='diarize';input.defaults.diarization_model_id=reference;
    assert.equal(master(input),true,reference);
  }
  for (const reference of ['Speech',' speech','speech ','https://example.org/model','a'.repeat(65),'base:invalid','source//speech']) {
    assert.equal(master({...request,data:{transcription:'generate',diarization:'reuse',recognition_model_id:reference}}),false,reference);
  }
});

test('revisioned model references reject untyped targets, misplaced IDs and credentials',()=>{
  const request={...example('runtime-request'),item_id:'22222222-2222-4222-8222-222222222222',operation:'models.alias.set',data:{expected_revision:0,alias:{id:'22222222-2222-4222-8222-222222222222',name:'speech',state:'active',target:{kind:'base',id:'33333333-3333-4333-8333-333333333333',operation:'transcription'}}}};
  assert.equal(master(request),true);
  const invalid=structuredClone(request);invalid.data.alias.target.kind='invented';assert.equal(master(invalid),false);
  invalid.data.alias.target.kind='base';invalid.data.alias.target.credential_id='44444444-4444-4444-8444-444444444444';assert.equal(master(invalid),false);
  assert.equal(master({...request,data:{...request.data,expected_revision:-1}}),false);
  assert.equal(master({...request,operation:'models.alias.remove',data:{expected_revision:0}}),false);
  assert.equal(master({...request,operation:'models.resolve',data:{reference:'speech',operation:'transcription'}}),false,'resolve carries reference in data, never item ID');
  const resolve={...example('runtime-request'),operation:'models.resolve',data:{reference:'source:archive/release/1',operation:'transcription'}};
  assert.equal(master(resolve),true);
  assert.equal(master({...resolve,data:{...resolve.data,operation:'voice-matching'}}),true);
  assert.equal(master({...resolve,data:{...resolve.data,token:'secret'}}),false);
});

test('model catalog and compatibility contracts retain complete pinned bundle metadata',()=>{
  const manifest=example('base-model-manifest');
  manifest.compatibility={adapter:'faster-whisper',contract_version:'1',architecture:'whisper',runtime_format:'ctranslate2',sample_rates:[16000],channels:[1]};
  assert.equal(master(manifest),true);
  manifest.compatibility.channels=[0];assert.equal(master(manifest),false);
  manifest.compatibility.channels=[1,1];assert.equal(master(manifest),false);
  manifest.compatibility.channels=[1];
  const source={kind:'model-catalog',schema_version:'0.0.0',entries:[{selector:'release-1',manifest}]};
  assert.equal(master(source),true);
  assert.equal(master({...source,entries:Array.from({length:129},()=>source.entries[0])}),false);
  const snapshot=example('catalog-snapshot');snapshot.catalog_schema=8;snapshot.records.model_aliases=[{id:'22222222-2222-4222-8222-222222222222',name:'speech',revision:1,state:'deleted',target:{kind:'base',id:'33333333-3333-4333-8333-333333333333',operation:'transcription'}}];
  assert.equal(master(snapshot),true);
  snapshot.records.model_aliases[0].revision=0;assert.equal(master(snapshot),false);
});

test('work model summaries distinguish operation-neutral acquisition from missing consumer selections',()=>{
  const id='33333333-3333-4333-8333-333333333333';
  const resolution={reference:'base:'+id,target:{kind:'base',id,operation:''},manifest_digest:'a'.repeat(64),upstream_revision:'immutable-1',state:'registered',compatible:true,diagnostics:[]};
  const response=selection=>({kind:'runtime-response',schema_version:'0.0.0',result:{model_selections:[selection]}});
  assert.equal(master(response(resolution)),true,'explicit acquisition does not elect an inference operation');
  for(const change of [{target:{kind:'speaker',id,operation:''}},{target:{kind:'base',id,operation:'invented'}},{target:{kind:'base',id,operation:'',adapter:'faster-whisper'}},{state:'missing'},{manifest_digest:''}])
    assert.equal(master(response({...resolution,...change})),false,'neutral acquisition cannot weaken execution or immutable identity');
  const missing={reference:'missing-speech',target:{kind:'unresolved',operation:'transcription'},manifest_digest:'',upstream_revision:'',state:'missing',compatible:false,diagnostics:['Selected model reference is missing.']};
  assert.equal(master(response(missing)),true,'pipeline inspection can diagnose an unresolved alias without inventing an ID');
  for(const change of [{compatible:true},{state:'registered'},{manifest_digest:'a'.repeat(64)},{target:{...missing.target,id}},{target:{kind:'unresolved',operation:''}}])
    assert.equal(master(response({...missing,...change})),false,'unresolved targets exist only in nonexecutable missing diagnostics');
  const alias={...example('runtime-request'),item_id:id,operation:'models.alias.set',data:{expected_revision:0,alias:{id,name:'speech',state:'active',target:resolution.target}}};
  assert.equal(master(alias),false,'aliases still require an operation-specific exact target');
  alias.data.alias.target=missing.target;assert.equal(master(alias),false,'an unresolved diagnostic is never an alias target');
});

test('recording operation requests keep transient assembly separate from durable settings', () => {
  const base = {...example('runtime-request'), operation:'recordings.process', item_id:'22222222-2222-4222-8222-222222222222', data:{transcription:'supplied',diarization:'run',diarization_model_id:'33333333-3333-4333-8333-333333333333'}};
  assert.equal(master(base),true);
  assert.equal(master({...base,data:{...base.data,transcription:'generate'}}),false);
  assert.equal(master({...base,data:{...base.data,transcription:'generate',recognition_model_id:'44444444-4444-4444-8444-444444444444'}}),true);
  assert.equal(master({...base,data:{transcription:'reuse',diarization:'reuse'}}),false);
  assert.equal(master({...base,data:{...base.data,turns:[]}}),false);
  assert.equal(master({...base,data:{...base.data,recognition:{device:'hosted'}}}),false);
  assert.equal(master({...base,operation:'recordings.assemble',data:{subtitle_format:'srt',turns:[{label:'voice',start_us:0,end_us:1000000}]}}),true);
  assert.equal(master({...base,operation:'recordings.assemble',data:{turns:[],archive:true}}),false);
  assert.equal(master({...base,operation:'recordings.assemble',data:{turns:[{label:'voice',start_us:-1,end_us:1000}]}}),false);
});

test('recording read, mapping and export requests enforce typed argument bounds', () => {
  const base = {...example('runtime-request'),item_id:'22222222-2222-4222-8222-222222222222'};
  assert.equal(master({...base,operation:'recordings.show'}),true);
  assert.equal(master({...base,operation:'recordings.show',data:{}}),false);
  assert.equal(master({...base,operation:'recordings.document',data:{offset:0,limit:65536,document_digest:'a'.repeat(64)}}),true);
  assert.equal(master({...base,operation:'recordings.document',data:{offset:0,limit:65537}}),false);
  assert.equal(master({...base,operation:'recordings.mappings',data:{limit:100}}),true);
  const segment={segment_id:'33333333-3333-4333-8333-333333333333',segment_revision:1};
  assert.equal(master({...base,operation:'recordings.resolve-segment',data:segment}),true);
  assert.equal(master({...base,operation:'recordings.resolve-segment',data:{...segment,segment_revision:0}}),false);
  assert.equal(master({...base,operation:'recordings.resolve-segment',data:{...segment,segment_id:'cue-0'}}),false);
  const mapping={revision:0,local_speaker_id:'33333333-3333-4333-8333-333333333333',speaker_id:'44444444-4444-4444-8444-444444444444',document_digest:'a'.repeat(64)};
  assert.equal(master({...base,operation:'recordings.map-speaker',data:mapping}),true);
  assert.equal(master({...base,operation:'recordings.map-speaker',data:{...mapping,mapping_revision:0}}),true);
  assert.equal(master({...base,operation:'recordings.map-speaker',data:{...mapping,mapping_revision:9}}),true);
  assert.equal(master({...base,operation:'recordings.map-speaker',data:{...mapping,mapping_revision:-1}}),false);
  assert.equal(master({...base,operation:'recordings.map-speaker',data:{...mapping,local_speaker_id:'voice1'}}),true);
  assert.equal(master({...base,operation:'recordings.map-speaker',data:{...mapping,local_speaker_id:' voice1'}}),false);
  assert.equal(master({...base,operation:'recordings.map-speaker',data:{...mapping,speaker_id:'voice1'}}),false);
  assert.equal(master({...base,operation:'recordings.export',data:{format:'cueson',destination:'/tmp/export.json',strict:true}}),true);
  assert.equal(master({...base,operation:'recordings.export',data:{format:'invented',destination:'/tmp/export.json'}}),false);
  assert.equal(master({...base,operation:'recordings.process',data:{diarization_model_id:'33333333-3333-4333-8333-333333333333'},publication_id:'55555555-5555-4555-8555-555555555555'}),false);
});

test('saved pipeline and context requests preserve routing, revisions and bounds', () => {
  const pipeline=example('pipeline-config');
  assert.equal(master(pipeline),true);
  pipeline.preset='connected';assert.equal(master(pipeline),false);
  pipeline.preset='custom';assert.equal(master(pipeline),true);
  pipeline.configuration.recognition.credential_id='33333333-3333-4333-8333-333333333333';assert.equal(master(pipeline),false);
  const base={...example('runtime-request'),operation:'terms.compile',data:{speaker_ids:[],max_hint_bytes:8192}};
  assert.equal(master(base),true);
  assert.equal(master({...base,item_id:'33333333-3333-4333-8333-333333333333'}),false);
  assert.equal(master({...base,data:{...base.data,max_hint_bytes:8193}}),false);
  assert.equal(master({...base,data:{pipeline_revision:1}}),false);
  const process={...base,operation:'recordings.process',item_id:'33333333-3333-4333-8333-333333333333',data:{pipeline_id:'66666666-6666-4666-8666-666666666666',pipeline_revision:1,transcription:'generate',diarization:'run'}};
  assert.equal(master(process),true);
  assert.equal(master({...process,data:{...process.data,overrides:{unknown:{}}}}),false);
});

test('processing tool contract selects exact executables and bounded local settings', () => {
  const value=example('processing-tools');
  assert.equal(master(value),true);
  value.processing.timeout_ms=0;assert.equal(master(value),false);
  value.processing.timeout_ms=600000;value.processing.threads=65;assert.equal(master(value),false);
  value.processing.threads=2;value.processing.worker.sha256='bad';assert.equal(master(value),false);
  value.processing.worker.sha256='b'.repeat(64);value.processing.provider='hosted';assert.equal(master(value),false);
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

test('downloaded models require exact file identity and reject credential material', () => {
  const value = example('base-model-manifest');
  assert.equal(master(value), true);
  value.files[0].sha256 = 'short'; assert.equal(master(value), false);
  value.files[0].sha256 = 'a'.repeat(64);
  value.files[0].size = -1; assert.equal(master(value), false);
  value.files[0].size = 0; assert.equal(master(value), true);
  value.files[0].password = 'secret'; assert.equal(master(value), false);
  delete value.files[0].password;
  value.files[0].credential_id = 'human-name'; assert.equal(master(value), false);
});

test('extractor configuration qualifies delegated support files and rejects undeclared fields', () => {
  const value = example('media-tools');
  value.exiftool.support_files = [{path:'/opt/insonic/tools/lib/ExifTool.pm',sha256:'c'.repeat(64)}];
  assert.equal(master(value), true);
  value.exiftool.support_files[0].sha256 = 'bad'; assert.equal(master(value), false);
  value.exiftool.support_files[0].sha256 = 'c'.repeat(64);
  value.ffprobe.extra = true; assert.equal(master(value), false);
});

test('import options expose explicit acquisition and current recording-date policies', () => {
  const value = example('import-manifest');
  value.defaults.date_precedence = 'filesystem-fallback';
  value.items[0].title = 'A recording'; value.items[0].new_entry = true;
  value.items[0].credential_id = '11111111-1111-4111-8111-111111111111';
  value.items[0].acquisition_adapter = 'https';
  assert.equal(master(value), true);
  value.defaults.date_precedence = 'invent-time'; assert.equal(master(value), false);
});
test('approximate import dates preserve open or closed UTC bounds and reject exact-date conflicts', () => {
  const value = example('import-manifest');
  value.defaults.originated_earliest = { iso: '1970-01-01T00:00:00.123456789Z', unix_ns: 123456789 };
  assert.equal(master(value), true);
  value.defaults.originated_latest = { iso: '1970-01-02T00:00:00Z', unix_ns: 86400000000000 };
  assert.equal(master(value), true);
  value.defaults.originated_on = '1970-01-01'; assert.equal(master(value), false);
  delete value.defaults.originated_on;
  delete value.items[0].originated_at;
  value.items[0].originated_latest = structuredClone(value.defaults.originated_latest);
  assert.equal(master(value), true);
  value.items[0].originated_at = '1970-01-01T00:00:00'; assert.equal(master(value), false);
  delete value.items[0].originated_at;
  delete value.items[0].originated_latest.unix_ns; assert.equal(master(value), false);
});

test('remote acquisition budgets and explicit local transport are typed per batch and item', () => {
  const value = example('import-manifest');
  Object.assign(value.defaults, {local_http: true, acquisition_max_bytes: 1024, acquisition_timeout_ms: 5000});
  Object.assign(value.items[0], {local_http: false, acquisition_max_bytes: 2048, acquisition_timeout_ms: 10000});
  assert.equal(master(value), true);
  value.items[0].acquisition_max_bytes = 0; assert.equal(master(value), false);
  value.items[0].acquisition_max_bytes = 2048;
  value.defaults.acquisition_timeout_ms = -1; assert.equal(master(value), false);
  value.defaults.acquisition_timeout_ms = 5000;
  value.items[0].local_http = 'true'; assert.equal(master(value), false);
});

test('assistance requests keep elections outside generated query data',()=>{
 const base={...example('runtime-request'),operation:'query.assist',data:{prompt:'List media',mode:'suggest'}};assert.equal(master(base),true);
 assert.equal(master({...base,data:{...base.data,credential:'secret'}}),false);
 assert.equal(master({...base,data:{...base.data,configuration:{enabled:true,limits:{max_rows:501}}}}),false);
 assert.equal(master({...base,item_id:'22222222-2222-4222-8222-222222222222'}),false);
 assert.equal(master({...base,operation:'query.assistance-show',data:{}}),false);
});

test('media and standalone transcript manifest shapes agree with runtime', () => {
 const base=example('import-manifest');const record='11111111-1111-4111-8111-111111111111';
 for(const item of [{source:'audio.flac'},{source:'audio.flac',transcript:'cue.json'},{source:'audio.flac',subtitle:'cue.srt'},{record,transcript:'cue.json'},{record,subtitle:'cue.srt'},{record,source:'cue.json'},{record,transcript:'cue.json',expected_revision:1,expected_recording_revision:0}]) assert.equal(master({...base,items:[item]}),true,JSON.stringify(item));
 for(const item of [{record},{transcript:'cue.json'},{source:'audio.flac',record,transcript:'cue.json'},{source:'audio.flac',record,subtitle:'cue.srt'},{record,transcript:'cue.json',subtitle:'cue.srt'},{source:'audio.flac',transcript:'cue.json',subtitle:'cue.srt'},{source:'audio.flac',expected_revision:1}]) assert.equal(master({...base,items:[item]}),false,JSON.stringify(item));
});
