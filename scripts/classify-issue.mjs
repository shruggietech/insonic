// SPDX-License-Identifier: Apache-2.0
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { runGh } from './github-bootstrap.mjs';

export function classifyNewIssue(event, manifest, command = runGh) {
  if (event.action !== 'opened' || event.repository?.full_name !== manifest.repository || event.issue?.pull_request || !Number.isSafeInteger(event.issue?.number) || event.issue.number < 1) return { applied: false, reason: 'Not a new issue in the configured repository.' };
  const labels = (event.issue.labels ?? []).map(label => typeof label === 'string' ? label : label.name);
  if (['type', 'priority', 'effort'].some(group => {
    const selected = labels.filter(name => typeof name === 'string' && name.startsWith(`${group}: `));
    return selected.length !== 1 || !manifest.taxonomy[group].some(value => selected[0] === `${group}: ${value}`);
  })) return { applied: false, reason: 'Canonical outcome classifications require triage.' };
  const section = event.issue.body?.match(/^### Affected areas\s*\n+([^]*?)(?=\n### |$)/m)?.[1]?.trim();
  if (!section) return { applied: false, reason: 'No structured affected areas.' };
  const selected = [...new Set(section.split(',').map(value => value.trim()))];
  if (!selected.length || selected.some(area => !manifest.taxonomy.area.includes(area))) return { applied: false, reason: 'Unknown affected area; labels were preserved.' };
  const additions = selected.map(area => `area: ${area}`).filter(name => !labels.includes(name));
  if (additions.length) command(['api', `repos/${manifest.repository}/issues/${event.issue.number}/labels`, '--method', 'POST', '--input', '-'], JSON.stringify({ labels: additions }));
  return { applied: true, added: additions };
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    const event = JSON.parse(readFileSync(process.env.GITHUB_EVENT_PATH, 'utf8'));
    const manifest = JSON.parse(readFileSync(new URL('../.github/bootstrap.json', import.meta.url), 'utf8'));
    console.log(JSON.stringify(classifyNewIssue(event, manifest)));
  } catch { console.error('Issue classification could not be completed; existing labels were preserved.'); process.exitCode = 1; }
}
