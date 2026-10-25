// SPDX-License-Identifier: Apache-2.0
import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile, readdir } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const json = async file => JSON.parse(await readFile(path.join(root, file), 'utf8'));
const stage = await json('internal/qualificationtiming/policy.json');
const aggregate = await json('scripts/ci-timing-policy.json');
const inventory = await json('specs/047-test-timeout-expansion/deadline-inventory.json');

test('expanded test policy grants useful work plus distinct enclosing cleanup time', () => {
  for (const [key, value] of Object.entries(stage)) {
    assert.match(key, /_seconds$/);
    assert.ok(Number.isSafeInteger(value) && value > 0, key);
  }
  assert.deepEqual(aggregate, {
    qualification_target_seconds: 1800,
    qualification_limit_seconds: 3600,
    finalization_reserve_seconds: 60
  });
  for (const [work, outer, minimum] of [
    ['help_watchdog_seconds', 'help_seconds', 180],
    ['gui_watchdog_seconds', 'gui_seconds', 600],
    ['exit_watchdog_seconds', 'exit_child_seconds', 180]
  ]) {
    assert.ok(stage[work] >= minimum, work);
    assert.ok(stage[outer] >= stage[work] + stage.cleanup_seconds, outer);
  }
  assert.ok(stage.exit_matrix_seconds >= 12 * stage.exit_child_seconds + 4 * stage.cleanup_seconds);
  assert.ok(stage.first_use_seconds >= 5 * stage.first_use_command_seconds + 2 * stage.cleanup_seconds);
  assert.ok(stage.docs_browser_seconds >= stage.docs_renderer_seconds + stage.request_seconds + stage.cleanup_seconds);
  assert.ok(stage.installer_worker_seconds >= stage.installer_ui_seconds + stage.cleanup_seconds);
  assert.ok(stage.installer_outer_seconds >= stage.installer_worker_seconds + 2 * stage.cleanup_seconds);
  assert.ok(stage.artifact_producer_seconds >= aggregate.qualification_limit_seconds, "Producer waits must cover the original permitted phase");
  assert.ok(stage.go_test_seconds >= 1800);
  assert.ok(stage.go_test_process_seconds >= stage.go_test_seconds + stage.build_seconds);
  // Three advisory requests plus fallback and backoff must fit the npm stage.
  assert.ok(stage.dependency_install_seconds >= 6 * stage.request_seconds + 6 + stage.cleanup_seconds);
});

const ignored = new Set(['node_modules', 'build', 'assets', '.next', 'out', 'offline', 'coverage']);
const extensions = new Set(['.py', '.mjs', '.go', '.ts', '.tsx', '.ps1']);
async function timingOwners(directory) {
  const found = [];
  for (const entry of await readdir(path.join(root, directory), { withFileTypes: true })) {
    const relative = directory + '/' + entry.name;
    if (entry.isDirectory()) {
      if (!ignored.has(entry.name)) found.push(...await timingOwners(relative));
    } else if (entry.isFile() && extensions.has(path.extname(entry.name))) {
      const source = await readFile(path.join(root, relative), 'utf8');
      if (/timeout|deadline|WaitForExit|WaitOne|WaitForSingleObject|time\.(?:After|NewTimer|Second|Minute)/i.test(source)) found.push(relative);
    }
  }
  return found;
}

test('every maintained timer-bearing owner has a checked inventory disposition', async () => {
  const known = new Set();
  for (const group of inventory.groups) {
    assert.ok(group.disposition && group.origin && group.parent && group.effective, group.id);
    for (const file of group.files) {
      assert.ok(!known.has(file), 'Duplicate inventory owner ' + file);
      known.add(file);
      await readFile(path.join(root, file), 'utf8');
    }
  }
  const current = new Set((await Promise.all(inventory.scope_roots.map(timingOwners))).flat());
  const missing = [...current].filter(file => !known.has(file));
  assert.deepEqual(missing.sort(), [], 'Unclassified timing owners must be reviewed explicitly');
  for (const exception of inventory.retained_symbols) {
    assert.ok(exception.reason && exception.symbol, exception.file);
    assert.ok(known.has(exception.file), exception.file);
  }
});

test('current governance matches the expanded policy without rewriting historic evidence', async () => {
  for (const file of ['AGENTS.md', 'CONTRIBUTING.md', '.specify/memory/constitution.md', 'docs/v1.0.1/development.md', 'docs/v1.0.1/product.md']) {
    const text = await readFile(path.join(root, file), 'utf8');
    assert.ok(text.includes('3,600'), file);
    assert.doesNotMatch(text, /qualification.{0,30}(?:660|900)-second/i, file);
  }
});
