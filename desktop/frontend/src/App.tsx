// SPDX-License-Identifier: Apache-2.0
import { useEffect, useMemo, useRef, useState } from 'react';
import { AppFrame } from '../../../brand/kit/web/react/server';
import { AppFrameEnvironmentBridge } from '../../../brand/kit/web/react/environment';
import { Actions, Button, Card, Check, EmptyState, Input } from './components';
import {
  Client,
  result,
  StaleResponse,
  type NativeBridge,
  type Obj,
} from './client';
import { Jobs, Library, Pipelines, Speakers, Terms, type Run } from './screens';
import { Settings } from './settings';
export function App({ bridge }: { bridge: NativeBridge }) {
  const client = useMemo(() => new Client(bridge), [bridge]);
  const [workspace, setWorkspace] = useState<Obj>(),
    [path, setPath] = useState(''),
    [name, setName] = useState('My voice archive'),
    [create, setCreate] = useState(false),
    [screen, setScreen] = useState('Library'),
    [error, setError] = useState(''),
    [notice, setNotice] = useState(''),
    [busy, setBusy] = useState(0);
  const generation = useRef(0);
  const run: Run = async (action) => {
    const token = generation.current;
    setError('');
    setNotice('');
    setBusy((n) => n + 1);
    try {
      await action();
      if (token === generation.current)
        setNotice('Current workspace state refreshed.');
    } catch (e) {
      if (!(e instanceof StaleResponse) && token === generation.current)
        setError(
          e instanceof Error
            ? e.message
            : 'The operation failed. Refresh current state and retry.',
        );
    } finally {
      setBusy((n) => Math.max(0, n - 1));
    }
  };
  const applyAppearance = (saved: Obj | null) => {
    const value = saved ?? { theme: 'system', reduced_motion: false };
    const light =
      value.theme === 'light' ||
      (value.theme === 'system' &&
        (window.matchMedia?.('(prefers-color-scheme: light)').matches ??
          false));
    document.documentElement.classList.toggle('bb-light', light);
    document.documentElement.dataset.theme = value.theme;
    document.documentElement.dataset.reducedMotion = String(
      value.reduced_motion,
    );
  };
  useEffect(() => {
    const token = generation.current;
    run(async () => {
      const response = await bridge.Show();
      if (token !== generation.current) return;
      if (response.error?.code === 'not_found') return;
      const value = result(response);
      if (response.workspace_id) {
        client.select(response.workspace_id);
        setWorkspace({ ...value, workspace_id: response.workspace_id });
      }
    });
  }, [bridge]);
  useEffect(() => {
    const media = window.matchMedia?.('(prefers-color-scheme: light)');
    const update = () => {
      if (
        !document.documentElement.dataset.theme ||
        document.documentElement.dataset.theme === 'system'
      )
        applyAppearance({
          theme: 'system',
          reduced_motion:
            document.documentElement.dataset.reducedMotion === 'true',
        });
    };
    media?.addEventListener('change', update);
    update();
    return () => media?.removeEventListener('change', update);
  }, []);
  useEffect(() => {
    if (workspace)
      run(async () => {
        const settings = await client.call('settings.show');
        applyAppearance(settings.sections.appearance.value);
        if (settings.sections.appearance.error)
          throw new Error(
            'Saved appearance could not be read. Open Settings to save a valid preference.',
          );
      });
  }, [workspace?.workspace_id]);
  const selectWorkspace = async () => {
    const token = generation.current;
    const response = await bridge.SelectWorkspace(path, name, create);
    if (token !== generation.current) return;
    const value = result(response);
    client.select(value.workspace_id);
    setWorkspace(value);
    setScreen('Library');
  };
  const navigation = [
    'Library',
    'Jobs',
    'Pipelines',
    'Speakers',
    'Terms',
    'Settings',
  ];
  const header = (
    <header className="app-header">
      <div>
        <strong>insonic</strong>
        <span>A ShruggieTech project</span>
      </div>
      <nav aria-label="Workspace views">
        {navigation.map((v) => (
          <Button
            key={v}
            variant={screen === v ? 'primary' : 'ghost'}
            disabled={!workspace}
            aria-current={screen === v ? 'page' : undefined}
            onClick={() => {
              ++generation.current;
              client.invalidate();
              setScreen(v);
              setError('');
              setNotice('');
            }}
          >
            {v}
          </Button>
        ))}
      </nav>
      <a
        className="help-link bb-control"
        href="/assets/help/index.html"
        target="_blank"
        rel="noreferrer"
      >
        Offline help
      </a>
    </header>
  );
  return (
    <AppFrame host="wails" layout="full-bleed" header={header}>
      <AppFrameEnvironmentBridge />
      <div className="desktop-content">
        <h1>{screen}</h1>
        <Card heading="Workspace">
          <p>
            {workspace
              ? `Selected: ${workspace.display_name ?? workspace.workspace_id}`
              : 'Choose where your library and durable jobs live.'}
          </p>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              ++generation.current;
              client.invalidate();
              run(selectWorkspace);
            }}
          >
            <Input label="Workspace path" value={path} onChange={setPath} />
            <Actions>
              <Button
                variant="secondary"
                onClick={() =>
                  run(async () => {
                    const chosen = await bridge.Choose('workspace');
                    if (chosen) setPath(chosen);
                  })
                }
              >
                Choose workspace
              </Button>
              <Check
                label="Create a new workspace"
                value={create}
                onChange={setCreate}
              />
            </Actions>
            {create && (
              <Input label="Workspace name" value={name} onChange={setName} />
            )}
            <Button type="submit">
              {create ? 'Create workspace' : 'Open workspace'}
            </Button>
          </form>
        </Card>
        <div role="status" aria-live="polite">
          {busy > 0 ? 'Working…' : notice}
        </div>
        {error && (
          <div role="alert" className="error">
            <p>{error}</p>
            <p>
              Refresh current state before retrying a revision conflict. A
              readback failure can follow a persisted change.
            </p>
          </div>
        )}
        {workspace ? (
          <section
            key={`${workspace.workspace_id}:${screen}`}
            aria-label={`${screen} controls`}
          >
            {screen === 'Library' ? (
              <Library client={client} run={run} />
            ) : screen === 'Jobs' ? (
              <Jobs client={client} run={run} />
            ) : screen === 'Pipelines' ? (
              <Pipelines client={client} run={run} />
            ) : screen === 'Speakers' ? (
              <Speakers client={client} run={run} />
            ) : screen === 'Terms' ? (
              <Terms client={client} run={run} />
            ) : (
              <Settings
                client={client}
                run={run}
                appearance={applyAppearance}
              />
            )}
          </section>
        ) : (
          <EmptyState
            id="workspace-empty"
            heading="Open your voice archive"
            description="Create or open a workspace, then import audio or video."
          />
        )}
      </div>
    </AppFrame>
  );
}
