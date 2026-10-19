// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict'
import { test } from 'node:test'
import { mkdir, mkdtemp, writeFile, readFile, copyFile, symlink, rm, stat } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'
import { spawnSync } from 'node:child_process'

// Catch a build that truncates an existing installation before preflight checks.
test('bundle rejects existing directory and symlink without modifying them', async () => {
  const parent = await mkdtemp(join(tmpdir(), 'picketx-bundle-test-'))
  try {
    const sentinel = join(parent, 'keep'), link = join(parent, 'link')
    await writeFile(sentinel, 'existing installation')
    await symlink(parent, link)
    for (const output of [parent, link, sentinel, 'relative-output']) {
      const result = spawnSync('bash', [resolve('scripts/build-controller-bundle.sh'), output], { encoding: 'utf8' })
      assert.equal(result.status, 2, 'invalid target must fail at preflight, not build')
      assert.equal(await readFile(sentinel, 'utf8'), 'existing installation')
    }
  } finally { await rm(parent, { recursive: true, force: true }) }
})

test('untracked source cannot be attributed to the committed tree', async () => {
  const parent = await mkdtemp(join(tmpdir(), 'picketx-bundle-source-'))
  try {
    await mkdir(join(parent, 'scripts'))
    const script = join(parent, 'scripts/build-controller-bundle.sh')
    await copyFile(resolve('scripts/build-controller-bundle.sh'), script)
    for (const args of [['init', '-q'], ['add', '.'], ['-c', 'user.name=Test', '-c', 'user.email=test@example.invalid', 'commit', '-qm', 'fixture']]) {
      assert.equal(spawnSync('git', args, { cwd: parent }).status, 0)
    }
    await writeFile(join(parent, 'uncommitted.go'), 'package uncommitted\n')
    const output = join(parent, 'bundle')
    const result = spawnSync('bash', [script, output], { encoding: 'utf8' })
    assert.equal(result.status, 2, 'must reject untracked source before invoking build tools')
    assert.equal(await stat(output).catch(() => null), null, 'must reject before output creation')
  } finally { await rm(parent, { recursive: true, force: true }) }
})
