// SPDX-License-Identifier: Apache-2.0
import { useEffect, useRef, useState } from 'react';
import { Actions, Button, Card, Facts, Input, Select } from './components';
import { Client, result, RuntimeError, StaleResponse, type Obj } from './client';
import { uuid } from './forms';
import { BackendSetup } from './backend-setup';
import { Workers } from './workers';
import { Assistance } from './assistance';
import { Jobs, type Run } from './screens';
const stages = ['workspace', 'backends', 'processing', 'services', 'import'] as const;
type Stage = typeof stages[number];
const titles: Record<Stage, string> = { workspace: 'Workspace and location', backends: 'Backend profiles', processing: 'Optional local processing', services: 'Optional connected services', import: 'First import and readiness' };
type Pending = {
    kind: 'draft' | 'journey';
    id: string;
    input: Obj;
    workspace: string;
    generation: number;
};
export function Setup({ client, onWorkspace, onNavigate, onClose }: {
    client: Client;
    onWorkspace: (value: Obj) => void;
    onNavigate: (view: string) => void;
    onClose: () => void;
}) {
    const [draft, setDraft] = useState<Obj>(), [journey, setJourney] = useState<Obj>(), [path, setPath] = useState(''), [name, setName] = useState('My voice archive'), [location, setLocation] = useState<Obj>(), [stage, setStage] = useState<Stage>('workspace');
    const [error, setError] = useState(''), [notice, setNotice] = useState(''), [busy, setBusy] = useState(0), [pending, setPending] = useState<Pending>(), [readiness, setReadiness] = useState<Obj>();
    const [processing, setProcessing] = useState('none'), [pipelines, setPipelines] = useState<Obj>({ items: [] }), [pipeline, setPipeline] = useState<Obj>(), [backendSelections, setBackendSelections] = useState<Obj>();
    const alive = useRef(true), ownership = useRef(0), draftPlan = useRef<Obj | undefined>(undefined), draftCredentialBackend = useRef('native'), heading = useRef<HTMLHeadingElement>(null), alert = useRef<HTMLDivElement>(null), fence = useRef(0);
    const applyJourney = (out: Obj) => { setJourney(out); setStage(out.state.current_stage); setProcessing(out.state.choices.processing_mode); setPipeline(out.state.choices.pipeline_id ? { id: out.state.choices.pipeline_id, revision: out.state.choices.pipeline_revision } : undefined); };
    const applyDraft = (out: Obj) => { setDraft(out); setPath(out.path); setName(out.plan?.display_name ?? 'My voice archive'); setStage(out.stage ?? 'workspace'); setLocation(out.location); draftPlan.current = out.plan ?? undefined; draftCredentialBackend.current = out.credential_backend ?? 'native'; };
    const load = async () => {
        const token = ++ownership.current, workspace = client.workspace;
        if (workspace) {
            const out = await client.call('setup.show');
            if (alive.current && token === ownership.current && workspace === client.workspace)
                applyJourney(out);
        }
        else {
            if (!client.bridge.SetupDraft)
                throw new Error('Use a matching desktop package with setup support.');
            const out = result(await client.bridge.SetupDraft('show', '{}', uuid()));
            if (alive.current && token === ownership.current && !client.workspace)
                applyDraft(out);
        }
    };
    const run: Run = async (action) => { setError(''); setNotice(''); setBusy(n => n + 1); try {
        await action();
    }
    catch (e) {
        if (!(e instanceof StaleResponse) && alive.current)
            setError(e instanceof Error ? e.message : 'The request failed. Read saved setup and retry.');
    }
    finally {
        if (alive.current)
            setBusy(n => Math.max(0, n - 1));
    } };
    useEffect(() => { alive.current = true; setPending(undefined); setReadiness(undefined); void run(load); return () => { alive.current = false; ownership.current++; fence.current++; }; }, [client, client.workspace]);
    useEffect(() => { heading.current?.focus(); }, [stage, !!journey, !!draft]);
    useEffect(() => { if (error)
        alert.current?.focus(); }, [error]);
    const accept = (out: Obj, kind: 'draft' | 'journey') => {
        // Accepted saves invalidate initial loads that observed an older revision.
        ownership.current++;
        if (kind === 'draft')
        applyDraft(out);
    else
        applyJourney(out); setPending(undefined); setNotice('Setup choices saved. Existing configuration and work remain authoritative.'); };
    const execute = async (request: Pending) => {
        if (request.workspace !== client.workspace || request.generation !== client.generation)
            throw new StaleResponse();
        setPending(request);
        try {
            const out = request.kind === 'draft' ? result(await client.bridge.SetupDraft!('save', JSON.stringify(request.input), request.id)) : await client.call('setup.save', '', request.input, undefined, request.id);
            if (!alive.current || request.workspace !== client.workspace || request.generation !== client.generation)
                throw new StaleResponse();
            accept(out, request.kind);
            return out;
        }
        catch (e) {
            if (e instanceof RuntimeError && ['conflict', 'invalid_request', 'incompatible_version', 'workspace_mismatch'].includes(e.code))
                setPending(undefined);
            throw e;
        }
    };
    const mutate = (kind: 'draft' | 'journey', input: Obj) => execute({ kind, id: uuid(), input: structuredClone(input), workspace: client.workspace, generation: client.generation });
    const changeLocation = (value: string) => {
        setPath(value);
        setLocation(undefined);
        if (value === draft?.path) {
            draftPlan.current = draft?.plan ? {...draft.plan, display_name: name} : undefined;
            return;
        }
        if (draftPlan.current) {
            const plan = structuredClone(draftPlan.current);
            plan.kind = 'backend-plan';
            delete plan.workspace_id;
            delete plan.credential_namespace_id;
            for (const profile of Object.values(plan.profiles ?? {}) as Obj[]) {
                delete profile.profile_id;
                delete profile.profile_revision;
            }
            draftPlan.current = plan;
        }
    };
    const openSaved = () => run(async () => {
        const value = result(await client.bridge.SelectWorkspace(draft!.path, draft!.plan?.display_name ?? name, false));
        onWorkspace(value);
        await load();
    });
    const saveDraft = async (next: Stage, status = 'active', plan = draftPlan.current) => mutate('draft', { expected_revision: draft?.revision ?? 0, path, plan: plan ?? { schema_version: '1.1.0', kind: 'backend-plan', display_name: name }, stage: next === 'workspace' ? 'workspace' : 'backends', status, credential_backend: draftCredentialBackend.current });
    const saveJourney = async (next: Stage, mark?: 'completed' | 'skipped', status = 'active', source = journey) => {
        const current = source ?? await client.call('setup.show');
        const state = structuredClone(current.state);
        state.status = status;
        state.current_stage = next;
        state.choices = { processing_mode: processing, ...(pipeline ? { pipeline_id: pipeline.id, pipeline_revision: pipeline.revision } : {}) };
        state.stages = state.stages.map((s: Obj) => s.state !== 'skipped' && !s.proof ? {...s, proof: current.observations[s.id]} : s);
        if (mark)
            state.stages = state.stages.map((s: Obj) => s.id === stage ? { id: s.id, state: mark, ...(mark === 'completed' ? { proof: current.observations[s.id] } : {}) } : s);
        return mutate('journey', { expected_revision: current.revision, state });
    };
    const observedJourney = async () => {
        const observed = await client.call('setup.show');
        if (observed.revision !== journey?.revision)
            throw new RuntimeError('conflict', 'Setup changed. Read saved setup before applying this stage.');
        return {...journey, observations: observed.observations};
    };
    const advance = (mark: 'completed' | 'skipped') => run(async () => {
        const next = stages[Math.min(stages.indexOf(stage) + 1, 4)];
        if (client.workspace) {
            const current = await observedJourney();
            await saveJourney(next, mark, stage === 'import' ? 'completed' : 'active', current);
            if(stage === 'import')onClose();
        }
        else
            await saveDraft(next);
    });
    const configured = async (value: Obj) => {
        onWorkspace(value);
        const current = await client.call('setup.show'), state = structuredClone(current.state);
        state.status = 'active';
        state.current_stage = 'processing';
        state.choices.processing_mode = processing;
        state.stages = state.stages.map((s: Obj) => ['workspace', 'backends'].includes(s.id) ? { id: s.id, state: 'completed', proof: current.observations[s.id] } : s);
        await mutate('journey', { expected_revision: current.revision, state });
    };
    const later = () => run(async () => { if (client.workspace)
        await saveJourney(stage, undefined, 'deferred');
    else
        await saveDraft(stage, 'deferred'); onClose(); });
    const back = () => run(async () => { const next = stages[Math.max(0, stages.indexOf(stage) - 1)]; if (client.workspace)
        await saveJourney(next);
    else
        await saveDraft(next); });
    const inspect = () => run(async () => setReadiness(await client.call('setup.readiness')));
    const loadPipelines = async (after = '') => { const page = await client.call('pipelines.list', '', { limit: 100, ...(after ? { after_id: after } : {}) }); setPipelines(old => ({ ...page, items: after ? [...old.items, ...page.items] : page.items })); };
    useEffect(() => { if (client.workspace && stage === 'processing')
        void run(() => loadPipelines()); if (client.workspace && stage === 'backends')
        void run(async () => setBackendSelections(await client.call('workspace.show'))); if (client.workspace && stage === 'import')
        void inspect(); }, [client.workspace, stage]);
    const selected = !!client.workspace, ready = selected ? !!journey : !!draft, blocked = busy > 0 || !!pending, savedLibrary = !!draft?.location?.existing_workspace && path === draft.path;
    return <section aria-label="First-run setup">
  <h2 ref={heading} tabIndex={-1}>{titles[stage]} (Step {stages.indexOf(stage) + 1} of 5)</h2>
  <p>Setup is optional. Save your choices as you go, skip optional stages or return later through Settings. Your existing library and durable work stay available.</p>
  <ol aria-label="Setup progress">{stages.map(id => { const derived = journey?.stages.find((s: Obj) => s.id === id); return <li key={id} aria-current={stage === id ? 'step' : undefined}>{titles[id]}: {stage === id ? 'current, ' : ''}{derived?.state ?? (id === 'workspace' && stage === 'backends' ? 'completed' : 'pending')}{derived?.correction && <p>{derived.correction}</p>}</li>; })}</ol>
  <div role="status" aria-live="polite">{busy ? 'Working on setup…' : notice}</div>
  {error && <div ref={alert} tabIndex={-1} role="alert" className="error"><p>{error}</p><p>Read saved setup before correcting a revision conflict. An interrupted response can follow a saved change.</p></div>}
  <Actions><Button variant="secondary" disabled={busy > 0} onClick={() => run(async () => { await load(); setNotice('Current saved setup read. Accepted settings and work have been retained.'); })}>Read saved setup</Button>{pending && <Button disabled={busy > 0} onClick={() => run(async () => { await execute(pending); })}>Retry the same save</Button>}</Actions>
  {ready && <>
   {!selected && savedLibrary && <Card heading="Resume your saved library"><p>A library already exists at {draft?.path}. Open it to resume accepted setup choices and durable work.</p><Button disabled={blocked} onClick={openSaved}>Open saved library and resume</Button></Card>}
   {stage === 'workspace' && <Card heading="Choose your library location">
    {selected ? <><p>This journey belongs to the selected library. Reopening setup preserves its identity and routes. Open or create a different library through the normal Workspace controls.</p><Facts value={journey?.paths}/><Button disabled={blocked} onClick={() => advance('completed')}>Continue</Button></> : <>
     <Input disabled={blocked} label="Library location" value={path} onChange={changeLocation} description="Choose a writable directory outside the application installation. A directory with an existing library can be opened without replacing it."/>
     <Input disabled={blocked} label="Library name" value={name} onChange={v => { setName(v); if (draftPlan.current)
                    draftPlan.current = { ...draftPlan.current, display_name: v }; }}/>
     <Actions><Button disabled={blocked} variant="secondary" onClick={() => run(async () => { const chosen = await client.bridge.Choose('workspace'); if (chosen) {
                    changeLocation(chosen);
                } })}>Choose library directory</Button><Button disabled={blocked} variant="secondary" onClick={() => run(async () => setLocation(result(await client.bridge.SetupDraft!('location', JSON.stringify({ path }), uuid()))))}>Preview location and space</Button><Button disabled={blocked || !path.trim() || !name.trim()} onClick={() => run(async () => { await saveDraft('backends'); })}>Save location and continue</Button><Button disabled={blocked || !path.trim()} variant="secondary" onClick={() => run(async () => { const value = result(await client.bridge.SelectWorkspace(path, name, false)); onWorkspace(value); await load(); })}>Open existing library</Button></Actions>
     <p>Control files contain catalog and configuration. Managed media and model artifacts can be much larger. Cache and temporary processing files are disposable; runtime environments are prepared separately. Preview reports observed free space, or explicitly unknown capacity. It creates no destination directory.</p>
     {location && <Facts value={{...location, available_bytes: location.available_bytes ?? null}}/>}
    </>}
   </Card>}
   {stage === 'backends' && (selected ? <Card heading="Current backend selections"><p>The selected library already owns its backend configuration. Changes to populated backends use the existing migration and backup operations.</p><Facts value={backendSelections}/><Button disabled={blocked} onClick={() => advance('completed')}>Continue</Button></Card> : savedLibrary ? <Card heading="Saved backend configuration"><p>Resume the existing library above. To choose a new library location, use Back. Provisioning remains a separate election for a fresh destination.</p></Card> : <fieldset disabled={blocked}><legend>Fresh backend provisioning</legend><BackendSetup bridge={client.bridge} run={run} path={path} name={name} initialPlan={draft?.plan} initialCredentialBackend={draft?.credential_backend} onDraft={(plan, mode) => { draftPlan.current = plan; draftCredentialBackend.current = mode; }} onPlan={async (plan) => { await saveDraft('backends', 'active', plan); }} onConfigured={configured}/></fieldset>)}
   {stage === 'processing' && selected && <>
    <Card heading="Processing preferences"><p>Ordinary import and supplied subtitles work without an account or model download. Optional preparation uses existing previews and durable jobs.</p><Select disabled={blocked} label="Processing preference" value={processing} onChange={v => { setProcessing(v); if (v === 'none')
                setPipeline(undefined); }} options={[{ value: 'none', label: 'Import and supplied subtitles only' }, { value: 'local', label: 'Optional local transcription and diarization' }, { value: 'connected', label: 'Elected connected processing pipeline' }, { value: 'custom', label: 'Existing custom processing pipeline' }]}/>{['connected', 'custom'].includes(processing) && <Button variant="secondary" disabled={blocked} onClick={() => run(async()=>{await saveJourney(stage);onNavigate('Pipelines')})}>Open pipeline configuration</Button>}{processing !== 'none' && <><Select disabled={blocked} label="Setup pipeline" value={pipeline?.id ?? ''} onChange={v => { const selected = pipelines.items.find((p: Obj) => p.id === v); setPipeline(selected ? { id: selected.id, revision: selected.revision } : undefined); }} options={[{ value: '', label: 'No setup pipeline selected' }, ...(pipeline && !pipelines.items.some((p: Obj) => p.id === pipeline.id) ? [{ value: pipeline.id, label: 'Saved pipeline ' + pipeline.id }] : []), ...pipelines.items.map((p: Obj) => ({ value: p.id, label: p.name + ' (revision ' + p.revision + ')' }))]}/><Actions><Button variant="secondary" disabled={blocked} onClick={() => run(() => loadPipelines())}>Refresh setup pipelines</Button>{pipelines.next_id && <Button variant="secondary" disabled={blocked} onClick={() => run(() => loadPipelines(pipelines.next_id))}>More setup pipelines</Button>}</Actions><p>Save the stage to retain the exact pipeline revision. Readiness follows that saved election; changing a pipeline later requires refreshing and selecting its current revision.</p></>}
    </Card>
    {processing === 'local' && <><p>Hugging Face model access uses a task-specific protected token form. Its opaque reference is generated automatically. Registering a token does not establish provider authentication.</p><Workers client={client} run={run}/><Jobs client={client} run={run}/></>}
   </>}
   {stage === 'services' && selected && <><p>Connected query assistance is optional. Discovery reads service metadata; electing and saving a route is separate. The protected bearer-token form automatically generates its opaque reference. Registration, saved configuration and a successful synthetic test are separate observations.</p><Assistance client={client} run={run} fence={fence} currentQuery={() => ({})} adopt={() => onNavigate('Explore')} executed={() => onNavigate('Explore')}/></>}
   {stage === 'import' && selected && <Card heading="Ready actions and remaining prerequisites">
    <p>Inspect current task readiness. Missing optional processing dependencies leave ordinary import available. Choosing Open import does not begin an import or acquire a model.</p>
    <p>The saved setup pipeline is used for readiness inspection. Open import uses the ordinary import choices; select your intended import preset there.</p>
    <Button variant="secondary" disabled={busy > 0} onClick={inspect}>Inspect setup readiness</Button>{readiness && <Facts value={readiness.tasks ?? readiness}/>}
    <Facts value={journey?.actions}/>
    <Actions><Button onClick={() => onNavigate('Library')}>Open import</Button><Button variant="secondary" onClick={() => onNavigate('Settings')}>Open Settings</Button><Button variant="secondary" onClick={() => onNavigate('Jobs')}>Open Jobs</Button><Button disabled={blocked} onClick={() => run(async () => { await saveJourney('import', 'completed', 'completed', await observedJourney()); onClose(); })}>Finish setup</Button></Actions>
   </Card>}
   <Actions><Button variant="secondary" disabled={blocked || stage === 'workspace'} onClick={back}>Back</Button>{selected && ['processing', 'services'].includes(stage) && <Button disabled={blocked} onClick={() => advance('completed')}>Save stage and continue</Button>}{selected && ['processing', 'services', 'import'].includes(stage) && <Button variant="secondary" disabled={blocked} onClick={() => advance('skipped')}>Skip this stage</Button>}<Button variant="secondary" disabled={blocked} onClick={later}>Set up later</Button></Actions>
  </>}
 </section>;
}
