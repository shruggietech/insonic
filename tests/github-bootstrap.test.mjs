// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict';
import test from 'node:test';
import { issueBody, runBootstrap } from '../scripts/github-bootstrap.mjs';

const manifest = {
  repository: 'shruggietech/insonic', description: 'A media orchestrator',
  settings: { has_issues: true, has_projects: true, allow_squash_merge: true, delete_branch_on_merge: true },
  required_checks: ['foundation', 'docs'],
  project: { title: 'insonic', statuses: ['Backlog', 'Ready', 'In progress', 'In review', 'Done'], text_fields: ['Slice', 'Release', 'Priority', 'Area'] },
  labels: [{ name: 'work-slice', color: '7057ff' }, { name: 'platform', color: 'c5def5' }],
  milestones: [{ title: 'M1', description: 'First milestone' }],
  issues: [{ slice: 'S001', title: 'S001: Foundation', milestone: 'M1', area: 'platform', requirements: 'R01 R02', acceptance: 'Prove the platform contracts.' }]
};
const defaults = [{ id: 'opt-todo', name: 'Todo', color: 'GRAY', description: 'Original description' }, { id: 'opt-done', name: 'Done', color: 'GREEN', description: 'Finished work' }];
const copy = value => structuredClone(value);
const arg = (args, name) => args[args.indexOf(name) + 1];

test('bootstrap preserves zero mandatory human approvals and detects an obsolete approval gate', () => {
  const { state, command } = mockGitHub();
  runBootstrap(manifest, { apply: true, command });
  assert.equal(state.protection.required_pull_request_reviews.required_approving_review_count, 0);
  assert.equal(state.protection.required_pull_request_reviews.require_code_owner_reviews, false);
  assert.equal(state.protection.required_pull_request_reviews.require_last_push_approval, false);
  assert.equal(runBootstrap(manifest, { command }).protectionConfirmed, true);
  state.protection.required_pull_request_reviews.required_approving_review_count = 1;
  assert.equal(runBootstrap(manifest, { command }).protectionConfirmed, false);
});

