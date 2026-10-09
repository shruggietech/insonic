// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict';
import { cpSync, mkdtempSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import test from 'node:test';
import { validateOwnedVersions } from '../scripts/check-schemas.mjs';
const source = new URL('../', import.meta.url);

test('release version checks include desktop locks and the shared runtime binding', () => {
  const temporary = mkdtempSync(join(tmpdir(), 'insonic-version-'));
  try {
    for (const filename of ['VERSION', 'docs/versions.json', 'internal/contracts/contracts.go', ...['', 'site/', 'desktop/frontend/'].flatMap(prefix => [`${prefix}package.json`, `${prefix}package-lock.json`])]) {
      const target = join(temporary, filename); mkdirSync(dirname(target), { recursive: true }); cpSync(new URL(filename, source), target);
    }
    const version = readFileSync(new URL('VERSION', source), 'utf8').trim(); assert.equal(validateOwnedVersions(temporary), version);
    const filename = join(temporary, 'desktop/frontend/package-lock.json'); const lock = JSON.parse(readFileSync(filename, 'utf8'));
    lock.packages[''].version = '999.999.999'; writeFileSync(filename, JSON.stringify(lock));
    assert.throws(() => validateOwnedVersions(temporary), /package\/lock/);
    lock.packages[''].version = version; writeFileSync(filename, JSON.stringify(lock));
    writeFileSync(join(temporary, 'internal/contracts/contracts.go'), 'const Version = "999.999.999"\n');
    assert.throws(() => validateOwnedVersions(temporary), /Runtime contract/);
  } finally { rmSync(temporary, { recursive: true, force: true }); }
});
