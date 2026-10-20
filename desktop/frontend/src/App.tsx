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
import {Setup} from './setup';
import {BackendSetup} from './backend-setup';
import { ImportPresets } from './import-presets';
import { Explore } from './explore';
import { SpeakerModels } from './speaker-models';
import {bindHelpNavigation} from './help';
import {bindNativeFileMenu} from './native-menu';
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
  useEffect(()=>{
    if(!bridge.OpenFileMenu)return;
    const unbind=bindNativeFileMenu(document,async()=>{result(await bridge.OpenFileMenu!());},setError);
    void bridge.NativeMenuReady?.().catch(()=>setError('Native menu readiness could not be established.'));
    return unbind;
  },[bridge]);
  const [helpOpening,setHelpOpening]=useState(false);
  const [helpURL,setHelpURL]=useState('');
  const helpCleanup=useRef<(()=>void)|undefined>(undefined);
  useEffect(()=>()=>helpCleanup.current?.(),[]);
  const openReference=async(url:string)=>{
    try {
      if(!bridge.OpenHelpReference)throw new Error('Web reference dispatcher unavailable.');
      result(await bridge.OpenHelpReference(url));setNotice('Web reference sent to your system browser.');
    }catch{setError('This web reference could not be opened. Copy its address into your browser.');}
  };
  const helpLoaded=(frame:HTMLIFrameElement)=>{
    helpCleanup.current?.();
    const doc=frame.contentDocument;
    if(!doc||doc.URL==='about:blank')return;
    const unbindLinks=bindHelpNavigation(doc,url=>void openReference(url),()=>setError('This help link is not a web reference or installed help page.'));
    const unbindMenu=bindNativeFileMenu(doc,bridge.OpenFileMenu?async()=>{result(await bridge.OpenFileMenu!());}:undefined,setError);
    helpCleanup.current=()=>{unbindLinks();unbindMenu();};
    setNotice('Offline documentation opened.');
  };
  const closeHelp=()=>{helpCleanup.current?.();helpCleanup.current=undefined;setHelpURL('');setNotice('');(document.querySelector('.help-action button') as HTMLElement)?.focus();};
  const openHelp=async()=>{
    setHelpOpening(true);setError('');setNotice('');
    try {
      if(!bridge.OpenHelp)throw new Error('Offline help is unavailable. Open the installed help directory or use installer Repair.');
      const help=result(await bridge.OpenHelp());
      if(typeof help?.url!=='string'||!help.url.startsWith('/assets/help/'))throw new Error('Installed offline help returned an invalid local entry. Use installer Repair.');
      setHelpURL(help.url);
    }catch(error){setError(error instanceof Error?error.message:'Offline help could not be opened. Use installer Repair.')}
    finally{setHelpOpening(false)}
  };
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
      if (response.error?.code === 'not_found') {
        if(bridge.SetupDraft)setScreen('Setup');
        return;
      }
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
    'Explore',
    'Jobs',
    'Pipelines',
    'Import presets',
    'Speakers',
    'Terms',
    'Models',
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
      {bridge.SetupDraft&&<Button variant="secondary" onClick={()=>{++generation.current;client.invalidate();setScreen('Setup');setError('');setNotice('')}}>Set up library</Button>}
      <div className="help-action"><Button variant="secondary" disabled={helpOpening} onClick={()=>void openHelp()}>Offline help</Button></div>
    </header>
  );
  return (
    <AppFrame host="wails" layout="full-bleed" header={header}>
      <AppFrameEnvironmentBridge />
      <div className="desktop-content">
        <h1>{screen}</h1>
        {helpURL&&<Card heading="Offline documentation"><Button variant="secondary" onClick={closeHelp}>Close documentation</Button><iframe className="offline-help-frame" title="Installed offline documentation" src={helpURL} sandbox="allow-scripts allow-same-origin allow-downloads" onLoad={event=>helpLoaded(event.currentTarget)} onError={()=>{setHelpURL('');setError('Offline help could not be loaded. Use installer Repair to restore matching help files.')}}/></Card>}
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
              if(!create||!bridge.BackendSetup)run(selectWorkspace);
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
            {create&&bridge.BackendSetup?<BackendSetup bridge={bridge} run={run} path={path} name={name} onConfigured={value=>{client.select(value.workspace_id);setWorkspace(value);setPath(value.path);setScreen("Library")}}/>:<Button type="submit">
              {create ? 'Create workspace' : 'Open workspace'}
            </Button>}
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
        {screen==='Setup'&&bridge.SetupDraft ? <Setup client={client} onWorkspace={value=>{++generation.current;client.select(value.workspace_id);setWorkspace(value);setPath(value.path??'')}} onNavigate={view=>{++generation.current;client.invalidate();setScreen(view);setError('');setNotice('')}} onClose={()=>{setScreen('Library');setError('');setNotice('Setup can be reopened through Settings.')}}/> : workspace ? (
          <section
            key={`${workspace.workspace_id}:${screen}`}
            aria-label={`${screen} controls`}
          >
            {screen === 'Library' ? (
              <Library client={client} run={run} />
            ) : screen === 'Explore' ? (<Explore client={client} run={run}/>) : screen === 'Jobs' ? (
              <Jobs client={client} run={run} />
            ) : screen === 'Import presets' ? (<ImportPresets client={client} run={run}/>) : screen === 'Pipelines' ? (
              <Pipelines client={client} run={run} />
            ) : screen === 'Speakers' ? (
              <Speakers client={client} run={run} />
            ) : screen === 'Terms' ? (
              <Terms client={client} run={run} />
            ) : screen === 'Models' ? (
              <SpeakerModels client={client} run={run} />
            ) : (
              <Settings
                client={client}
                run={run}
                appearance={applyAppearance}
                onSetup={bridge.SetupDraft?()=>setScreen('Setup'):undefined}
                onWorkspace={value=>{client.select(value.workspace_id);setWorkspace(value);setPath(value.path);setScreen("Library")}}
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