// This in-memory command boundary never launches gh or contacts a service.
function mockGitHub({ existingProject = false, occupiedOnCreate = false } = {}) {
  const project = { id: 'project-1', number: 1, title: 'insonic', url: 'https://github.com/orgs/shruggietech/projects/1' };
  const state = {
    repository: { full_name: manifest.repository, default_branch: 'main', node_id: 'repo-1', ...manifest.settings },
    labels: [], milestones: [], issues: [], projects: existingProject ? [project] : [],
    fields: [{ id: 'status-1', name: 'Status', options: copy(defaults) }],
    items: new Map(), linkedRepositories: [], calls: [], privateReporting: { enabled: false }, protection: null
  };
  const command = (args, input) => {
    const body = input ? JSON.parse(input) : undefined;
    state.calls.push({ args: [...args], body: copy(body) });
    if (args[0] === 'api') {
      const endpoint = args[1]; const method = args.includes('--method') ? arg(args, '--method') : 'GET';
      if (args.includes('--paginate')) {
        if (endpoint.includes('/labels?')) return [copy(state.labels)];
        if (endpoint.includes('/milestones?')) return [copy(state.milestones)];
        if (endpoint.includes('/issues?')) return [copy(state.issues)];
      }
      if (endpoint === 'graphql') {
        if (body.query.includes('repositories(first')) {
          const offset = Number(body.variables.after ?? 0); const next = offset + 100;
          return { data: { node: { repositories: { nodes: copy(state.linkedRepositories.slice(offset, next)), pageInfo: { hasNextPage: next < state.linkedRepositories.length, endCursor: String(next) } } } } };
        }
        if (body.query.includes('items(first')) {
          assert.ok(body.query.includes('archivedStates: [ARCHIVED, NOT_ARCHIVED]'));
          const nodes = [...state.items.entries()].map(([url, item]) => {
            const node = { id: item.id, content: { url } };
            for (const [index, name] of [...manifest.project.text_fields, 'Status'].entries()) {
              const field = state.fields.find(candidate => candidate.name === name); const value = item.values?.[field?.id];
              node[`field${index}`] = value === undefined ? null : name === 'Status' ? { name: state.fields[0].options.find(option => option.id === value)?.name ?? null } : { text: value };
            }
            return node;
          });
          const offset = Number(body.variables.after ?? 0); const next = offset + 100;
          return { data: { node: { items: { nodes: nodes.slice(offset, next), pageInfo: { hasNextPage: next < nodes.length, endCursor: String(next) } } } } };
        }
        if (body.query.startsWith('query')) return { data: { node: { options: copy(state.fields[0].options) } } };
        if (body.query.includes('updateProjectV2Field')) {
          state.fields[0].options = body.variables.input.singleSelectOptions.map((option, index) => ({ ...option, id: option.id ?? `new-option-${index}` }));
          return { data: { updateProjectV2Field: { projectV2Field: { id: 'status-1' } } } };
        }
        if (body.query.includes('linkProjectV2ToRepository')) {
          if (!state.linkedRepositories.some(repository => repository.id === 'repo-1')) state.linkedRepositories.push({ id: 'repo-1' });
          return { data: { linkProjectV2ToRepository: { repository: { id: 'repo-1' } } } };
        }
      }
      if (endpoint === `repos/${manifest.repository}`) {
        if (method === 'PATCH') Object.assign(state.repository, body);
        return copy(state.repository);
      }
      if (endpoint.endsWith('/labels') && method === 'POST') { state.labels.push(body); return copy(body); }
      if (endpoint.includes('/labels/') && method === 'PATCH') { Object.assign(state.labels.find(label => label.name === body.name), body); return copy(body); }
      if (endpoint.endsWith('/milestones') && method === 'POST') { const milestone = { ...body, number: state.milestones.length + 1 }; state.milestones.push(milestone); return copy(milestone); }
      if (endpoint.endsWith('/issues') && method === 'POST') {
        const issue = { ...body, number: state.issues.length + 1, url: `https://api.github.com/repos/${manifest.repository}/issues/1`, html_url: `https://github.com/${manifest.repository}/issues/1`, state: 'open' };
        state.issues.push(issue); return copy(issue);
      }
      if (endpoint.endsWith('/branches/main/protection')) {
        if (method === 'PUT') state.protection = { ...body, required_conversation_resolution: { enabled: body.required_conversation_resolution }, required_linear_history: { enabled: body.required_linear_history }, allow_force_pushes: { enabled: body.allow_force_pushes }, allow_deletions: { enabled: body.allow_deletions } };
        if (!state.protection) throw new Error('No branch protection');
        return copy(state.protection);
      }
      if (endpoint.endsWith('/private-vulnerability-reporting')) {
        if (method === 'PUT') state.privateReporting.enabled = true;
        return copy(state.privateReporting);
      }
    }
    if (args[0] === 'project') {
      if (args[1] === 'list') return { projects: copy(state.projects.filter(candidate => args.includes('--closed') || !candidate.closed)) };
      if (args[1] === 'create') { state.projects.push(project); if (occupiedOnCreate) state.items.set('existing', { id: 'occupied' }); return copy(project); }
      if (args[1] === 'field-list') return { fields: copy(state.fields) };
      if (args[1] === 'field-create') { const field = { id: `field-${arg(args, '--name')}`, name: arg(args, '--name') }; state.fields.push(field); return copy(field); }
      if (args[1] === 'item-list') return { items: [...state.items.values()].map(copy) };
      if (args[1] === 'item-add') {
        const url = arg(args, '--url');
        if (!state.items.has(url)) state.items.set(url, { id: `item-${state.items.size + 1}`, values: {} });
        return copy(state.items.get(url));
      }
      if (args[1] === 'item-edit') {
        assert.equal(arg(args, '--format'), 'json');
        const item = [...state.items.values()].find(value => value.id === arg(args, '--id'));
        item.values[arg(args, '--field-id')] = args.includes('--text') ? arg(args, '--text') : arg(args, '--single-select-option-id');
        return copy(item);
      }
    }
    throw new Error(`Unhandled mock command: ${args.join(' ')}`);
  };
  return { state, command };
}
const optionWrites = state => state.calls.filter(call => call.body?.query?.includes('updateProjectV2Field'));
const itemEdits = state => state.calls.filter(call => call.args[0] === 'project' && call.args[1] === 'item-edit');

test('issue bodies contain real paragraphs and a stable slice identity', () => {
  const body = issueBody(manifest.issues[0], manifest.repository);
  assert.ok(body.startsWith('<!-- insonic-work-slice: S001 -->\n\n'));
  assert.ok(body.includes('\n\nRequirements: R01 R02.\n\n'));
  assert.ok(!body.includes('\\n'));
});

