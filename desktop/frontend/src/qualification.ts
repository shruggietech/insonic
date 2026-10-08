// SPDX-License-Identifier: Apache-2.0
import { result, type NativeBridge, type Obj } from './client';
// Runs only during the explicit native qualification mode. Every write below
// goes through the same mounted controls and shared operations as ordinary use.
export async function qualifyDesktop(bridge: NativeBridge): Promise<Obj> {
  let stage = 'Load qualification fixtures';
  try {
    return await qualifyJourney(bridge, (value) => {
      stage = value;
    });
  } catch (error) {
    throw new Error(
      `[${stage}] ${error instanceof Error ? error.message : 'Unknown failure.'}`,
    );
  }
}
async function qualifyJourney(
  bridge: NativeBridge,
  stage: (value: string) => void,
): Promise<Obj> {
  if (!bridge.NativeQualificationData)
    throw new Error('Qualification fixtures unavailable.');
  const fixtures = result(await bridge.NativeQualificationData());
  if (!fixtures.audio_path || !fixtures.video_path || !fixtures.subtitle_path)
    throw new Error('Committed media fixtures unavailable.');
  const response = await bridge.Show();
  const workspace = response.workspace_id;
  const call = async (operation: string, item_id = '', data?: Obj) => {
    const response = await bridge.Operate({
      operation,
      workspace_id: workspace,
      ...(item_id ? { item_id } : {}),
      ...(data === undefined ? {} : { data }),
    });
    if (response.error)
      throw new Error(
        `${operation}: ${response.error.code}: ${response.error.message}`,
      );
    return result(response);
  };
  const wait = async (
    condition: () => boolean | Promise<boolean>,
    message: string,
    timeout = 25000,
  ) => {
    const deadline = Date.now() + timeout;
    while (Date.now() < deadline) {
      if (await condition()) return;
      await new Promise((resolve) => setTimeout(resolve, 40));
    }
    throw new Error(message);
  };
  const idle = async () => {
    await new Promise((resolve) => setTimeout(resolve, 50));
    await wait(
      () =>
        !document
          .querySelector('[role=status]')
          ?.textContent?.includes('Working'),
      'UI operation did not finish.',
    );
    const error = document.querySelector('[role=alert]');
    if (error) throw new Error(error.textContent ?? 'UI failure.');
  };
  const button = (text: string) => {
    const b = [...document.querySelectorAll('button')].find(
      (b) => b.textContent === text,
    );
    if (!b) throw new Error(`Missing control: ${text}`);
    return b;
  };
  const click = async (text: string) => {
    stage(`Click ${text}`);
    button(text).click();
    await idle();
  };
  const fill = async (label: string, value: string) => {
    stage(`Fill ${label}`);
    const l = [...document.querySelectorAll('label')].find(
      (l) => l.textContent === label,
    );
    const control = l && document.getElementById(l.htmlFor);
    if (!(
      control instanceof HTMLInputElement ||
      control instanceof HTMLSelectElement ||
      control instanceof HTMLTextAreaElement
    ))
      throw new Error(`Missing field: ${label}`);
    const prototype =
      control instanceof HTMLSelectElement
        ? HTMLSelectElement.prototype
        : control instanceof HTMLTextAreaElement
          ? HTMLTextAreaElement.prototype
          : HTMLInputElement.prototype;
    Object.getOwnPropertyDescriptor(prototype, 'value')!.set!.call(
      control,
      value,
    );
    control.dispatchEvent(new Event('input', { bubbles: true }));
    control.dispatchEvent(new Event('change', { bubbles: true }));
    await new Promise((resolve) => setTimeout(resolve, 20));
  };
  const flags: Obj = {};
  stage('Wait for mounted Library initialization');
  await idle();
  const distinctLabel = [...document.querySelectorAll('label')].find(
    (l) => l.textContent === 'Create a distinct library entry for this source',
  );
  const distinct =
    distinctLabel && document.getElementById(distinctLabel.htmlFor);
  if (distinct instanceof HTMLInputElement && !distinct.checked) {
    distinct.click();
    await idle();
  }
  const copyLabel = [...document.querySelectorAll('label')].find(
    (l) => l.textContent === 'Copy original into workspace storage',
  );
  const copy = copyLabel && document.getElementById(copyLabel.htmlFor);
  if (copy instanceof HTMLInputElement && copy.checked) {
    copy.click();
    await idle();
  }
  for (const [kind, path] of [
    ['audio', fixtures.audio_path],
    ['video', fixtures.video_path],
  ]) {
    await fill('Media path or URL', path);
    await fill('Title', `Desktop qualification ${kind}`);
    await fill(
      'Supplied subtitle path',
      kind === 'audio' ? fixtures.subtitle_path : '',
    );
    await click('Import');
    stage(`Wait for ${kind} import job`);
    await wait(async () => {
      const jobs = await call('work.list', '', { limit: 100 });
      return jobs.items
        .filter((j: Obj) => j.kind === 'media.import')
        .every((j: Obj) => j.state === 'succeeded');
    }, 'Media import fixture did not succeed.');
    await click('Refresh library');
    await click(`Open Desktop qualification ${kind}`);
    await click('Play original');
    stage(`Wait for ${kind} playback metadata`);
    await wait(() => {
      const media = document.querySelector(
        kind === 'video' ? 'video' : 'audio',
      ) as HTMLMediaElement | null;
      return !!media && media.readyState >= 1 && media.duration > 0;
    }, `${kind} playback metadata unavailable.`);
    const element = document.querySelector(
      kind === 'video' ? 'video' : 'audio',
    ) as HTMLMediaElement;
    element.muted = true;
    const before = element.currentTime;
    stage(`Decode and play ${kind}`);
    await element.play();
    await wait(
      () => element.currentTime > before + 0.05,
      `${kind} decoding/playback did not advance.`,
      8000,
    );
    element.pause();
    flags[`ui_${kind}_playback`] = 'passed';
    if (kind === 'audio') {
      await click('Inspect raw metadata and dates');
      if (!document.querySelector('details[open]'))
        throw new Error('Metadata details did not render.');
      await fill('Origination date', '1961-01-20');
      await click('Save date correction');
      await fill('New original path', fixtures.audio_path);
      await click('Verify identity and reconnect');
      flags.ui_metadata_date = 'passed';
      await fill('Turn label', 'Qualification voice');
      await fill('Turn start milliseconds', '2500');
      await fill('Turn end milliseconds', '2600');
      await click('Add turn');
      await click('Assemble current recording');
      if (
        !document.querySelector('table caption') ||
        !document.body.textContent?.includes('Current source-linked cues')
      )
        throw new Error('Current cues unavailable.');
      flags.ui_current_assembly = 'passed';
      const seek = [...document.querySelectorAll('button')].find(
        (b) =>
          b.textContent?.startsWith('Seek ') && b.textContent?.endsWith(' ms'),
      );
      if (!seek) throw new Error('Source cue seek unavailable.');
      const cueSeconds =
        Number(seek.textContent!.match(/Seek (\d+) ms/)![1]) / 1000;
      seek.click();
      stage('Seek current source cue');
      await idle();
      await wait(() => {
        const audio = document.querySelector('audio');
        return (
          !!audio &&
          audio.readyState >= 1 &&
          Math.abs(audio.currentTime - cueSeconds) < 0.02
        );
      }, 'Cue seek did not reach its original-source time.');
      await click('Seek speaker span');
      await wait(() => {
        const audio = document.querySelector('audio');
        return (
          !!audio &&
          audio.readyState >= 1 &&
          Math.abs(audio.currentTime - 2.5) < 0.02
        );
      }, 'Speaker span seek did not reach 2.5 source seconds.');
      flags.ui_cue_seek = 'passed';
    }
  }
  flags.ui_library_import = 'passed';
  await click('Terms');
  await fill('Canonical term', 'Qualification terminology');
  await fill('Term variants', 'Fixture spelling');
  await click('Save term');
  if (
    !document
      .querySelector('table')
      ?.textContent?.includes('Qualification terminology')
  )
    throw new Error('Term did not persist.');
  flags.ui_terms = 'passed';
  await click('Speakers');
  await fill('Speaker name', 'Qualification person');
  await fill('Aliases', 'Fixture alias');
  await click('Save speaker and aliases');
  if (
    !document
      .querySelector('table')
      ?.textContent?.includes('Qualification person')
  )
    throw new Error('Speaker did not persist.');
  stage('Verify persisted speaker identity and aliases');
  const identities = await call('speakers.list', '', { limit: 100 });
  const identity = identities.items.find(
    (item: Obj) => item.speaker.name === 'Qualification person',
  );
  if (!identity?.speaker.id)
    throw new Error('Saved speaker absent from the native identity catalog.');
  const person = await call('speakers.show', identity.speaker.id);
  if (
    person.speaker.id !== identity.speaker.id ||
    person.speaker.name !== 'Qualification person' ||
    !person.aliases.some(
      (alias: Obj) =>
        alias.text === 'Fixture alias' &&
        alias.speaker_id === person.speaker.id &&
        alias.id,
    )
  )
    throw new Error(
      'Current native speaker identity or alias association did not persist.',
    );
  await click('Edit Qualification person');
  const aliasLabel = [...document.querySelectorAll('label')].find(
    (label) => label.textContent === 'Aliases',
  );
  const aliases = aliasLabel && document.getElementById(aliasLabel.htmlFor);
  if (
    !(aliases instanceof HTMLTextAreaElement) ||
    aliases.value !== 'Fixture alias'
  )
    throw new Error('Persisted aliases did not populate the speaker editor.');
  flags.ui_speakers = 'passed';
  await click('Pipelines');
  await fill('Pipeline name', 'Qualification connected route');
  await fill('Preset', 'connected');
  for (const stage of ['recognition', 'diarization']) {
    await fill(`${stage} endpoint`, 'http://127.0.0.1:9/fixture');
    await fill(`${stage} remote model`, 'qualification-no-inference');
  }
  await click('Save pipeline');
  await click('Inspect elected stages and context');
  if (!document.body.textContent?.includes('not-probed'))
    throw new Error('Pipeline inspection missing.');
  flags.ui_pipelines = 'passed';
  await click('Jobs');
  await click('Refresh jobs');
  const inspect = [...document.querySelectorAll('button')].find(
    (b) => b.textContent === 'Inspect job',
  );
  if (!inspect) throw new Error('Durable jobs missing.');
  inspect.click();
  stage('Inspect durable job');
  await idle();
  flags.ui_jobs = 'passed';
  await click('Settings');
  await fill('Theme preference', 'light');
  await click('Save appearance');
  if (!document.documentElement.classList.contains('bb-light'))
    throw new Error('Theme preference not applied.');
  flags.ui_settings = 'passed';
  for (const field of document.querySelectorAll('input,select,textarea'))
    if (
      ![...document.querySelectorAll('label')].some(
        (l) => l.htmlFor === field.id,
      )
    )
      throw new Error('Unlabeled control.');
  button('Library').focus();
  if (
    document.activeElement !== button('Library') ||
    !document.querySelector('a[href="/assets/help/index.html"]')
  )
    throw new Error('Keyboard/help controls missing.');
  flags.ui_keyboard_help = 'passed';
  return flags;
}
