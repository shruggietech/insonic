// SPDX-License-Identifier: Apache-2.0
import { useEffect, useLayoutEffect, useRef, useState } from 'react';
import {
  Actions,
  Area,
  Button,
  Card,
  Check,
  Facts,
  Input,
  Select,
  StatusBadge,
  Table,
} from './components';
import { appendCapture, Client, rationalSeconds, type Obj } from './client';
import { ModelChoice } from './models';
import { SpeakerTraining, RosterMatching } from './speaker-models';
import {
  lines,
  pipelinePayload,
  pipelineValues,
  speakerPayload,
  termPayload,
  uuid,
} from './forms';
export type Run = (action: () => Promise<void>) => void;
type Props = { client: Client; run: Run };
const states = ['active', 'inactive'];
// Speaker list entries carry identity and aliases together; selectors and
// tables display the nested person rather than discarding that wire shape.
const catalogItem = (operation: string, item: Obj): Obj =>
  operation === 'speakers.list' ? item.speaker : item;
export function Pager({ next, load }: { next: unknown; load: () => void }) {
  return next ? (
    <Button variant="secondary" onClick={load}>
      Next page
    </Button>
  ) : null;
}
function CatalogChoice({
  label,
  operation,
  value,
  onChange,
  client,
  run,
  empty = 'Any',
}: {
  label: string;
  operation: string;
  value: string;
  onChange: (value: string) => void;
  client: Client;
  run: Run;
  empty?: string;
}) {
  const [options, setOptions] = useState<Obj[]>([]),
    [next, setNext] = useState('');
  const load = async (after = '') => {
    const page = await client.call(operation, '', {
      limit: 100,
      ...(after ? { after_id: after } : {}),
    });
    const items = page.items.map((item: Obj) => catalogItem(operation, item));
    setOptions((old) =>
      after
        ? [
            ...old,
            ...items.filter((item: Obj) => !old.some((o) => o.id === item.id)),
          ]
        : items,
    );
    setNext(page.next_id);
  };
  useEffect(() => {
    run(() => load());
  }, [client.workspace]);
  return (
    <>
      <Select
        label={label}
        value={value}
        onChange={onChange}
        options={[
          { value: '', label: empty },
          ...(value && !options.some((o) => o.id === value)
            ? [{ value, label: value }]
            : []),
          ...options.map((o) => ({
            value: o.id,
            label: `${o.name}${o.model_version ? ` (${o.model_version}, ${o.state})` : ''}`,
          })),
        ]}
      />
      <Actions>
        <Button variant="secondary" onClick={() => run(() => load())}>
          Refresh {label.toLowerCase()} choices
        </Button>
        {next && (
          <Button variant="secondary" onClick={() => run(() => load(next))}>
            More {label.toLowerCase()} choices
          </Button>
        )}
      </Actions>
    </>
  );
}
function useList(client: Client, operation: string, run: Run) {
  const [page, setPage] = useState<Obj>({ items: [] });
  const load = async (after = '') => {
    const page = await client.call(operation, '', {
      limit: 25,
      ...(after ? { after_id: after } : {}),
    });
    setPage({
      ...page,
      items: page.items.map((item: Obj) => catalogItem(operation, item)),
    });
  };
  useEffect(() => {
    run(() => load());
  }, [client.workspace]);
  return { page, load };
}
export function Library({ client, run }: Props) {
  const { page, load } = useList(client, 'media.list', run);
  const [entry, setEntry] = useState<Obj>();
  const [metadata, setMetadata] = useState<Obj>();
  const [targetAudio,setTargetAudio]=useState(false),[replaceAudio,setReplaceAudio]=useState(false),
    [existingTranscript,setExistingTranscript]=useState(''),[transcriptApplies,setTranscriptApplies]=useState(false),
    [existingRoster,setExistingRoster]=useState(''),[declareRoster,setDeclareRoster]=useState(false),[knownSpeakers,setKnownSpeakers]=useState('');
  const [roster,setRoster]=useState<Obj>(),[rosterMode,setRosterMode]=useState<'add'|'remove'|'replace'>('add'),[rosterRefs,setRosterRefs]=useState('');
  const [source, setSource] = useState(''),
    [subtitle, setSubtitle] = useState(''),
    [title, setTitle] = useState(''),
    [transcriptOnly, setTranscriptOnly] = useState(false),
    [newEntry, setNewEntry] = useState(false);
  const [recordTarget, setRecordTarget] = useState(''),
 [replaceTranscript,setReplaceTranscript]=useState(false),
 [admissionAttribution,setAdmissionAttribution]=useState('auto'),
 [transcriptHint,setTranscriptHint]=useState(''),
 [embeddedIndex,setEmbeddedIndex]=useState(''),
 [embeddedLanguage,setEmbeddedLanguage]=useState(''),
 [mediaCredential,setMediaCredential]=useState(''),
 [transcriptCredential,setTranscriptCredential]=useState(''),
 [transcriptMax,setTranscriptMax]=useState('16777216'),
 [transcriptTimeout,setTranscriptTimeout]=useState('120000'),
 [playbackTrack,setPlaybackTrack]=useState('');
 const [date, setDate] = useState(''),
    [precision, setPrecision] = useState('date'),
    [zone, setZone] = useState(''),
    [fold, setFold] = useState(''),
    [gap, setGap] = useState('');
  const [relocation, setRelocation] = useState('');
  const [cues, setCues] = useState<Obj>(),
    [mappings, setMappings] = useState<Obj>({ items: [] }),
    [pipelines, setPipelines] = useState<Obj[]>([]),
    [speakers, setSpeakers] = useState<Obj[]>([]);
  const [pipeline, setPipeline] = useState(''),
    [transcription, setTranscription] = useState(''),
    [diarization, setDiarization] = useState('run'),
    [recognitionModel, setRecognitionModel] = useState(''),
    [diarizationModel, setDiarizationModel] = useState(''),
    [format, setFormat] = useState('srt'),
    [destination, setDestination] = useState(''),
    [strict, setStrict] = useState(false);
  const [pipelineCursor, setPipelineCursor] = useState(''),
    [speakerCursor, setSpeakerCursor] = useState('');
  const [turnLabel, setTurnLabel] = useState('Voice 1'),
    [turnStart, setTurnStart] = useState('0'),
    [turnEnd, setTurnEnd] = useState('1000'),
    [turns, setTurns] = useState<Obj[]>([]);
  const [participationCue, setParticipationCue] = useState(''),
    [participationLabel, setParticipationLabel] = useState(''),
    [participation, setParticipation] = useState<Obj[]>([]);
  const [localVoice, setLocalVoice] = useState(''),
    [person, setPerson] = useState('');
  const [playback, setPlayback] = useState<Obj>();
  const media = useRef<HTMLMediaElement | null>(null),
    selected = useRef(0);
  const pendingSeek = useRef<{url:string;seconds:number} | undefined>(undefined);
  const applyPendingSeek = () => {
    const pending = pendingSeek.current, node = media.current;
    if (pending && node?.getAttribute('src') === pending.url)
      node.currentTime = pending.seconds;
  };
  const confirmPendingSeek = () => {
    const pending = pendingSeek.current, node = media.current;
    if (pending && node?.getAttribute('src') === pending.url && node.readyState >= 2 &&
        !node.seeking && Math.abs(node.currentTime - pending.seconds) < 0.001)
      pendingSeek.current = undefined;
  };
  const entryID = entry?.id ?? entry?.media_id ?? '';
  const select = async (id: string) => {
    client.invalidate();
    const generation = ++selected.current;
    setEntry(undefined);
    setRoster(undefined);
    setRosterRefs('');
    setPlayback(undefined);
    setPlaybackTrack('');
    setCues(undefined);
    setMetadata(undefined);
    setMappings({ items: [] });
    setLocalVoice('');
    setPerson('');
    const nextEntry = await client.call('media.show', id);
    if (generation !== selected.current) return;
    setEntry(nextEntry);
    const [p, s, m, rosterResult] = await Promise.all([
      client.call('pipelines.list', '', { limit: 100 }),
      client.call('speakers.list', '', { limit: 100 }),
      client
        .call('recordings.mappings', id, { limit: 100 })
        .catch(() => ({ items: [] })),
      client.roster(id),
    ]);
    if (generation !== selected.current) return;
    if (pipeline && !p.items.some((value: Obj) => value.id === pipeline))
      p.items.push(await client.call('pipelines.show', pipeline));
    if (generation !== selected.current) return;
    setPipelines(p.items);
    setPipelineCursor(p.next_id);
    setSpeakers(s.items.map((item: Obj) => item.speaker));
    setSpeakerCursor(s.next_id);
    setMappings(m);
    setRoster(rosterResult);
    try {
      const current = await client.call('recordings.cues', id, { limit: 25 });
      if (generation === selected.current) setCues(current);
    } catch (error) {
      if ((error as Obj).code !== 'not_found') throw error;
    }
  };
  const refresh = async () => {
    if (entryID) await select(entryID);
    await load();
  };
  const mutateRoster=async(mode:'add'|'remove'|'replace'|'clear')=>{
    if (!roster) return;
    const references=mode==='clear'?[]:lines(rosterRefs);
    if((mode==='add'||mode==='remove')&&references.length===0)throw new Error('Enter at least one speaker reference for add or remove.');
    try {setRoster(await client.updateRoster(entryID,roster.revision??0,mode,references));setRosterRefs('');}
    catch(error){if((error as Obj).code==='conflict') setRoster(await client.roster(entryID));throw error;}
  };
  const capture = async (section = 'metadata') => {
    setMetadata(
      appendCapture(
        await client.call('media.capture', entryID, {
          revision: entry?.revision,
          section,
          limit: 65536,
        }),
      ),
    );
  };
  const dateOptions = () => ({
    ...(date
      ? precision === 'date'
        ? { originated_on: date }
        : { originated_at: date }
      : {}),
    ...(zone ? { timezone: zone } : {}),
    ...(fold ? { dst_fold: fold } : {}),
    ...(gap ? { dst_gap: gap } : {}),
  });
  const play = async (seek?: number) => {
    const generation = selected.current;
    const response = playbackTrack!=='' && client.bridge.PlaybackTrack
 ? await client.bridge.PlaybackTrack(entryID,entry?.revision??0,cues?.revision??0,cues?.document_digest??'',Number(playbackTrack))
 : await client.bridge.Playback(
      entryID,
      entry?.revision ?? 0,
      cues?.revision ?? 0,
      cues?.document_digest ?? '',
    );
    if (generation !== selected.current) {
      if (response.result?.url)
        await client.bridge.ClosePlayback(response.result.url);
      return;
    }
    if (response.error) throw new Error(response.error.message);

    setPlayback({ ...response.result, seek });
  };
  useEffect(() => {
    return () => {
      if (playback?.url)
        void client.bridge.ClosePlayback(playback.url).catch(() => {});
    };
  }, [playback?.url]);
  useEffect(() => {
    if (!playback?.url) return;
    let active = true,
      checking = false;
    const generation = selected.current;
    const timer = setInterval(async () => {
      if (checking) return;
      checking = true;
      try {
        const response = await client.bridge.VerifyPlayback(playback.url);
        if (!active || generation !== selected.current) return;
        if (response.error) throw new Error(response.error.message);
      } catch {
        if (!active || generation !== selected.current) return;
        media.current?.pause();
        setPlayback(undefined);
        setCues(undefined);
        setMappings({ items: [] });
        setLocalVoice('');
        setPerson('');
        run(async () => {
          throw new Error(
            'Playback authority changed or expired. Refresh current results and reacquire playback.',
          );
        });
      } finally {
        checking = false;
      }
    }, 5000);
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, [playback?.url]);
  useEffect(
    () => () => {
      selected.current++;
    },
    [],
  );
  useLayoutEffect(() => {
    // Native decoders may lose a metadata-time seek while data becomes ready.
    // Keep the elected source position pending until seeked confirms it; later
    // readiness events must never rewind ordinary playback after confirmation.
    pendingSeek.current = playback?.seek === undefined ? undefined : {
      url:playback.url,
      seconds:Math.max(0,playback.seek-(playback.timeline_offset_seconds??0)),
    };
    if (media.current && media.current.readyState >= 1) applyPendingSeek();
  }, [playback]);
  return (
    <>
      <Card heading="Add audio or video">
        <form
          onSubmit={(e) => {
            e.preventDefault();
            run(async () => {
              if (!/^\d+$/.test(transcriptMax) || Number(transcriptMax)<1 || Number(transcriptMax)>16777216 || !/^\d+$/.test(transcriptTimeout) || Number(transcriptTimeout)<1 || Number(transcriptTimeout)>600000 || (embeddedIndex!=='' && (!/^\d+$/.test(embeddedIndex) || !Number.isSafeInteger(Number(embeddedIndex))))) throw new Error('Enter valid transcript limits and a nonnegative stream index.');
              await client.call('media.import', '', {
                kind: 'import-manifest',
                schema_version: '0.0.0',
                defaults: { ...dateOptions(),attribution:admissionAttribution,
 replace_transcript:replaceTranscript,
 ...(targetAudio?{replace_audio:replaceAudio,...(existingTranscript?{existing_transcript:existingTranscript}:{}),...(existingTranscript==='keep'?{transcript_applies:transcriptApplies}:{}),...(existingRoster?{existing_roster:existingRoster}:{})}:{}),
 ...(!targetAudio&&!transcriptOnly&&declareRoster?{known_speakers:lines(knownSpeakers)}:{}),
 ...(transcriptHint?{transcript_format:transcriptHint}:{}),
 ...(embeddedIndex!==''?{subtitle_stream_index:Number(embeddedIndex)}:{}),
 ...(embeddedLanguage?{subtitle_language:embeddedLanguage}:{}),
 transcript_max_bytes:Number(transcriptMax),transcript_timeout_ms:Number(transcriptTimeout),
 ...(admissionAttribution==='diarize'?{diarization_model_id:diarizationModel}:{}),
 },
                items: [
                  {
                    ...(transcriptOnly?{kind:'transcript',record:recordTarget,transcript:subtitle}:{kind:'media',source,...(targetAudio?{record:recordTarget}:{}),...(subtitle?{transcript:subtitle}:{}),...(!targetAudio&&newEntry?{new_entry:true}:{})}),
                    ...(mediaCredential&&!transcriptOnly?{credential_id:mediaCredential}:{}),
                    ...(transcriptCredential?{transcript_credential_id:transcriptCredential}:{}),
                    ...(title ? { title } : {}),
                  },
                ],
              });
              setSource('');
              await load();
            });
          }}
        >
          <Check label="Import transcript into an existing recording" value={transcriptOnly} onChange={(v)=>{setTranscriptOnly(v);if(v)setTargetAudio(false)}}/>
          <Check label="Target existing recording audio" value={targetAudio} onChange={(v)=>{setTargetAudio(v);if(v)setTranscriptOnly(false)}}/>
          {targetAudio && <>
            <Check label="Replace current audio" value={replaceAudio} onChange={setReplaceAudio}/>
            <p>Without replacement elected, the target is skipped before opening the candidate source.</p>
            {replaceAudio && <>
              <Select label="Existing transcript policy" value={existingTranscript} onChange={setExistingTranscript} options={[{value:'',label:'Choose when a transcript exists'},{value:'keep',label:'Keep current transcript'},{value:'clear',label:'Clear current transcript'},{value:'replace',label:'Replace with selected transcript'}]}/>
              {existingTranscript==='keep' && <Check label="Current transcript applies to replacement audio" value={transcriptApplies} onChange={setTranscriptApplies}/>}
              <Select label="Existing roster policy" value={existingRoster} onChange={setExistingRoster} options={[{value:'',label:'Choose when a roster is declared'},{value:'retain',label:'Retain declared roster'},{value:'clear',label:'Clear declared roster'}]}/>
            </>}
          </>}
          {!targetAudio&&!transcriptOnly && <>
            <Check label="Declare known speakers at import" value={declareRoster} onChange={setDeclareRoster}/>
            {declareRoster && <Area label="Known speaker IDs, names or aliases" value={knownSpeakers} onChange={setKnownSpeakers} description="One existing identity per line. An empty list declares an empty roster."/>}
          </>}
          {(transcriptOnly||targetAudio) && <Input label="Existing recording ID or exact title" value={recordTarget} onChange={setRecordTarget} required/>}
          <Input
            label="Media path or URL"
            value={source}
            onChange={setSource}
            required={!transcriptOnly}
          />
          <Button
            variant="secondary"
            onClick={() =>
              run(async () => {
                const path = await client.bridge.Choose('media');
                if (path) setSource(path);
              })
            }
          >
            Choose media
          </Button>
          <Input label="Title" value={title} onChange={setTitle} />
          <Input
            label="Transcript path or URL"
            value={subtitle}
            onChange={setSubtitle}
            required={transcriptOnly}
          />
          <Button
            variant="secondary"
            onClick={() =>
              run(async () => {
                const path = await client.bridge.Choose('subtitle');
                if (path) setSubtitle(path);
              })
            }
          >
            Choose subtitle
          </Button>
          <p>New recordings retain canonical audio and captured source facts. Owner-owned inputs remain untouched.</p>
          <Check label="Replace an existing current transcript" value={replaceTranscript} onChange={setReplaceTranscript}/>
          <Select label="Import speaker attribution" value={admissionAttribution} onChange={setAdmissionAttribution} options={[
            {value:'auto',label:'Auto (preserve assignments, then native or heuristic observations)'},{value:'native',label:'Native observations only'},{value:'off',label:'Preserve without deriving assignments'},{value:'diarize',label:'Run configured diarization without recognition'}
          ]}/>
          {admissionAttribution==='diarize' && <ModelChoice label="Import diarization model" value={diarizationModel} onChange={setDiarizationModel} operation="diarization" client={client} run={run}/>}
          <Select label="Transcript format" value={transcriptHint} onChange={setTranscriptHint} options={[
            {value:'',label:'Detect from bytes'},...['cueson','srt','vtt','ass','ssa'].map(value=>({value,label:value.toUpperCase()}))
          ]}/>
          <Input label="Embedded subtitle stream index (optional)" value={embeddedIndex} onChange={setEmbeddedIndex}/>
          <Input label="Embedded subtitle language (optional)" value={embeddedLanguage} onChange={setEmbeddedLanguage}/>
          <Input label="Media credential ID (optional)" value={mediaCredential} onChange={setMediaCredential}/>
          <Input label="Transcript credential ID (optional)" value={transcriptCredential} onChange={setTranscriptCredential}/>
          <Input label="Transcript byte limit" value={transcriptMax} onChange={setTranscriptMax}/>
          <Input label="Transcript timeout in milliseconds" value={transcriptTimeout} onChange={setTranscriptTimeout}/>
          <Check
            label="Create a distinct library entry for this source"
            value={newEntry}
            onChange={setNewEntry}
          />
          <Button type="submit">Import</Button>
        </form>
      </Card>
      <Card heading="Library">
        <Actions>
          <Button variant="secondary" onClick={() => run(refresh)}>
            Refresh library
          </Button>
        </Actions>
        <Table
          caption="Current audio and video"
          heads={['Title', 'Type', 'Availability', 'Details']}
        >
          {page.items.map((v: Obj) => (
            <tr key={v.id ?? v.media_id}>
              <td>{v.title ?? v.media_id}</td>
              <td>{v.class ?? 'Unknown'}</td>
              <td>{v.availability}</td>
              <td>
                <Button
                  variant="ghost"
                  onClick={() => run(() => select(v.id ?? v.media_id))}
                >
                  Open {v.title ?? 'media'}
                </Button>
              </td>
            </tr>
          ))}
        </Table>
        <Pager next={page.next_id} load={() => run(() => load(page.next_id))} />
      </Card>
      {entry && (
        <section aria-label="Recording details">
          <Card heading={entry.title ?? 'Recording details'}>
            <Facts
              value={{
                identity: entryID,
                source_digest: entry.digest,
                revision: entry.revision,
                availability: entry.availability,
                duration_microseconds: entry.duration_us,
                facts: entry.facts,
                dates: entry.dates,
              }}
            />
            <Actions>
              {(entry.facts?.streams??[]).filter((stream:Obj)=>stream.codec_type==='audio').length>1 && <Select label="Playback audio track" value={playbackTrack} onChange={setPlaybackTrack} options={[
                {value:'',label:'First audio track'},...(entry.facts?.streams??[]).filter((stream:Obj)=>stream.codec_type==='audio').map((stream:Obj)=>({value:String(stream.index),label:`Track ${stream.index}: ${stream.tags?.language??'unknown language'}, ${stream.channels??'?'} channels`}))
              ]}/>}
              <Button onClick={() => run(() => play())}>Play recording audio</Button>
              {entry.subtitle_publication_id && <Button variant="secondary" onClick={()=>run(async()=>{
                await client.call('media.import','',{kind:'import-manifest',schema_version:'0.0.0',defaults:{replace_transcript:replaceTranscript,attribution:admissionAttribution,...(admissionAttribution==='diarize'?{diarization_model_id:diarizationModel}:{})},items:[{record:entryID,transcript:'<managed>'}]});await refresh();
              })}>Convert managed legacy subtitle</Button>}
              <Button
                variant="secondary"
                onClick={() =>
                  run(async () => {
                    await capture();
                  })
                }
              >
                Inspect raw metadata and dates
              </Button>
              <Button
                variant="secondary"
                onClick={() =>
                  run(async () => {
                    await client.call('media.refresh', entryID, {});
                    await refresh();
                  })
                }
              >
                Recapture metadata
              </Button>
            </Actions>
            {metadata && (
              <details open>
                <summary>Captured metadata and provenance</summary>
                <Actions>
                  {['metadata', 'facts', 'dates'].map((section) => (
                    <Button
                      key={section}
                      variant="secondary"
                      onClick={() => run(() => capture(section))}
                    >
                      Read exact {section}
                    </Button>
                  ))}
                </Actions>
                <p>
                  Capture {metadata.section}, revision {metadata.revision},
                  SHA-256 {metadata.sha256}
                </p>
                <pre tabIndex={0} aria-label="Exact captured JSON bytes">
                  {metadata.text}
                </pre>
                {!metadata.complete && (
                  <Button
                    variant="secondary"
                    onClick={() =>
                      run(async () => {
                        setMetadata(
                          appendCapture(
                            await client.call('media.capture', entryID, {
                              section: metadata.section,
                              revision: metadata.revision,
                              sha256: metadata.sha256,
                              offset: metadata.next_offset,
                              limit: 65536,
                            }),
                            metadata,
                          ),
                        );
                      })
                    }
                  >
                    Read next capture chunk
                  </Button>
                )}
              </details>
            )}
            {playback && (
              <>
                <p>
                  {playback.preview
                    ? 'Playing the documented decoded preview, linked to the original source.'
                    : 'Playing the identity-verified original.'}
                </p>
                {entry.class === 'video' ? (
                  <video
                    key={playback.url}
                    aria-label="Original video"
                    controls
                    src={playback.url}
                    ref={(node) => {
                      media.current = node;
                    }}
                    onLoadedMetadata={applyPendingSeek}
                    onLoadedData={applyPendingSeek}
                    onCanPlay={applyPendingSeek}
                    onSeeked={confirmPendingSeek}
                    onError={() =>
                      run(async () => {
                        throw new Error(
                          'This codec is unavailable in the native webview. Use an externally decoded preview as described in offline help.',
                        );
                      })
                    }
                  />
                ) : (
                  <audio
                    key={playback.url}
                    aria-label="Original audio"
                    controls
                    src={playback.url}
                    ref={(node) => {
                      media.current = node;
                    }}
                    onLoadedMetadata={applyPendingSeek}
                    onLoadedData={applyPendingSeek}
                    onCanPlay={applyPendingSeek}
                    onSeeked={confirmPendingSeek}
                  />
                )}
              </>
            )}
          </Card>
          <Card heading="Declared speaker roster">
            <p>{roster?.declared ? ((roster.members?.length??0)===0?'Explicitly empty roster':'Declared roster') : 'Roster not declared'}. Revision {roster?.revision??0}.</p>
            <p>Declared speakers are intentional context. Membership does not assign transcript voices or provide acoustic evidence.</p>
            <Table caption="Declared roster members" heads={['Speaker','Identity','State']}>
              {(roster?.members??[]).map((sp:Obj)=><tr key={sp.id}><td>{sp.name}</td><td>{sp.id}</td><td>{sp.state}</td></tr>)}
            </Table>
            <Select label="Roster edit" value={rosterMode} onChange={(v)=>setRosterMode(v as typeof rosterMode)} options={[{value:'add',label:'Add members'},{value:'remove',label:'Remove members'},{value:'replace',label:'Replace membership'}]}/>
            <Area label="Roster speaker references" value={rosterRefs} onChange={setRosterRefs} description="One ID, unique name or active alias per line. Inactive members can be removed by ID."/>
            <Actions>
              <Button onClick={()=>run(()=>mutateRoster(rosterMode))}>Apply roster edit</Button>
              <Button variant="secondary" onClick={()=>run(()=>mutateRoster('clear'))}>Clear roster</Button>
              <Button variant="secondary" onClick={()=>run(async()=>setRoster(await client.roster(entryID)))}>Refresh roster</Button>
            </Actions>
          </Card>
          <RosterMatching key={entryID} client={client} run={run} recordingID={entryID} roster={roster}/>
          <Card heading="Recording date">
            <p>
              Keep unknown dates explicit. Date-only values preserve precision
              without inventing a time.
            </p>
            <Select
              label="Date precision"
              value={precision}
              onChange={setPrecision}
              options={[
                { value: 'date', label: 'Date only' },
                { value: 'time', label: 'Date and time' },
              ]}
            />
            <Input
              label="Origination date"
              value={date}
              onChange={setDate}
              description="YYYY-MM-DD or ISO date and time"
            />
            <Input
              label="Time zone"
              value={zone}
              onChange={setZone}
              description="IANA zone, UTC or numeric offset"
            />
            <Select
              label="Repeated wall time"
              value={fold}
              onChange={setFold}
              options={[
                { value: '', label: 'Leave unresolved' },
                'earlier',
                'later',
              ]}
            />
            <Select
              label="Nonexistent wall time"
              value={gap}
              onChange={setGap}
              options={[
                { value: '', label: 'Leave unresolved' },
                'shift-forward',
              ]}
            />
            <Button
              onClick={() =>
                run(async () => {
                  await client.call('media.set-origin', entryID, {
                    revision: entry.revision,
                    options: dateOptions(),
                  });
                  await refresh();
                })
              }
            >
              Save date correction
            </Button>
          </Card>
          {entry.mode==='reference' && <Card heading="Relocate original">
            <Input
              label="New original path"
              value={relocation}
              onChange={setRelocation}
            />
            <Button
              variant="secondary"
              onClick={() =>
                run(async () => {
                  const path = await client.bridge.Choose('media');
                  if (path) setRelocation(path);
                })
              }
            >
              Choose relocated original
            </Button>
            <Button
              onClick={() =>
                run(async () => {
                  await client.call('media.relocate', entryID, {
                    revision: entry.revision,
                    path: relocation,
                  });
                  await refresh();
                })
              }
            >
              Verify identity and reconnect
            </Button>
          </Card>}
          <Card heading="Processing">
            <Select
              label="Saved pipeline"
              value={pipeline}
              onChange={setPipeline}
              options={[
                { value: '', label: 'Explicit local model selection' },
                ...pipelines.map((p) => ({ value: p.id, label: p.name })),
              ]}
            />
            {pipelineCursor && (
              <Button
                variant="secondary"
                onClick={() =>
                  run(async () => {
                    const p = await client.call('pipelines.list', '', {
                      limit: 100,
                      after_id: pipelineCursor,
                    });
                    setPipelines((old) => [...old, ...p.items]);
                    setPipelineCursor(p.next_id);
                  })
                }
              >
                Load more saved pipelines
              </Button>
            )}
            <Select
              label="Transcription input"
              value={transcription}
              onChange={setTranscription}
              options={[
                {
                  value: '',
                  label: 'Supplied subtitle when present, otherwise generate',
                },
                'supplied',
                'generate',
                'reuse',
              ]}
            />
            <Select
              label="Speaker attribution"
              value={diarization}
              onChange={setDiarization}
              options={['run', 'reuse']}
            />
            {!pipeline && (
              <>
                {(!transcription || transcription === 'generate') && (
                  <ModelChoice
                    label="Local recognition model"
                    operation="transcription"
                    value={recognitionModel}
                    onChange={setRecognitionModel}
                    client={client}
                    run={run}
                  />
                )}
                {diarization === 'run' && (
                  <ModelChoice
                    label="Local diarization model"
                    operation="diarization"
                    value={diarizationModel}
                    onChange={setDiarizationModel}
                    client={client}
                    run={run}
                  />
                )}
                <p>
                  Choose a compatible model reference. The runtime freezes its
                  exact version and acquires missing verified files before processing.
                </p>
              </>
            )}
            <Actions>
              <Button
                onClick={() =>
                  run(async () => {
                    await client.call('recordings.process', entryID, {
                      ...(pipeline
                        ? {
                            pipeline_id: pipeline,
                            pipeline_revision: pipelines.find(
                              (p) => p.id === pipeline,
                            )?.revision,
                          }
                        : {
                            ...(recognitionModel &&
                            (!transcription || transcription === 'generate')
                              ? { recognition_model_id: recognitionModel }
                              : {}),
                            ...(diarizationModel && diarization === 'run'
                              ? { diarization_model_id: diarizationModel }
                              : {}),
                          }),
                      ...(transcription ? { transcription } : {}),
                      diarization,
                    });
                    await refresh();
                  })
                }
              >
                {cues ? 'Rerun processing' : 'Process recording'}
              </Button>
              <Button variant="secondary" onClick={() => run(refresh)}>
                Refresh current results
              </Button>
            </Actions>
            <p>
              Configured stages run without fallback. Processing continues when
              the desktop closes.
            </p>
          </Card>
          <Card heading="Assemble supplied subtitles">
            <p>
              Use supplied subtitles with source-linked speaker turns. Times are
              original-source milliseconds.
            </p>
            <Input
              label="Turn label"
              value={turnLabel}
              onChange={setTurnLabel}
            />
            <Input
              label="Turn start milliseconds"
              type="number"
              value={turnStart}
              onChange={setTurnStart}
            />
            <Input
              label="Turn end milliseconds"
              type="number"
              value={turnEnd}
              onChange={setTurnEnd}
            />
            <Button
              variant="secondary"
              onClick={() => {
                const start = Number(turnStart),
                  end = Number(turnEnd);
                if (
                  Number.isSafeInteger(start * 1000) &&
                  Number.isSafeInteger(end * 1000) &&
                  start >= 0 &&
                  end > start
                )
                  setTurns([
                    ...turns,
                    {
                      label: turnLabel,
                      start_us: start * 1000,
                      end_us: end * 1000,
                    },
                  ]);
                else
                  run(async () => {
                    throw new Error('Enter a valid source interval.');
                  });
              }}
            >
              Add turn
            </Button>
            <Facts value={turns} />
            <Input
              label="Untimed participation cue ID"
              value={participationCue}
              onChange={setParticipationCue}
            />
            <Input
              label="Untimed participation label"
              value={participationLabel}
              onChange={setParticipationLabel}
            />
            <Button
              variant="secondary"
              onClick={() =>
                setParticipation([
                  ...participation,
                  { cue_id: participationCue, label: participationLabel },
                ])
              }
            >
              Add untimed participation
            </Button>
            <Facts value={participation} />
            <Button variant="secondary" onClick={() => setParticipation([])}>
              Clear participation
            </Button>
            <Actions>
              <Button variant="secondary" onClick={() => setTurns([])}>
                Clear turns
              </Button>
              <Button
                onClick={() =>
                  run(async () => {
                    await client.call('recordings.assemble', entryID, {
                      subtitle_format: format,
                      turns,
                      participation,
                    });
                    await refresh();
                  })
                }
              >
                Assemble current recording
              </Button>
            </Actions>
          </Card>
          {cues && (
            <>
              <Card heading="Current transcript">
                <p>
                  Revision {cues.revision}. Selecting timed evidence seeks the
                  original source.
                </p>
                <Table
                  caption="Current source-linked cues"
                  heads={['Cue', 'Text', 'Time', 'Speaker spans']}
                >
                  {cues.cues.map((cue: Obj) => (
                    <tr key={cue.id}>
                      <td>{cue.ordinal + 1}</td>
                      <td>{cue.text}</td>
                      <td>
                        {cue.seek_seconds === null ? (
                          'Untimed'
                        ) : (
                          <Button
                            variant="ghost"
                            onClick={() => run(() => play(cue.seek_seconds))}
                          >
                            Seek {cue.timing?.start_milliseconds} ms
                          </Button>
                        )}
                      </td>
                      <td>
                        {cue.speaker_attributions?.map((a: Obj, i: number) => (
                          <div key={i}>
                            {a.local_speaker_id}
                            {a.seek_seconds !== undefined ? (
                              <Button
                                variant="ghost"
                                onClick={() => run(() => play(a.seek_seconds))}
                              >
                                Seek speaker span
                              </Button>
                            ) : (
                              <span> Untimed participation</span>
                            )}
                          </div>
                        ))}
                      </td>
                    </tr>
                  ))}
                </Table>
                <Pager
                  next={cues.next_ordinal !== null}
                  load={() =>
                    run(async () => {
                      setCues(
                        await client.call('recordings.cues', entryID, {
                          revision: cues.revision,
                          document_digest: cues.document_digest,
                          after_ordinal: cues.next_ordinal,
                          limit: 25,
                        }),
                      );
                    })
                  }
                />
              </Card>
              <Card heading="Known speaker mapping">
                <Select
                  label="Local voice"
                  value={localVoice}
                  onChange={setLocalVoice}
                  options={[
                    { value: '', label: 'Choose local voice' },
                    ...(cues.voices ?? []).map((v: Obj) => ({
                      value: v.id,
                      label: v.name || v.id,
                    })),
                  ]}
                />
                <Select
                  label="Known person"
                  value={person}
                  onChange={setPerson}
                  options={[
                    { value: '', label: 'Choose person' },
                    ...speakers.map((s) => ({ value: s.id, label: s.name })),
                  ]}
                />
                {speakerCursor && (
                  <Button
                    variant="secondary"
                    onClick={() =>
                      run(async () => {
                        const s = await client.call('speakers.list', '', {
                          limit: 100,
                          after_id: speakerCursor,
                        });
                        setSpeakers((old) => [
                          ...old,
                          ...s.items.map((item: Obj) => item.speaker),
                        ]);
                        setSpeakerCursor(s.next_id);
                      })
                    }
                  >
                    Load more known speakers
                  </Button>
                )}
                <Button
                  onClick={() =>
                    run(async () => {
                      await client.call('recordings.map-speaker', entryID, {
                        revision: cues.revision,
                        document_digest: cues.document_digest,
                        mapping_revision:
                          mappings.items.find(
                            (m: Obj) => m.local_speaker_id === localVoice,
                          )?.revision ?? 0,
                        local_speaker_id: localVoice,
                        speaker_id: person,
                      });
                      await refresh();
                    })
                  }
                >
                  Save current mapping
                </Button>
                <Facts value={mappings.items} />
                <Pager
                  next={mappings.next_id}
                  load={() =>
                    run(async () => {
                      const next = await client.call(
                        'recordings.mappings',
                        entryID,
                        {
                          after_id: mappings.next_id,
                          limit: 100,
                        },
                      );
                      setMappings((old: Obj) => ({
                        ...next,
                        items: [
                          ...old.items.filter(
                            (item: Obj) =>
                              !next.items.some(
                                (incoming: Obj) =>
                                  incoming.local_speaker_id ===
                                  item.local_speaker_id,
                              ),
                          ),
                          ...next.items,
                        ],
                      }));
                    })
                  }
                />
              </Card>
              <Card heading="Export current subtitles">
                <Select
                  label="Subtitle format"
                  value={format}
                  onChange={setFormat}
                  options={['srt', 'vtt', 'cueson']}
                />
                <Input
                  label="Export destination"
                  value={destination}
                  onChange={setDestination}
                />
                <Button
                  variant="secondary"
                  onClick={() =>
                    run(async () => {
                      const path = await client.bridge.Choose('output');
                      if (path) setDestination(path);
                    })
                  }
                >
                  Choose export destination
                </Button>
                <Check
                  label="Require lossless export"
                  value={strict}
                  onChange={setStrict}
                />
                <Button
                  onClick={() =>
                    run(async () => {
                      await client.call('recordings.export', entryID, {
                        format,
                        strict,
                        destination,
                        document_digest: cues.document_digest,
                      });
                    })
                  }
                >
                  Export subtitles
                </Button>
                <p>Exports create a new file and report conversion losses.</p>
              </Card>
            </>
          )}
        </section>
      )}
    </>
  );
}
export function Jobs({ client, run }: Props) {
  const { page, load } = useList(client, 'work.list', run);
  const [detail, setDetail] = useState<Obj>(),
    [results, setResults] = useState<Obj>();
  return (
    <Card heading="Jobs">
      <Actions>
        <Button variant="secondary" onClick={() => run(() => load())}>
          Refresh jobs
        </Button>
      </Actions>
      <Table
        caption="Durable work"
        heads={['Operation', 'State', 'Stage', 'Actions']}
      >
        {page.items.map((job: Obj) => (
          <tr key={job.id}>
            <td>{job.kind}</td>
            <td>
              <StatusBadge
                status={
                  job.state === 'failed'
                    ? 'error'
                    : job.state === 'succeeded'
                      ? 'success'
                      : 'neutral'
                }
              >
                {job.state}
              </StatusBadge>
            </td>
            <td>{job.phase}</td>
            <td>
              <Actions>
                <Button
                  variant="ghost"
                  onClick={() =>
                    run(async () => {
                      client.invalidate();
                      setDetail(undefined);
                      setResults(undefined);
                      setDetail(await client.call('work.show', job.id));
                      setResults(
                        await client.call('work.results', job.id, {
                          limit: 25,
                        }),
                      );
                    })
                  }
                >
                  Inspect job
                </Button>
                {['pending', 'running', 'interrupted'].includes(job.state) && (
                  <Button
                    variant="secondary"
                    onClick={() =>
                      run(async () => {
                        await client.call('work.cancel', job.id);
                        await load();
                      })
                    }
                  >
                    Cancel selected job
                  </Button>
                )}
                {['failed', 'cancelled', 'interrupted'].includes(job.state) &&
                  (job.kind === 'recordings.assemble' ? (
                    <p>
                      Open this recording in Library, supply the intended turns
                      and submit assembly again. Assembly recovery requires
                      those inputs again.
                    </p>
                  ) : (
                    <Button
                      onClick={() =>
                        run(async () => {
                          await client.call('work.retry', job.id);
                          await load();
                        })
                      }
                    >
                      Retry selected job
                    </Button>
                  ))}
              </Actions>
            </td>
          </tr>
        ))}
      </Table>
      <Pager next={page.next_id} load={() => run(() => load(page.next_id))} />
      {detail && (
        <>
          <h3>Job details</h3>
          <Facts
            value={{
              id: detail.id,
              state: detail.state,
              phase: detail.phase,
              selected_models: detail.model_selections,
              selected_model_count: detail.model_selection_count,
              acquisition_jobs: detail.acquisition_ids,
              acquisition_job_count: detail.acquisition_count,
              attempt: detail.generation,
              error: detail.error,
              result: results,
            }}
          />
          <Pager
            next={
              results?.next_ordinal !== null &&
              results?.next_ordinal !== undefined
            }
            load={() =>
              run(async () => {
                setResults(
                  await client.call('work.results', detail.id, {
                    after_ordinal: results?.next_ordinal,
                    limit: 25,
                  }),
                );
              })
            }
          />
        </>
      )}
      <p>
        Stage and attempt state come from the runtime. Closing this window
        leaves jobs running. Refresh after interruption to inspect recovery.
      </p>
    </Card>
  );
}
export function Pipelines({ client, run }: Props) {
  const { page, load } = useList(client, 'pipelines.list', run);
  const [current, setCurrent] = useState<Obj>(),
    [values, setValues] = useState<Obj>(pipelineValues()),
    [inspection, setInspection] = useState<Obj>();
  const update = (key: string, value: any) =>
    setValues((v) => ({ ...v, [key]: value }));
  const select = async (id: string) => {
    client.invalidate();
    const p = await client.call('pipelines.show', id);
    setCurrent(p);
    setValues(pipelineValues(p));
    setInspection(undefined);
  };
  return (
    <>
      <Card heading="Pipelines">
        <Button
          variant="secondary"
          onClick={() => {
            setCurrent(undefined);
            setValues(pipelineValues());
            setInspection(undefined);
          }}
        >
          New pipeline
        </Button>
        <Table
          caption="Saved processing routes"
          heads={['Name', 'Preset', 'Revision', 'Actions']}
        >
          {page.items.map((p: Obj) => (
            <tr key={p.id}>
              <td>{p.name}</td>
              <td>{p.preset}</td>
              <td>{p.revision}</td>
              <td>
                <Button variant="ghost" onClick={() => run(() => select(p.id))}>
                  Edit {p.name}
                </Button>
              </td>
            </tr>
          ))}
        </Table>
        <Pager next={page.next_id} load={() => run(() => load(page.next_id))} />
      </Card>
      <Card heading={current ? 'Edit pipeline' : 'Create pipeline'}>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            run(async () => {
              const payload = pipelinePayload(values, current);
              await client.call('pipelines.set', payload.pipeline.id, payload);
              await select(payload.pipeline.id);
              await load();
            });
          }}
        >
          <Input
            label="Pipeline name"
            value={values.name}
            onChange={(v) => update('name', v)}
            required
          />
          <Select
            label="Preset"
            value={values.preset}
            onChange={(v) => {
              setValues((old) => ({
                ...old,
                preset: v,
                ...(v === 'custom'
                  ? {}
                  : {
                      recognition_mode: v === 'local' ? 'local' : 'hosted',
                      diarization_mode: v === 'local' ? 'local' : 'hosted',
                    }),
              }));
            }}
            options={['local', 'connected', 'custom']}
          />
          <Check label="Include transcription and diarization" value={values.processing_enabled} onChange={value=>update('processing_enabled',value)}/>
          {values.processing_enabled&&<>
          {['recognition', 'diarization'].map((kind) => (
            <fieldset key={kind}>
              <legend>
                {kind === 'recognition'
                  ? 'Transcription stage'
                  : 'Diarization stage'}
              </legend>
              <Select
                label={`${kind} route`}
                value={values[`${kind}_mode`]}
                onChange={(v) => update(`${kind}_mode`, v)}
                options={['local', 'hosted']}
              />
              {values[`${kind}_mode`] === 'local' ? (
                <>
                  <ModelChoice
                    label={`${kind} model`}
                    operation={kind==='recognition'?'transcription':'diarization'}
                    value={values[`${kind}_model`]}
                    onChange={(value) => update(`${kind}_model`, value)}
                    client={client}
                    run={run}
                  />
                  <Select
                    label={`${kind} device`}
                    value={values[`${kind}_device`]}
                    onChange={(v) => update(`${kind}_device`, v)}
                    options={['cpu', 'cuda', 'auto']}
                  />
                </>
              ) : (
                <>
                  <Input
                    label={`${kind} endpoint`}
                    value={values[`${kind}_endpoint`]}
                    onChange={(v) => update(`${kind}_endpoint`, v)}
                  />
                  <Input
                    label={`${kind} remote model`}
                    value={values[`${kind}_remote_model`]}
                    onChange={(v) => update(`${kind}_remote_model`, v)}
                  />
                  <Input
                    label={`${kind} credential ID`}
                    value={values[`${kind}_credential`]}
                    onChange={(v) => update(`${kind}_credential`, v)}
                  />
                </>
              )}
              <Input
                label={`${kind} audio byte limit`}
                type="number"
                value={values[`${kind}_audio_limit`]}
                onChange={(v) => update(`${kind}_audio_limit`, v)}
              />
              <Input
                label={`${kind} response byte limit`}
                type="number"
                value={values[`${kind}_response_limit`]}
                onChange={(v) => update(`${kind}_response_limit`, v)}
              />
              <Input
                label={`${kind} timeout milliseconds`}
                type="number"
                value={values[`${kind}_timeout`]}
                onChange={(v) => update(`${kind}_timeout`, v)}
              />
            </fieldset>
          ))}
          <Input
            label="Recognition language"
            value={values.language}
            onChange={(v) => update('language', v)}
          />
          <Area
            label="Additional recognition hints"
            value={values.hints}
            onChange={(v) => update('hints', v)}
            description="One hint per line. Runtime compiles the selected terminology and aliases."
          />
          <Check
            label="Hosted recognition supports hints"
            value={values.hints_supported}
            onChange={(v) => update('hints_supported', v)}
          />
          <Input
            label="Hosted hint byte limit"
            type="number"
            value={values.hint_limit}
            onChange={(v) => update('hint_limit', v)}
          />
          <Input
            label="Minimum speakers"
            type="number"
            value={values.min_speakers}
            onChange={(v) => update('min_speakers', v)}
          />
          <Input
            label="Maximum speakers"
            type="number"
            value={values.max_speakers}
            onChange={(v) => update('max_speakers', v)}
          />
          <Check
            label="Automatic quality diagnostics"
            value={values.quality}
            onChange={(v) => update('quality', v)}
          />
          {['min_speech_coverage', 'max_overlap_fraction', 'short_turn_us'].map(
            (key) => (
              <Input
                key={key}
                label={`Quality ${key.replaceAll('_', ' ')}`}
                type="number"
                value={values[key]}
                onChange={(v) => update(key, v)}
              />
            ),
          )}
          </>}
          <Area label="Saved speaker training configuration" value={values.speaker_training} onChange={value=>update('speaker_training',value)} description="Optional selected adapter, output kind, base model reference and parameters. A training-only profile can omit transcription and diarization."/>
          <Button type="submit">Save pipeline</Button>
        </form>
        {current && (
          <Button
            variant="secondary"
            onClick={() =>
              run(async () => {
                setInspection(
                  await client.call('pipelines.inspect', current.id, {
                    revision: current.revision,
                  }),
                );
              })
            }
          >
            Inspect elected stages and context
          </Button>
        )}
        {inspection && <Facts value={inspection} />}
      </Card>
    </>
  );
}
export function Speakers({ client, run }: Props) {
  const { page, load } = useList(client, 'speakers.list', run);
  const [current, setCurrent] = useState<Obj>(),
    [values, setValues] = useState<Obj>({
      name: '',
      aliases: '',
      language: '',
      scope: '',
      state: 'active',
    }),
    [evidence, setEvidence] = useState<Obj>();
  const [recordingScope, setRecordingScope] = useState('');
  const sourceAudio = useRef<HTMLAudioElement | null>(null);
  useEffect(() => {
    const url = evidence?.playback?.url;
    if (!url) return;
    let active = true,
      checking = false;
    const generation = client.generation;
    const timer = setInterval(async () => {
      if (checking) return;
      checking = true;
      try {
        const response = await client.bridge.VerifyPlayback(url);
        if (!active || generation !== client.generation) return;
        if (response.error) throw new Error(response.error.message);
      } catch {
        if (!active || generation !== client.generation) return;
        sourceAudio.current?.pause();
        setEvidence(undefined);
        run(async () => {
          throw new Error(
            'Speaker playback authority changed or expired. Inspect current evidence and reacquire playback.',
          );
        });
      } finally {
        checking = false;
      }
    }, 5000);
    return () => {
      active = false;
      clearInterval(timer);
      void client.bridge.ClosePlayback(url).catch(() => {});
    };
  }, [evidence?.playback?.url]);
  const update = (key: string, value: string) =>
    setValues((v) => ({ ...v, [key]: value }));
  const select = async (id: string) => {
    client.invalidate();
    const s = await client.call('speakers.show', id);
    setCurrent(s);
    setValues({
      name: s.speaker.name,
      state: s.speaker.state,
      aliases: s.aliases.map((a: Obj) => a.text).join('\n'),
      language: s.aliases[0]?.language ?? '',
      scope: s.aliases[0]?.scope ?? '',
    });
    setEvidence(undefined);
  };
  return (
    <>
      <Card heading="Speakers">
        <Button
          variant="secondary"
          onClick={() => {
            setCurrent(undefined);
            setValues({
              name: '',
              aliases: '',
              language: '',
              scope: '',
              state: 'active',
            });
            setEvidence(undefined);
          }}
        >
          New speaker
        </Button>
        <Table
          caption="Known people"
          heads={['Name', 'State', 'Revision', 'Details']}
        >
          {page.items.map((s: Obj) => (
            <tr key={s.id}>
              <td>{s.name}</td>
              <td>{s.state}</td>
              <td>{s.revision}</td>
              <td>
                <Button variant="ghost" onClick={() => run(() => select(s.id))}>
                  Edit {s.name}
                </Button>
              </td>
            </tr>
          ))}
        </Table>
        <Pager next={page.next_id} load={() => run(() => load(page.next_id))} />
      </Card>
      <Card heading={current ? 'Edit speaker' : 'Create speaker'}>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            run(async () => {
              const payload = speakerPayload(values, current);
              await client.call('speakers.set', payload.speaker.id, payload);
              await select(payload.speaker.id);
              await load();
            });
          }}
        >
          <Input
            label="Speaker name"
            value={values.name}
            onChange={(v) => update('name', v)}
            required
          />
          <Select
            label="Speaker state"
            value={values.state}
            onChange={(v) => update('state', v)}
            options={states}
          />
          <Area
            label="Aliases"
            value={values.aliases}
            onChange={(v) => update('aliases', v)}
            description="One alias per line"
          />
          <Input
            label="Alias language"
            value={values.language}
            onChange={(v) => update('language', v)}
          />
          <Input
            label="Alias scope"
            value={values.scope}
            onChange={(v) => update('scope', v)}
          />
          <Button type="submit">Save speaker and aliases</Button>
        </form>
        {current && (
          <>
            <Input
              label="Evidence recording filter"
              value={recordingScope}
              onChange={setRecordingScope}
              description="Optional recording ID; empty selects the current library-wide corpus"
            />
            <Button
              variant="secondary"
              onClick={() =>
                run(async () => {
                  client.invalidate();
                  setEvidence(undefined);
                  setEvidence(
                    await client.call('speakers.select', current.speaker.id, {
                      limit: 25,
                      ...(recordingScope
                        ? { recording_id: recordingScope }
                        : {}),
                    }),
                  );
                })
              }
            >
              Inspect current speaker evidence
            </Button>
            {evidence && (
              <>
                <Facts
                  value={{
                    diagnostics: evidence.selection.diagnostics,
                    duplicates: evidence.selection.duplicates,
                    partial: evidence.selection.partial,
                  }}
                />
                <Table
                  caption="Current speaker corpus"
                  heads={['Recording', 'Cue', 'Original source span']}
                >
                  {evidence.selection.items.map((item: Obj, index: number) => (
                    <tr key={index}>
                      <td>{item.reference.recording_id}</td>
                      <td>{item.reference.cue_id}</td>
                      <td>
                        {item.untimed && <span>Untimed participation</span>}
                        {item.source_intervals.map(
                          (interval: Obj, i: number) => (
                            <Button
                              key={i}
                              variant="ghost"
                              onClick={() =>
                                run(async () => {
                                  const generation = client.generation;
                                  const start = rationalSeconds(
                                    interval.start_seconds,
                                  );
                                  if (start === null)
                                    throw new Error(
                                      'This interval cannot be played.',
                                    );
                                  const mediaEntry = await client.call(
                                    'media.show',
                                    item.reference.recording_id,
                                  );
                                  const response = await client.bridge.Playback(
                                    item.reference.recording_id,
                                    mediaEntry.revision,
                                    item.reference.recording_revision,
                                    item.reference.document_digest,
                                  );
                                  if (response.error)
                                    throw new Error(response.error.message);
                                  if (generation !== client.generation) {
                                    if (response.result?.url)
                                      await client.bridge.ClosePlayback(
                                        response.result.url,
                                      );
                                    return;
                                  }
                                  setEvidence((old: Obj | undefined) => ({
                                    ...old,
                                    playback: { ...response.result, start },
                                  }));
                                })
                              }
                            >
                              Play {interval.start_seconds}–
                              {interval.end_seconds} seconds
                            </Button>
                          ),
                        )}
                      </td>
                    </tr>
                  ))}
                </Table>
                {evidence.playback && (
                  <audio
                    ref={sourceAudio}
                    aria-label="Speaker source audio"
                    controls
                    src={evidence.playback.url}
                    onLoadedMetadata={(e) => {
                      e.currentTarget.currentTime = Math.max(
                        0,
                        evidence.playback.start -
                          (evidence.playback.timeline_offset_seconds ?? 0),
                      );
                    }}
                  />
                )}
                <Pager
                  next={evidence.next_cursor}
                  load={() =>
                    run(async () => {
                      setEvidence(
                        await client.call(
                          'speakers.select',
                          current.speaker.id,
                          {
                            cursor: evidence.next_cursor,
                            limit: 25,
                            ...(recordingScope
                              ? { recording_id: recordingScope }
                              : {}),
                          },
                        ),
                      );
                    })
                  }
                />
              </>
            )}
          </>
        )}
      </Card>
      {current&&<SpeakerTraining key={current.speaker.id} client={client} run={run} speakerID={current.speaker.id}/>}
    </>
  );
}
export function Terms({ client, run }: Props) {
  const { page, load } = useList(client, 'terms.list', run);
  const [current, setCurrent] = useState<Obj>(),
    [values, setValues] = useState<Obj>({
      canonical: '',
      variants: '',
      language: '',
      context: '',
      speaker_id: '',
      state: 'active',
    }),
    [compiled, setCompiled] = useState<Obj>();
  const [pipeline, setPipeline] = useState('');
  const update = (key: string, value: string) =>
    setValues((v) => ({ ...v, [key]: value }));
  const select = async (id: string) => {
    client.invalidate();
    const t = await client.call('terms.show', id);
    setCurrent(t);
    setValues({
      ...t,
      variants: t.variants.join('\n'),
      speaker_id: t.speaker_id ?? '',
    });
  };
  return (
    <>
      <Card heading="Terms">
        <Button
          variant="secondary"
          onClick={() => {
            setCurrent(undefined);
            setValues({
              canonical: '',
              variants: '',
              language: '',
              context: '',
              speaker_id: '',
              state: 'active',
            });
          }}
        >
          New term
        </Button>
        <Table
          caption="Recognition terminology"
          heads={['Term', 'Language', 'State', 'Details']}
        >
          {page.items.map((t: Obj) => (
            <tr key={t.id}>
              <td>{t.canonical}</td>
              <td>{t.language || 'Any'}</td>
              <td>{t.state}</td>
              <td>
                <Button variant="ghost" onClick={() => run(() => select(t.id))}>
                  Edit {t.canonical}
                </Button>
              </td>
            </tr>
          ))}
        </Table>
        <Pager next={page.next_id} load={() => run(() => load(page.next_id))} />
      </Card>
      <Card heading={current ? 'Edit term' : 'Create term'}>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            run(async () => {
              const payload = termPayload(values, current);
              await client.call('terms.set', payload.term.id, payload);
              await select(payload.term.id);
              await load();
            });
          }}
        >
          <Input
            label="Canonical term"
            value={values.canonical}
            onChange={(v) => update('canonical', v)}
            required
          />
          <Area
            label="Term variants"
            value={values.variants}
            onChange={(v) => update('variants', v)}
            description="One spelling per line"
          />
          <Input
            label="Term language"
            value={values.language}
            onChange={(v) => update('language', v)}
          />
          <Input
            label="Term context"
            value={values.context}
            onChange={(v) => update('context', v)}
          />
          <CatalogChoice
            label="Associated speaker"
            operation="speakers.list"
            value={values.speaker_id}
            onChange={(value) => update('speaker_id', value)}
            client={client}
            run={run}
          />
          <Input
            label="Term speaker ID"
            value={values.speaker_id}
            onChange={(v) => update('speaker_id', v)}
          />
          <Select
            label="Term state"
            value={values.state}
            onChange={(v) => update('state', v)}
            options={states}
          />
          <Button type="submit">Save term</Button>
        </form>
      </Card>
      <Card heading="Recognition context preview">
        <CatalogChoice
          label="Preview pipeline"
          operation="pipelines.list"
          value={pipeline}
          onChange={setPipeline}
          client={client}
          run={run}
          empty="Local recognition default context"
        />
        <Input
          label="Preview pipeline ID"
          value={pipeline}
          onChange={setPipeline}
        />
        <Button
          variant="secondary"
          onClick={() =>
            run(async () => {
              client.invalidate();
              setCompiled(undefined);
              setCompiled(
                await client.call('terms.compile', '', {
                  ...(pipeline ? { pipeline_id: pipeline } : {}),
                  language: values.language,
                  context: values.context,
                  ...(values.speaker_id
                    ? { speaker_ids: [values.speaker_id] }
                    : {}),
                }),
              );
            })
          }
        >
          Preview effective hints
        </Button>
        {compiled && <Facts value={compiled} />}
      </Card>
    </>
  );
}
