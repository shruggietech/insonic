// SPDX-License-Identifier: Apache-2.0
import { useEffect, useState } from 'react';
import {
  Actions,
  Area,
  Button,
  Card,
  Check,
  Facts,
  Input,
  Select,
  Table,
} from './components';
import { Client, result, type Obj } from './client';
import { uuid } from './forms';
import { Pager, type Run } from './screens';
type Props = { client: Client; run: Run; appearance: (value: Obj) => void };
function ToolFields({
  label,
  tool,
  setTool,
}: {
  label: string;
  tool: Obj;
  setTool: (tool: Obj) => void;
}) {
  return (
    <fieldset>
      <legend>{label}</legend>
      <Input
        label={`${label} executable path`}
        value={tool.path ?? tool.executable ?? ''}
        onChange={(v) =>
          setTool({
            ...tool,
            ...(label === 'Cueson' ? { executable: v } : { path: v }),
          })
        }
      />
      <Input
        label={`${label} SHA-256`}
        value={tool.sha256 ?? tool.executable_sha256 ?? ''}
        onChange={(v) =>
          setTool({
            ...tool,
            ...(label === 'Cueson' ? { executable_sha256: v } : { sha256: v }),
          })
        }
      />
      {['ffprobe', 'ffmpeg', 'exiftool'].includes(label) && (
        <Input
          label={`${label} version`}
          value={tool.version ?? ''}
          onChange={(v) => setTool({ ...tool, version: v })}
        />
      )}
    </fieldset>
  );
}
export function Settings({ client, run, appearance }: Props) {
  const [settings, setSettings] = useState<Obj>(),
    [mediaTools, setMediaTools] = useState<Obj>({}),
    [processingTools, setProcessingTools] = useState<Obj>({}),
    [theme, setTheme] = useState('system'),
    [motion, setMotion] = useState(false);
  const [mediaRaw, setMediaRaw] = useState(''),
    [processingRaw, setProcessingRaw] = useState('');
  const [mediaOverride, setMediaOverride] = useState(false),
    [processingOverride, setProcessingOverride] = useState(false);
  const [credentialID, setCredentialID] = useState(''),
    [secret, setSecret] = useState(''),
    [passphrase, setPassphrase] = useState(''),
    [backend, setBackend] = useState(''),
    [credentialState, setCredentialState] = useState<Obj>();
  const [models, setModels] = useState<Obj>({ items: [] }),
    [model, setModel] = useState<Obj>(),
    [modelName, setModelName] = useState(''),
    [modelVersion, setModelVersion] = useState(''),
    [upstream, setUpstream] = useState(''),
    [license, setLicense] = useState(''),
    [capability, setCapability] = useState('transcription'),
    [fileRole, setFileRole] = useState('weights'),
    [fileURL, setFileURL] = useState(''),
    [fileHash, setFileHash] = useState(''),
    [fileSize, setFileSize] = useState('0'),
    [files, setFiles] = useState<Obj[]>([]);
  const load = async () => {
    const s = await client.call('settings.show');
    setSettings(s);
    setBackend(
      ['native', 'vault', 'session'].includes(s.credentials?.backend)
        ? s.credentials.backend
        : '',
    );
    const mediaExplicit = s.sections.media_tools.origin !== 'package';
    const processingExplicit = s.sections.processing_tools.origin !== 'package';
    const media = mediaExplicit ? (s.sections.media_tools.value ?? {}) : {};
    const processing = processingExplicit
      ? (s.sections.processing_tools.value ?? {})
      : {};
    const mediaEditor = {
      kind: 'media-tools',
      schema_version: '0.0.0',
      ...media,
    };
    const processingEditor = {
      kind: 'processing-tools',
      schema_version: '0.0.0',
      ...processing,
    };
    setMediaOverride(mediaExplicit);
    setProcessingOverride(processingExplicit);
    setMediaRaw(JSON.stringify(mediaEditor, null, 2));
    setProcessingRaw(JSON.stringify(processingEditor, null, 2));
    setMediaTools(mediaEditor);
    setProcessingTools(processingEditor);
    const a = s.sections.appearance.value ?? {
      theme: 'system',
      reduced_motion: false,
    };
    setTheme(a.theme);
    setMotion(a.reduced_motion);
    appearance(a);
    setModels(await client.call('models.list', '', { limit: 25 }));
  };
  useEffect(() => {
    run(load);
  }, [client.workspace]);
  const save = async (section: string, value: Obj | null) => {
    if (
      value !== null &&
      ((section === 'media_tools' && !mediaOverride) ||
        (section === 'processing_tools' && !processingOverride))
    )
      return;
    await client.call('settings.set', '', {
      section,
      revision: settings?.sections[section].revision,
      value,
    });
    await load();
  };
  const credentials = async (action: string) => {
    if (
      action === 'select' &&
      (!settings || !backend || backend === settings.credentials?.backend)
    )
      return;
    if (
      !credentialID &&
      ['add', 'replace', 'delete', 'status'].includes(action)
    )
      throw new Error('Enter the credential reference ID.');
    const input = JSON.stringify({
      ...(secret ? { value: secret } : {}),
      ...(passphrase ? { passphrase } : {}),
    });
    try {
      setCredentialState(
        result(
          await client.bridge.Credential(
            action === 'select'
              ? ['select', backend]
              : action === 'unlock'
                ? ['unlock']
                : [action, credentialID],
            input,
          ),
        ),
      );
      if (action === 'select') await load();
    } finally {
      setSecret('');
      setPassphrase('');
    }
  };
  return (
    <>
      <Card heading="Appearance">
        {settings?.sections.appearance.error && (
          <p role="alert">
            The saved appearance is invalid. Choose a theme and save to repair
            this section.
          </p>
        )}
        <Select
          label="Theme preference"
          value={theme}
          onChange={setTheme}
          options={['system', 'light', 'dark']}
        />
        <Check label="Reduce motion" value={motion} onChange={setMotion} />
        <Button
          onClick={() =>
            run(() => save('appearance', { theme, reduced_motion: motion }))
          }
        >
          Save appearance
        </Button>
      </Card>
      <Card heading="Workspace adapters">
        <p>
          These are the selected runtime backends. Moving an established
          workspace between backends requires migration through the CLI.
        </p>
        <Facts value={settings?.profiles} />
        <p>{settings?.profile_configuration}</p>
      </Card>
      <Card heading="Media tools">
        <p>
          Explicit workspace tools override installed package defaults. Hashes
          identify selected bytes.
        </p>
        <Facts value={{ origin: settings?.sections.media_tools.origin }} />
        <details>
          <summary>Effective media tools</summary>
          <Facts value={settings?.sections.media_tools.value} />
        </details>
        <Check
          label="Edit explicit workspace media tools"
          value={mediaOverride}
          onChange={setMediaOverride}
        />
        <p>
          Installed defaults follow the current package location. Enter a
          complete explicit configuration to override them for this workspace.
          Changes take effect when you save. Reset removes the workspace
          override.
        </p>
        <fieldset disabled={!mediaOverride}>
          <legend>Explicit media tools</legend>
          {['ffprobe', 'ffmpeg', 'exiftool'].map((key) => (
            <ToolFields
              key={key}
              label={key}
              tool={mediaTools[key] ?? {}}
              setTool={(value) =>
                setMediaTools((old) => ({ ...old, [key]: value }))
              }
            />
          ))}
          <Button
            disabled={!mediaOverride || !settings}
            onClick={() => run(() => save('media_tools', mediaTools))}
          >
            Save media tools
          </Button>
        </fieldset>
        <Button
          variant="secondary"
          disabled={settings?.sections.media_tools.origin !== 'workspace'}
          onClick={() => run(() => save('media_tools', null))}
        >
          Reset media tools to installed defaults
        </Button>
      </Card>
      <Card heading="Processing tools">
        <p>
          Configure local processing workers for this workspace. Weights are
          acquired separately. Choosing tools does not run inference.
        </p>
        <Facts value={{ origin: settings?.sections.processing_tools.origin }} />
        <details>
          <summary>Effective processing tools</summary>
          <Facts value={settings?.sections.processing_tools.value} />
        </details>
        <Check
          label="Edit explicit workspace processing tools"
          value={processingOverride}
          onChange={setProcessingOverride}
        />
        <p>
          Installed defaults follow the current package location. Enter a
          complete explicit configuration to override them for this workspace.
          Changes take effect when you save. Reset removes the workspace
          override.
        </p>
        <fieldset disabled={!processingOverride}>
          <legend>Explicit processing tools</legend>
          <ToolFields
            label="Cueson"
            tool={processingTools.cueson ?? {}}
            setTool={(value) =>
              setProcessingTools((old) => ({
                ...old,
                kind: 'processing-tools',
                schema_version: '0.0.0',
                cueson: value,
              }))
            }
          />
          {['ffmpeg', 'recognition_python', 'diarization_python', 'worker'].map(
            (key) => (
              <ToolFields
                key={key}
                label={key}
                tool={processingTools.processing?.[key] ?? {}}
                setTool={(value) =>
                  setProcessingTools((old) => ({
                    ...old,
                    kind: 'processing-tools',
                    schema_version: '0.0.0',
                    processing: { ...old.processing, [key]: value },
                  }))
                }
              />
            ),
          )}
          {[
            'threads',
            'timeout_ms',
            'max_input_bytes',
            'max_duration_us',
            'max_output_bytes',
          ].map((key) => (
            <Input
              key={key}
              label={`Processing ${key.replaceAll('_', ' ')}`}
              type="number"
              value={processingTools.processing?.[key] ?? 0}
              onChange={(value) =>
                setProcessingTools((old) => ({
                  ...old,
                  kind: 'processing-tools',
                  schema_version: '0.0.0',
                  processing: { ...old.processing, [key]: Number(value) },
                }))
              }
            />
          ))}
          <Button
            disabled={!processingOverride || !settings}
            onClick={() => run(() => save('processing_tools', processingTools))}
          >
            Save processing tools
          </Button>
        </fieldset>
        <Button
          variant="secondary"
          disabled={settings?.sections.processing_tools.origin !== 'workspace'}
          onClick={() => run(() => save('processing_tools', null))}
        >
          Reset processing tools to installed defaults
        </Button>
        <details>
          <summary>Advanced support-file configuration</summary>
          <p>
            Interpreter and support trees use versioned tool contracts.
            Executable, hash and resource settings are above.
          </p>
          <fieldset disabled={!mediaOverride}>
            <legend>Explicit media tool contract</legend>
            <Area
              label="Advanced media tool contract"
              value={mediaRaw}
              onChange={setMediaRaw}
            />
            <Button
              variant="secondary"
              disabled={!mediaOverride || !settings}
              onClick={() =>
                run(async () => {
                  await save('media_tools', JSON.parse(mediaRaw));
                })
              }
            >
              Validate and save advanced media tools
            </Button>
          </fieldset>
          <fieldset disabled={!processingOverride}>
            <legend>Explicit processing tool contract</legend>
            <Area
              label="Advanced processing tool contract"
              value={processingRaw}
              onChange={setProcessingRaw}
            />
            <Button
              variant="secondary"
              disabled={!processingOverride || !settings}
              onClick={() =>
                run(async () => {
                  await save('processing_tools', JSON.parse(processingRaw));
                })
              }
            >
              Validate and save advanced processing tools
            </Button>
          </fieldset>
        </details>
      </Card>
      <Card heading="Credentials">
        <p>
          Only newly entered values travel through protected native input. Saved
          values are never returned.
        </p>
        <Select
          label="Credential backend"
          value={backend}
          onChange={setBackend}
          options={[
            { value: '', label: 'Read saved selection first' },
            'native',
            'vault',
            'session',
          ]}
        />
        <Button
          variant="secondary"
          disabled={
            !settings || !backend || backend === settings.credentials?.backend
          }
          onClick={() => run(() => credentials('select'))}
        >
          Select credential backend
        </Button>
        <Input
          label="Credential reference ID"
          value={credentialID}
          onChange={setCredentialID}
        />
        <Button variant="secondary" onClick={() => setCredentialID(uuid())}>
          Create reference ID
        </Button>
        <Input
          label="New credential value"
          type="password"
          value={secret}
          onChange={setSecret}
        />
        <Input
          label="Vault passphrase"
          type="password"
          value={passphrase}
          onChange={setPassphrase}
        />
        <Actions>
          {['add', 'replace', 'status', 'delete', 'unlock'].map((action) => (
            <Button
              key={action}
              variant={action === 'delete' ? 'destructive' : 'secondary'}
              onClick={() => run(() => credentials(action))}
            >
              {action === 'unlock'
                ? 'Unlock vault'
                : `${action[0].toUpperCase()}${action.slice(1)} credential`}
            </Button>
          ))}
        </Actions>
        {credentialState && <Facts value={credentialState} />}
      </Card>
      <Card heading="Downloaded models">
        <Table
          caption="Managed model installations"
          heads={['Name', 'Version', 'State', 'Actions']}
        >
          {models.items.map((m: Obj) => (
            <tr key={m.id}>
              <td>{m.name}</td>
              <td>{m.model_version}</td>
              <td>{m.state}</td>
              <td>
                <Actions>
                  <Button
                    variant="ghost"
                    onClick={() =>
                      run(async () => {
                        client.invalidate();
                        setModel(undefined);
                        setModel(await client.call('models.show', m.id));
                      })
                    }
                  >
                    Inspect model
                  </Button>
                  <Button
                    variant="secondary"
                    onClick={() =>
                      run(async () => {
                        setModel(await client.call('models.verify', m.id));
                      })
                    }
                  >
                    Verify model bytes
                  </Button>
                  <Button
                    variant="secondary"
                    onClick={() =>
                      run(async () => {
                        setModel(await client.call('models.materialize', m.id));
                      })
                    }
                  >
                    Materialize model
                  </Button>
                </Actions>
              </td>
            </tr>
          ))}
        </Table>
        <Pager
          next={models.next_id}
          load={() =>
            run(async () => {
              setModels(
                await client.call('models.list', '', {
                  after_id: models.next_id,
                  limit: 25,
                }),
              );
            })
          }
        />
        {model && <Facts value={model} />}
      </Card>
      <Card heading="Acquire a pinned model">
        <Input label="Model name" value={modelName} onChange={setModelName} />
        <Input
          label="Model version"
          value={modelVersion}
          onChange={setModelVersion}
        />
        <Input
          label="Upstream revision"
          value={upstream}
          onChange={setUpstream}
        />
        <Input label="Model license" value={license} onChange={setLicense} />
        <Select
          label="Model capability"
          value={capability}
          onChange={setCapability}
          options={['transcription', 'diarization']}
        />
        <Input
          label="Model file role"
          value={fileRole}
          onChange={setFileRole}
        />
        <Input
          label="Model file source URL"
          value={fileURL}
          onChange={setFileURL}
        />
        <Input
          label="Model file SHA-256"
          value={fileHash}
          onChange={setFileHash}
        />
        <Input
          label="Model file byte size"
          type="number"
          value={fileSize}
          onChange={setFileSize}
        />
        <Button
          variant="secondary"
          onClick={() =>
            setFiles([
              ...files,
              {
                role: fileRole,
                url: fileURL,
                sha256: fileHash,
                size: Number(fileSize),
              },
            ])
          }
        >
          Add pinned file
        </Button>
        <Facts value={files} />
        <Button variant="secondary" onClick={() => setFiles([])}>
          Clear pinned files
        </Button>
        <Actions>
          {['acquire', 'register'].map((action) => (
            <Button
              key={action}
              onClick={() =>
                run(async () => {
                  const manifest = {
                    kind: 'base-model-manifest',
                    schema_version: '0.0.0',
                    name: modelName,
                    model_version: modelVersion,
                    upstream_revision: upstream,
                    license,
                    capabilities: [capability],
                    files,
                  };
                  await client.call(`models.${action}`, '', { manifest });
                  await load();
                })
              }
            >
              {action === 'acquire'
                ? 'Download pinned model'
                : 'Register model manifest'}
            </Button>
          ))}
        </Actions>
        <p>
          Model weights download when you choose Download pinned model.
          Registering a manifest records its identity without downloading files.
        </p>
      </Card>
    </>
  );
}