test('fresh apply uses public issue URLs and preserves initial option identities and metadata', () => {
  const { state, command } = mockGitHub();
  const result = runBootstrap(manifest, { apply: true, command });
  assert.equal(result.complete, true); assert.equal(result.createdIssues, 1); assert.equal(result.createdProjectItems, 1); assert.equal(result.repositoryProjectLinked, true);
  const add = state.calls.find(call => call.args[1] === 'item-add');
  assert.equal(arg(add.args, '--url'), 'https://github.com/shruggietech/insonic/issues/1');
  assert.deepEqual(optionWrites(state)[0].body.variables.input.singleSelectOptions.slice(0, 2), defaults);
  assert.equal(itemEdits(state).length, 5);
  assert.ok(state.fields[0].options.some(option => option.name === 'Backlog'));
});

test('repeated apply preserves edited or closed issues and every existing item field', () => {
  const { state, command } = mockGitHub(); runBootstrap(manifest, { apply: true, command });
  Object.assign(state.issues[0], { title: 'Owner revised title', body: '<!-- insonic-work-slice: S001 -->\n\nOwner revised acceptance.', state: 'closed', labels: ['owner-label'], milestone: 99 });
  const existingItem = [...state.items.values()][0]; Object.assign(existingItem.values, { 'field-Priority': 'P0', 'status-1': 'opt-done', 'field-Area': 'owner-area' });
  const savedIssue = copy(state.issues[0]); const savedValues = copy(existingItem.values); state.calls = [];
  const result = runBootstrap(manifest, { apply: true, command });
  assert.equal(result.complete, true); assert.equal(result.createdIssues, 0); assert.equal(result.reusedIssues, 1);
  assert.deepEqual(state.issues[0], savedIssue); assert.deepEqual(existingItem.values, savedValues);
  assert.equal(itemEdits(state).length, 0); assert.equal(optionWrites(state).length, 0);
  assert.ok(!state.calls.some(call => call.args[1].endsWith('/issues') && call.args.includes('POST')));
});

test('closed projects are reused without reopening or changing established progress', () => {
  const { state, command } = mockGitHub(); runBootstrap(manifest, { apply: true, command });
  state.projects[0].closed = true;
  const existingItem = [...state.items.values()][0];
  existingItem.values['status-1'] = 'opt-done';
  const savedProject = copy(state.projects[0]); const savedValues = copy(existingItem.values); state.calls = [];
  assert.equal(runBootstrap(manifest, { command }).complete, true);
  const result = runBootstrap(manifest, { apply: true, command });
  assert.equal(result.complete, true); assert.equal(result.createdProject, false);
  assert.equal(result.createdIssues, 0); assert.equal(result.createdProjectItems, 0);
  assert.equal(state.projects.length, 1); assert.deepEqual(state.projects[0], savedProject);
  assert.deepEqual(existingItem.values, savedValues); assert.equal(itemEdits(state).length, 0);
  assert.equal(optionWrites(state).length, 0);
  assert.ok(state.calls.filter(call => call.args[0] === 'project' && call.args[1] === 'list').every(call => call.args.includes('--closed')));
  assert.ok(!state.calls.some(call => call.args[0] === 'project' && ['create', 'edit', 'close'].includes(call.args[1])));
});

test('existing project missing desired statuses reports incomplete without replacing options', () => {
  const { state, command } = mockGitHub({ existingProject: true });
  const result = runBootstrap(manifest, { apply: true, command });
  assert.equal(result.complete, false); assert.deepEqual(result.missingProjectStatuses, ['Backlog', 'Ready', 'In progress', 'In review']);
  assert.match(result.actionRequired, /without replacing/);
  assert.deepEqual(state.fields[0].options, defaults); assert.equal(optionWrites(state).length, 0);
});

test('check reads state without any REST write, project edit or GraphQL mutation', () => {
  const { state, command } = mockGitHub(); runBootstrap(manifest, { apply: true, command }); state.calls = [];
  assert.equal(runBootstrap(manifest, { command }).complete, true);
  for (const call of state.calls) {
    if (call.args[0] === 'api') assert.ok(call.body?.query?.startsWith('query') || !call.args.includes('--method') || arg(call.args, '--method') === 'GET');
    else assert.ok(['list', 'field-list'].includes(call.args[1]));
  }
});

