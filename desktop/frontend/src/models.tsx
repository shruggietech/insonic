// SPDX-License-Identifier: Apache-2.0
import { useEffect, useState } from 'react';
import { Actions, Button, Facts, Input, Select } from './components';
import { Client, type Obj } from './client';

type Run = (action: () => Promise<void>) => void;
export function modelLabel(model: Obj): string {
  const capabilities = model.manifest?.capabilities ?? model.capabilities ?? [];
  return `${model.name} (${model.model_version ?? model.version ?? model.id}, ${capabilities.join('/') || 'capability not inspected'}, ${model.state})`;
}

// References remain user inputs here. Only the runtime may resolve, pin or
// determine whether a bundle is compatible and verified for an election.
export function ModelChoice({ label, value, onChange, operation, client, run }: {
  label: string; value: string; onChange: (value: string) => void;
  operation: 'transcription' | 'diarization'; client: Client; run: Run;
}) {
  const [models, setModels] = useState<Obj[]>([]), [aliases, setAliases] = useState<Obj[]>([]);
  const [modelNext, setModelNext] = useState(''), [aliasNext, setAliasNext] = useState('');
  const [inspection, setInspection] = useState<Obj>();
  const load = async (more = false) => {
    const [installed, named] = await Promise.all([
      client.call('models.list', '', {limit:100, ...(more && modelNext ? {after_id:modelNext} : {})}),
      client.call('models.alias.list', '', {limit:100, ...(more && aliasNext ? {after_id:aliasNext} : {})}),
    ]);
    const merge = (old: Obj[], incoming: Obj[]) => more ? [...old, ...incoming.filter((item) => !old.some((o) => o.id === item.id))] : incoming;
    setModels((old) => merge(old, installed.items));
    setAliases((old) => merge(old, named.items));
    setModelNext(installed.next_id ?? ''); setAliasNext(named.next_id ?? '');
  };
  useEffect(() => { setInspection(undefined); run(() => load()); }, [client.workspace]);
  const choices = [
    ...aliases.filter((alias) => alias.state === 'active' && alias.target.operation === operation).map((alias) => {
      const target = models.find((model) => model.id === alias.target.id);
      const compatibility = target?.operation_compatibility?.[operation] === false ? ', incompatible with default adapter' : '';
      return {value:alias.name, label:`${alias.name} → ${alias.target.kind}:${alias.target.id ?? alias.target.remote_model} (${target?.model_version ?? target?.version ?? 'exact target'}, ${operation}, ${target?.state ?? 'resolve to inspect'})${compatibility}`};
    }),
    ...models.map((model) => {
      const capabilities=model.manifest?.capabilities??model.capabilities??[];
      const compatibility = model.operation_compatibility?.[operation] === false ? ', incompatible with default adapter' : capabilities.length&&!capabilities.includes(operation)?', incompatible capability':'';
      return {value:model.id, label:modelLabel(model)+compatibility};
    }),
  ];
  return <>
    <Select label={label} value={value} onChange={(reference) => {setInspection(undefined);onChange(reference);}} options={[
      {value:'',label:`Choose a ${operation} model`},
      ...(value && !choices.some((choice) => choice.value === value) ? [{value,label:value}] : []),
      ...choices,
    ]} />
    <Input label={`${label} reference`} value={value} onChange={(reference) => {setInspection(undefined);onChange(reference);}}
      description="Alias, exact model ID, base:ID, speaker:ID or configured source selector. Missing verified files acquire automatically when you submit processing." />
    <Actions>
      <Button variant="secondary" onClick={() => run(() => load())}>Refresh {label.toLowerCase()} choices</Button>
      {(modelNext || aliasNext) && <Button variant="secondary" onClick={() => run(() => load(true))}>More {label.toLowerCase()} choices</Button>}
      <Button variant="secondary" onClick={() => run(async () => {setInspection(await client.call('models.resolve','',{reference:value,operation}));})}>Inspect {label.toLowerCase()} reference</Button>
    </Actions>
    {inspection && <><p>{inspection.compatible ? 'Compatible reference' : 'Incompatible reference'}: {inspection.state}. Inspection does not acquire files or execute models.</p><Facts value={inspection}/></>}
  </>;
}
