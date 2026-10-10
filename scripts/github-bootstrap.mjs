import { readFileSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

// Invoked manually after the repository exists and the owner authorizes setup.
export function runGh(args, input) {
  const result = spawnSync('gh', args, {
    cwd: fileURLToPath(new URL('..', import.meta.url)),
    windowsHide: true,
    shell: false,
    encoding: 'utf8',
    timeout: 60000,
    input,
    maxBuffer: 8 * 1024 * 1024,
    env: { ...process.env, GH_PROMPT_DISABLED: '1', GIT_TERMINAL_PROMPT: '0', GH_PAGER: '' }
  });
  if (result.error || result.status !== 0) {
    // Avoid echoing potentially sensitive gh stderr or environment values.
    throw new Error(`GitHub command failed (${args[0]} ${args[1] ?? ''}); check authentication and repository/project permissions.`);
  }
  return result.stdout.trim() ? JSON.parse(result.stdout) : null;
}
export function issueBody(planned, repository) {
  return [
    `<!-- insonic-work-slice: ${planned.slice} -->`,
    planned.acceptance,
    `Requirements: ${planned.requirements}.`,
    `System specification: https://github.com/${repository}/blob/main/docs/v${readFileSync(new URL('../VERSION', import.meta.url), 'utf8').trim()}/roadmap.md`,
    'Create a focused Spec Kit slice before implementation. This issue records intended work, not completed behavior.'
  ].join('\n\n');
}
export function findSliceIssue(issues, planned) {
  const marker = `<!-- insonic-work-slice: ${planned.slice} -->`;
  const matches = issues.filter(issue => issue.body?.includes(marker) || issue.title === planned.title || issue.title?.startsWith(`${planned.slice}:`));
  if (matches.length > 1) throw new Error(`Multiple issues identify ${planned.slice}; resolve that ambiguity before setup.`);
  return matches[0];
}
export function outcomeLabels(planned, manifest) {
  // Historical manifests remain readable; canonical plans must be complete.
  if (!manifest.taxonomy) return ['work-slice', planned.area];
  const selected = [planned.type, planned.priority, planned.effort];
  for (const [index, group] of ['type', 'priority', 'effort'].entries()) {
    if (!manifest.taxonomy[group]?.includes(selected[index])) throw new Error(`Invalid ${group} for ${planned.slice}.`);
  }
  if (!Array.isArray(planned.areas) || !planned.areas.length || new Set(planned.areas).size !== planned.areas.length || planned.areas.some(area => !manifest.taxonomy.area?.includes(area))) throw new Error(`Invalid areas for ${planned.slice}.`);
  const labels = ['work-slice', ...selected.map((value, index) => `${['type', 'priority', 'effort'][index]}: ${value}`), ...planned.areas.map(area => `area: ${area}`)];
  if (labels.some(name => !manifest.labels.some(label => label.name === name))) throw new Error(`Undeclared outcome label for ${planned.slice}.`);
  return labels;
}
function protectionMatches(protection, checks) {
  return Boolean(protection?.required_status_checks?.strict && checks.every(name => protection.required_status_checks.contexts?.includes(name)) && protection.required_pull_request_reviews?.required_approving_review_count === 0 && protection.required_pull_request_reviews?.require_code_owner_reviews === false && protection.required_pull_request_reviews?.require_last_push_approval === false && protection.required_pull_request_reviews?.dismiss_stale_reviews === true && protection.required_conversation_resolution?.enabled === true && protection.required_linear_history?.enabled === true && protection.allow_force_pushes?.enabled === false && protection.allow_deletions?.enabled === false);
}
export function runBootstrap(manifest, { apply = false, command = runGh } = {}) {
  if (!/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/.test(manifest.repository ?? '')) throw new Error('Bootstrap requires an exact owner/repository name.');
  // Validate all planned outcomes before authentication or any external mutation.
  for (const planned of manifest.issues) outcomeLabels(planned, manifest);
  const gh = command;
  const [owner] = manifest.repository.split('/');
  const base = `repos/${manifest.repository}`;
  function api(endpoint, method = 'GET', body) {
    return gh(['api', endpoint, '--method', method, ...(body ? ['--input', '-'] : [])], body ? JSON.stringify(body) : undefined);
  }
  function pages(endpoint) {
    return gh(['api', endpoint, '--paginate', '--slurp']).flat();
  }
  function gql(query, variables) {
    const result = api('graphql', 'POST', { query, variables });
    if (result.errors?.length) throw new Error('GitHub GraphQL returned errors; check project permissions and field configuration.');
    return result;
  }
  function projectConnection(project, key, selection) {
    const nodes = []; const cursors = new Set(); let after = null;
    const query = `query($project: ID!, $after: String) { node(id: $project) { ... on ProjectV2 { ${selection} } } }`;
    do {
      const connection = gql(query, { project: project.id, after }).data?.node?.[key];
      if (!Array.isArray(connection?.nodes) || typeof connection.pageInfo?.hasNextPage !== 'boolean') throw new Error(`Project ${key} inventory could not be confirmed.`);
      nodes.push(...connection.nodes.filter(Boolean));
      if (!connection.pageInfo.hasNextPage) return nodes;
      after = connection.pageInfo.endCursor;
      if (!after || cursors.has(after)) throw new Error(`Project ${key} pagination did not advance.`);
      cursors.add(after);
    } while (true);
  }
  const itemFieldNames = [...manifest.project.text_fields, 'Status'];
  function projectItems(project) {
    const values = itemFieldNames.map((name, index) => `field${index}: fieldValueByName(name: ${JSON.stringify(name)}) { ... on ProjectV2ItemFieldTextValue { text } ... on ProjectV2ItemFieldSingleSelectValue { name } }`).join(' ');
    // Archived work remains tracked work and retains its metadata and status.
    return projectConnection(project, 'items', `items(first: 100, after: $after, archivedStates: [ARCHIVED, NOT_ARCHIVED]) { nodes { id content { ... on Issue { url } } ${values} } pageInfo { hasNextPage endCursor } }`).map(node => ({ id: node.id, url: node.content?.url, values: Object.fromEntries(itemFieldNames.map((name, index) => [name, node[`field${index}`]?.text ?? node[`field${index}`]?.name ?? null])) }));
  }
  function repositoryLinked(project, repository) {
    return projectConnection(project, 'repositories', 'repositories(first: 100, after: $after) { nodes { id } pageInfo { hasNextPage endCursor } }').some(item => item.id === repository.node_id);
  }
  function membership(items, linked) {
    const missingProjectItems = []; const emptyProjectItemFields = [];
    for (const planned of manifest.issues) {
      const issue = findSliceIssue(issues, planned);
      const item = issue && items.find(candidate => candidate.url === (issue.html_url ?? issue.url));
      if (!item) { missingProjectItems.push(planned.slice); continue; }
      const fields = itemFieldNames.filter(name => typeof item.values[name] !== 'string' || !item.values[name].trim());
      if (fields.length) emptyProjectItemFields.push({ slice: planned.slice, fields });
    }
    return { repositoryProjectLinked: linked, missingProjectItems, emptyProjectItemFields };
  }
  function requiredActions(status) {
    const actions = [];
    if (!status.repositoryProjectLinked) actions.push('Link the project to the repository, then rerun --check.');
    if (status.missingProjectItems.length) actions.push('Add the reported untracked slices to the project, then rerun --check.');
    if (status.missingProjectStatuses.length) actions.push('Add the reported missing Status options without replacing current options, then rerun --check.');
    if (status.emptyProjectItemFields.length) actions.push('Complete the reported empty existing item fields manually; existing edits and progress were left unchanged.');
    if (actions.length) status.actionRequired = actions.join(' ');
  }

  const repository = api(base);
  if (repository.full_name !== manifest.repository || repository.default_branch !== 'main') {
    throw new Error('The exact target repository must exist with main as its default branch. This tool never creates or pushes a repository.');
  }
  const labels = pages(`${base}/labels?per_page=100`);
  const milestones = pages(`${base}/milestones?state=all&per_page=100`);
  const projects = gh(['project', 'list', '--owner', owner, '--closed', '--format', 'json', '--limit', '100']).projects;
  if (projects.length >= 100) throw new Error('Project lookup reached its limit; narrow the owner project inventory before setup.');
  const matches = projects.filter(item => item.title === manifest.project.title);
  if (matches.length > 1) throw new Error('Multiple projects have the requested title; resolve that ambiguity before setup.');
  let project = matches[0];
  const issues = pages(`${base}/issues?state=all&per_page=100`).filter(issue => !issue.pull_request);
  const sliceIds = new Set();
  for (const planned of manifest.issues) {
    if (!/^S\d+$/.test(planned.slice ?? '') || sliceIds.has(planned.slice)) throw new Error('Bootstrap slice identifiers must be unique S-prefixed numbers.');
    sliceIds.add(planned.slice); findSliceIssue(issues, planned);
  }
  // Read membership before applying any changes, so established item values
  // never receive fresh-item defaults on repeated setup.
  const initialItems = project ? projectItems(project) : [];
  if (!apply) {
    let protection;
    let privateReporting;
    try { protection = api(`${base}/branches/main/protection`); } catch { protection = null; }
    try { privateReporting = api(`${base}/private-vulnerability-reporting`); } catch { privateReporting = null; }
    const fields = project ? gh(['project', 'field-list', String(project.number), '--owner', owner, '--format', 'json', '--limit', '100']).fields : [];
    const status = {
      repository: repository.full_name,
      settingsMatch: Object.entries(manifest.settings).every(([key, value]) => repository[key] === value),
      missingLabels: manifest.labels.filter(item => !labels.some(label => label.name === item.name)).map(item => item.name),
      missingMilestones: manifest.milestones.filter(item => !milestones.some(m => m.title === item.title)).map(item => item.title),
      project: project?.url ?? null,
      missingProjectFields: manifest.project.text_fields.filter(name => !fields.some(field => field.name === name)),
      missingProjectStatuses: manifest.project.statuses.filter(name => !fields.find(field => field.name === 'Status')?.options?.some(option => option.name === name)),
      protectionConfirmed: protectionMatches(protection, manifest.required_checks),
      privateReportingConfirmed: privateReporting?.enabled === true,
      missingSlices: manifest.issues.filter(item => !findSliceIssue(issues, item)).map(item => item.slice),
      ...membership(initialItems, project ? repositoryLinked(project, repository) : false)
    };
    status.complete = Boolean(status.settingsMatch && !status.missingLabels.length && !status.missingMilestones.length && !status.missingSlices.length && !status.missingProjectFields.length && !status.missingProjectStatuses.length && !status.missingProjectItems.length && !status.emptyProjectItemFields.length && status.repositoryProjectLinked && status.protectionConfirmed && status.privateReportingConfirmed && project);
    requiredActions(status);
    return status;
  }
  api(base, 'PATCH', { description: manifest.description, ...manifest.settings });
  for (const label of manifest.labels) {
    const exists = labels.some(item => item.name === label.name);
    api(exists ? `${base}/labels/${encodeURIComponent(label.name)}` : `${base}/labels`, exists ? 'PATCH' : 'POST', label);
  }
  for (const milestone of manifest.milestones) {
    if (!milestones.some(item => item.title === milestone.title)) {
      milestones.push(api(`${base}/milestones`, 'POST', milestone));
    }
  }
  const createdProject = !project;
  if (createdProject) {
    project = gh(['project', 'create', '--owner', owner, '--title', manifest.project.title, '--format', 'json']);
  }
  gql('mutation($project: ID!, $repository: ID!) { linkProjectV2ToRepository(input: {projectId: $project, repositoryId: $repository}) { repository { id } } }', { project: project.id, repository: repository.node_id });
  const fields = gh(['project', 'field-list', String(project.number), '--owner', owner, '--format', 'json', '--limit', '100']).fields;
  for (const name of manifest.project.text_fields) {
    if (!fields.some(field => field.name === name)) {
      gh(['project', 'field-create', String(project.number), '--owner', owner, '--name', name, '--data-type', 'TEXT', '--format', 'json']);
    }
  }
  const statusField = fields.find(field => field.name === 'Status');
  // Only a new, empty project receives initial Status configuration.
  if (createdProject && statusField && manifest.project.statuses.some(name => !statusField.options?.some(option => option.name === name))) {
    const items = projectItems(project);
    if (items.length) throw new Error('The newly created project is no longer empty; Status options were left unchanged.');
    const existing = gql('query($field: ID!) { node(id: $field) { ... on ProjectV2SingleSelectField { options { id name color description } } } }', { field: statusField.id }).data?.node?.options;
    if (!Array.isArray(existing) || existing.some(option => !option.id || !option.name || !option.color || typeof option.description !== 'string')) throw new Error('Existing Status option identities and metadata could not be read; options were left unchanged.');
    const additions = manifest.project.statuses.filter(name => !existing.some(option => option.name === name));
    const colors = ['GRAY', 'BLUE', 'YELLOW', 'PURPLE', 'GREEN'];
    gql('mutation($input: UpdateProjectV2FieldInput!) { updateProjectV2Field(input: $input) { projectV2Field { ... on ProjectV2SingleSelectField { id } } } }', {
      input: { fieldId: statusField.id, singleSelectOptions: [...existing.map(({ id, name, color, description }) => ({ id, name, color, description })), ...additions.map((name, index) => ({ name, color: colors[index % colors.length], description: `${name} work` }))] }
    });
  }
  const updatedFields = gh(['project', 'field-list', String(project.number), '--owner', owner, '--format', 'json', '--limit', '100']).fields;
  const backlog = updatedFields.find(field => field.name === 'Status')?.options?.find(option => option.name === 'Backlog');
  let createdIssues = 0; let createdProjectItems = 0;
  for (const planned of manifest.issues) {
    let issue = findSliceIssue(issues, planned);
    const createdIssue = !issue;
    if (createdIssue) {
      const body = issueBody(planned, manifest.repository);
      const milestone = milestones.find(item => item.title === planned.milestone);
      if (!milestone) throw new Error(`Missing milestone for ${planned.slice}.`);
      issue = api(`${base}/issues`, 'POST', {
        title: planned.title,
        body,
        labels: outcomeLabels(planned, manifest),
        milestone: milestone.number
      });
      issues.push(issue); createdIssues += 1;
    }
    const existingItem = initialItems.find(candidate => candidate.url === (issue.html_url ?? issue.url));
    const item = existingItem ?? gh(['project', 'item-add', String(project.number), '--owner', owner, '--url', issue.html_url ?? issue.url, '--format', 'json']);
    // An existing issue can receive a newly added project item. Defaults apply
    // to that new item only; every pre-existing item keeps its current values.
    if (!existingItem) {
      createdProjectItems += 1;
      for (const [name, value] of Object.entries({ Slice: planned.slice, Release: planned.milestone, Priority: planned.priority ?? (planned.slice === 'S001' ? 'P1' : 'P2'), Area: planned.areas?.join(', ') ?? planned.area })) {
        const field = updatedFields.find(candidate => candidate.name === name);
        if (field) gh(['project', 'item-edit', '--id', item.id, '--project-id', project.id, '--field-id', field.id, '--text', value, '--format', 'json']);
      }
      if (backlog) {
        gh(['project', 'item-edit', '--id', item.id, '--project-id', project.id, '--field-id', updatedFields.find(field => field.name === 'Status').id, '--single-select-option-id', backlog.id, '--format', 'json']);
      }
    }
  }
  api(`${base}/branches/main/protection`, 'PUT', {
    required_status_checks: { strict: true, contexts: manifest.required_checks },
    enforce_admins: false,
    required_pull_request_reviews: { dismiss_stale_reviews: true, required_approving_review_count: 0, require_code_owner_reviews: false, require_last_push_approval: false },
    restrictions: null,
    required_conversation_resolution: true,
    required_linear_history: true,
    allow_force_pushes: false,
    allow_deletions: false
  });
  api(`${base}/private-vulnerability-reporting`, 'PUT');
  const confirmed = api(base);
  const protection = api(`${base}/branches/main/protection`);
  const privateReporting = api(`${base}/private-vulnerability-reporting`);
  const settingsMatch = Object.entries(manifest.settings).every(([key, value]) => confirmed[key] === value);
  const checksMatch = manifest.required_checks.every(name => protection.required_status_checks?.contexts?.includes(name));
  const protectionConfirmed = protectionMatches(protection, manifest.required_checks);
  const missingProjectFields = manifest.project.text_fields.filter(name => !updatedFields.some(field => field.name === name));
  const missingProjectStatuses = manifest.project.statuses.filter(name => !updatedFields.find(field => field.name === 'Status')?.options?.some(option => option.name === name));
  if (!settingsMatch || !protectionConfirmed || privateReporting?.enabled !== true) throw new Error('Readback did not confirm required repository settings.');
  const status = { repository: confirmed.full_name, project: project.url, settingsMatch, checksMatch, protectionConfirmed, privateReportingConfirmed: true, createdProject, createdIssues, reusedIssues: manifest.issues.length - createdIssues, createdProjectItems, reusedProjectItems: manifest.issues.length - createdProjectItems, missingProjectFields, missingProjectStatuses, ...membership(projectItems(project), repositoryLinked(project, confirmed)) };
  status.complete = Boolean(!missingProjectFields.length && !missingProjectStatuses.length && !status.missingProjectItems.length && !status.emptyProjectItemFields.length && status.repositoryProjectLinked);
  requiredActions(status);
  return status;
}

async function main() {
  if (process.argv.length !== 3 || !['--check', '--apply'].includes(process.argv[2])) {
    console.error('Usage: node scripts/github-bootstrap.mjs --check|--apply'); process.exitCode = 2; return;
  }
  const manifest = JSON.parse(readFileSync(new URL('../.github/bootstrap.json', import.meta.url), 'utf8'));
  const status = runBootstrap(manifest, { apply: process.argv[2] === '--apply' });
  console.log(JSON.stringify(status, null, 2)); process.exitCode = status.complete ? 0 : 1;
}
if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) main().catch(error => { console.error(error.message); process.exitCode = 1; });