test('ambiguous existing project or slice identities fail before mutations', () => {
  const projects = mockGitHub({ existingProject: true }); projects.state.projects.push(copy(projects.state.projects[0]));
  assert.throws(() => runBootstrap(manifest, { apply: true, command: projects.command }), /Multiple projects/);
  const issues = mockGitHub(); issues.state.issues.push({ title: 'S001: Old title' }, { title: 'S001: Another title' });
  assert.throws(() => runBootstrap(manifest, { apply: true, command: issues.command }), /Multiple issues/);
  for (const mock of [projects, issues]) assert.ok(mock.state.calls.every(call => call.args[0] === 'api' ? !call.args.includes('--method') || arg(call.args, '--method') === 'GET' : call.args[1] === 'list'));
});

test('new project receiving concurrent items retains its original statuses', () => {
  const { state, command } = mockGitHub({ occupiedOnCreate: true });
  assert.throws(() => runBootstrap(manifest, { apply: true, command }), /no longer empty/);
  assert.equal(optionWrites(state).length, 0); assert.deepEqual(state.fields[0].options, defaults);
});

test('existing issue newly added to the project receives initial item metadata only', () => {
  const { state, command } = mockGitHub(); runBootstrap(manifest, { apply: true, command });
  Object.assign(state.issues[0], { title: 'S001: Manually revised work', body: 'Owner acceptance', labels: ['owner-label'], milestone: 77, state: 'closed' });
  const savedIssue = copy(state.issues[0]); state.items.clear(); state.calls = [];
  const result = runBootstrap(manifest, { apply: true, command });
  assert.equal(result.complete, true); assert.equal(result.createdIssues, 0); assert.equal(result.createdProjectItems, 1);
  assert.deepEqual(state.issues[0], savedIssue); assert.equal(itemEdits(state).length, 5);
  assert.equal([...state.items.values()][0].values['field-Slice'], 'S001');
});

test('check reports existing issues absent from the project instead of claiming completion', () => {
  const { state, command } = mockGitHub(); runBootstrap(manifest, { apply: true, command }); state.items.clear(); state.calls = [];
  const result = runBootstrap(manifest, { command });
  assert.equal(result.complete, false); assert.deepEqual(result.missingSlices, []); assert.deepEqual(result.missingProjectItems, ['S001']);
  assert.equal(result.repositoryProjectLinked, true); assert.equal(itemEdits(state).length, 0);
});

test('empty fields on established items are reported and never overwrite existing progress', () => {
  const { state, command } = mockGitHub(); runBootstrap(manifest, { apply: true, command });
  const item = [...state.items.values()][0]; delete item.values['field-Priority']; item.values['status-1'] = 'opt-done';
  const saved = copy(item.values); state.calls = [];
  const result = runBootstrap(manifest, { apply: true, command });
  assert.equal(result.complete, false); assert.deepEqual(result.emptyProjectItemFields, [{ slice: 'S001', fields: ['Priority'] }]);
  assert.match(result.actionRequired, /manually/); assert.deepEqual(item.values, saved); assert.equal(itemEdits(state).length, 0);
});

test('check confirms actual repository-project linkage', () => {
  const { state, command } = mockGitHub(); runBootstrap(manifest, { apply: true, command }); state.linkedRepositories = []; state.calls = [];
  const result = runBootstrap(manifest, { command });
  assert.equal(result.complete, false); assert.equal(result.repositoryProjectLinked, false);
  assert.match(result.actionRequired, /Link the project/);
  assert.ok(!state.calls.some(call => call.body?.query?.startsWith('mutation')));
});

test('item and linked-repository inventories paginate beyond the first hundred entries', () => {
  const { state, command } = mockGitHub(); runBootstrap(manifest, { apply: true, command });
  const [url, item] = [...state.items.entries()][0]; state.items.clear();
  for (let index = 0; index < 120; index += 1) state.items.set(`https://github.com/another/repo/issues/${index + 1}`, { id: `unrelated-${index}`, values: {} });
  state.items.set(url, item); state.linkedRepositories = Array.from({ length: 120 }, (_, index) => ({ id: `other-repository-${index}` })); state.linkedRepositories.push({ id: 'repo-1' }); state.calls = [];
  const result = runBootstrap(manifest, { apply: true, command });
  assert.equal(result.complete, true); assert.equal(result.createdProjectItems, 0); assert.equal(itemEdits(state).length, 0);
  assert.ok(state.calls.some(call => call.body?.query?.includes('items(first') && call.body.variables.after === '100'));
  assert.ok(state.calls.some(call => call.body?.query?.includes('repositories(first') && call.body.variables.after === '100'));
});
