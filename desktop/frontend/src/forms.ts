// SPDX-License-Identifier: Apache-2.0
import type { Obj } from './client';
export const lines = (value: string) =>
  value
    .split(/\r?\n/)
    .map((s) => s.trim())
    .filter(Boolean);
export function uuid(): string {
  if (typeof crypto.randomUUID === 'function') return crypto.randomUUID();
  // Native custom schemes need not be secure contexts. getRandomValues still
  // supplies cryptographic entropy when the secure-context UUID API is absent.
  const bytes = crypto.getRandomValues(new Uint8Array(16));
  bytes[6] = (bytes[6] & 0x0f) | 0x40;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;
  const hex = Array.from(bytes, (byte) =>
    byte.toString(16).padStart(2, '0'),
  ).join('');
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}
export function pipelinePayload(values: Obj, previous?: Obj): Obj {
  const id = previous?.id ?? uuid();
  const stage = (kind: 'recognition' | 'diarization') => {
    const hosted = values[`${kind}_mode`] === 'hosted';
    const old = previous?.configuration?.[kind] ?? {};
    const defaults = {
      ...old,
      adapter: hosted
        ? 'insonic-http'
        : kind === 'recognition'
          ? 'faster-whisper'
          : 'pyannote',
      contract_version: '1',
      mode: hosted ? 'hosted' : 'local',
      limits: {
        max_audio_bytes: Number(values[`${kind}_audio_limit`]),
        max_response_bytes: Number(values[`${kind}_response_limit`]),
        timeout_ms: Number(values[`${kind}_timeout`]),
      },
    };
    // Full-stage replacement removes fields that are illegal for a new mode.
    const output: Obj = {
      adapter: defaults.adapter,
      contract_version: defaults.contract_version,
      mode: defaults.mode,
      limits: defaults.limits,
    };
    if (hosted) {
      output.remote_model = values[`${kind}_remote_model`];
      output.endpoint = values[`${kind}_endpoint`];
      if (values[`${kind}_credential`])
        output.credential_id = values[`${kind}_credential`];
      output.capabilities = [
        kind === 'recognition' ? 'transcription' : 'diarization',
      ];
      if (kind === 'recognition' && values.hints_supported) {
        output.supports_hints = true;
        output.max_hint_bytes = Number(values.hint_limit);
      }
    } else output.model_id = values[`${kind}_model`];
    if (kind === 'recognition')
      output.recognition = {
        ...(hosted ? {} : { device: values.recognition_device }),
        language: values.language,
        hints: lines(values.hints ?? ''),
      };
    else
      output.diarization = {
        ...(hosted ? {} : { device: values.diarization_device }),
        min_speakers: Number(values.min_speakers || 0),
        max_speakers: Number(values.max_speakers || 0),
      };
    return output;
  };
  return {
    expected_revision: previous?.revision ?? 0,
    pipeline: {
      id,
      name: values.name,
      preset: values.preset,
      revision: previous?.revision ?? 0,
      configuration: {
        ...(values.processing_enabled===false?{}:{
        recognition: stage('recognition'),
        diarization: stage('diarization'),
        quality: {
          enabled: values.quality,
          min_speech_coverage: Number(values.min_speech_coverage ?? 0.2),
          max_overlap_fraction: Number(values.max_overlap_fraction ?? 0.2),
          short_turn_us: Number(values.short_turn_us ?? 250000),
        },
        }),
        ...(values.speaker_training?.trim()?{speaker_training:JSON.parse(values.speaker_training)}:{}),
      },
    },
  };
}
export function pipelineValues(p?: Obj): Obj {
  const c = p?.configuration ?? {};
  const value: Obj = {
    name: p?.name ?? '',
    preset: p?.preset ?? 'local',
    processing_enabled: !p||!!c.recognition||!!c.diarization,
    speaker_training: c.speaker_training?JSON.stringify(c.speaker_training,null,2):'',
    quality: c.quality?.enabled ?? true,
    language: c.recognition?.recognition?.language ?? 'en',
    hints: (c.recognition?.recognition?.hints ?? []).join('\n'),
    min_speakers: c.diarization?.diarization?.min_speakers ?? '',
    max_speakers: c.diarization?.diarization?.max_speakers ?? '',
    hints_supported: c.recognition?.supports_hints ?? false,
    hint_limit: c.recognition?.max_hint_bytes ?? 200,
    min_speech_coverage: c.quality?.min_speech_coverage ?? 0.2,
    max_overlap_fraction: c.quality?.max_overlap_fraction ?? 0.2,
    short_turn_us: c.quality?.short_turn_us ?? 250000,
  };
  for (const kind of ['recognition', 'diarization']) {
    const s = c[kind] ?? {};
    value[`${kind}_mode`] = s.mode ?? 'local';
    value[`${kind}_model`] = s.model_id ?? '';
    value[`${kind}_remote_model`] = s.remote_model ?? '';
    value[`${kind}_endpoint`] = s.endpoint ?? '';
    value[`${kind}_credential`] = s.credential_id ?? '';
    value[`${kind}_device`] = s[kind]?.device ?? 'cpu';
    value[`${kind}_audio_limit`] = s.limits?.max_audio_bytes ?? 134217728;
    value[`${kind}_response_limit`] = s.limits?.max_response_bytes ?? 16777216;
    value[`${kind}_timeout`] = s.limits?.timeout_ms ?? 600000;
  }
  return value;
}
export function speakerPayload(values: Obj, previous?: Obj) {
  const speaker = previous?.speaker;
  const id = speaker?.id ?? uuid();
  const aliases = lines(values.aliases).map((text) => {
    const prior = previous?.aliases?.find((a: Obj) => a.text === text);
    return prior
      ? { ...prior, text }
      : {
          id: uuid(),
          speaker_id: id,
          text,
          language: values.language,
          scope: values.scope,
          state: 'active',
          provenance: {},
        };
  });
  return {
    expected_revision: speaker?.revision ?? 0,
    speaker: {
      id,
      name: values.name,
      revision: speaker?.revision ?? 0,
      state: values.state,
    },
    aliases,
  };
}
export function termPayload(values: Obj, previous?: Obj) {
  return {
    expected_revision: previous?.revision ?? 0,
    term: {
      id: previous?.id ?? uuid(),
      revision: previous?.revision ?? 0,
      canonical: values.canonical,
      variants: lines(values.variants),
      language: values.language,
      context: values.context,
      state: values.state,
      speaker_id: values.speaker_id || null,
      alias_id: previous?.alias_id ?? null,
      provenance: previous?.provenance ?? {},
    },
  };
}
