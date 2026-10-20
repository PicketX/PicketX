// SPDX-License-Identifier: Apache-2.0
// Test/deployment verification utility, not an online backup API. Caller must
// prove the service is stopped and own the parent directory against mutation.
import assert from 'node:assert/strict'
import { constants } from 'node:fs'
import { chmod, copyFile, lstat, mkdir, readFile, readdir, realpath } from 'node:fs/promises'
import { createHash } from 'node:crypto'
import { dirname, join, resolve, sep } from 'node:path'
import { isDeepStrictEqual } from 'node:util'

// Failure stacks are public CI output: attach only the comparison result, never
// actual/expected private documents or an automatically generated value diff.
export function assertPrivateStateEqual(actual, expected) {
  assert(isDeepStrictEqual(actual, expected), 'private restored state differs')
}

async function canonical(path) {
  assert.equal(path, resolve(path), 'canonical absolute path required')
  assert.equal(await realpath(path), path, 'symlinked path refused')
}
export async function snapshotPrivateTree(root) {
  await canonical(root)
  const entries = []
  async function walk(relative) {
    const path = relative ? join(root, relative) : root, info = await lstat(path)
    if (info.isDirectory()) {
      assert.equal(info.mode & 0o777, 0o700, 'private directory mode required')
      for (const name of (await readdir(path)).sort()) await walk(relative ? join(relative, name) : name)
    } else {
      assert(info.isFile(), 'only regular private files allowed')
      assert.equal(info.mode & 0o777, 0o600, 'private file mode required')
      entries.push([relative, createHash('sha256').update(await readFile(path)).digest('hex')])
    }
  }
  await walk('')
  return Object.fromEntries(entries.sort(([a], [b]) => a.localeCompare(b)))
}
export async function copyPrivateState(source, target) {
  await canonical(source)
  assert.equal(target, resolve(target), 'canonical absolute target required')
  await canonical(dirname(target))
  assert(!target.startsWith(source + sep), 'backup must be outside source')
  assert.equal(await lstat(target).then(() => true, e => { if (e.code !== 'ENOENT') throw e; return false }), false, 'existing target refused')
  const before = await snapshotPrivateTree(source)
  await mkdir(target, { mode: 0o700 }) // Exclusive; leave partial target for inspection on failure.
  for (const relative of Object.keys(before)) {
    const destination = join(target, relative)
    await mkdir(dirname(destination), { recursive: true, mode: 0o700 })
    await copyFile(join(source, relative), destination, constants.COPYFILE_EXCL)
    await chmod(destination, 0o600)
  }
  assert.deepEqual(await snapshotPrivateTree(source), before, 'source changed during stopped copy')
  assert.deepEqual(await snapshotPrivateTree(target), before, 'backup bytes differ')
  return before
}
export function relocateInstanceConfig(config, old, target) {
  assert.equal(old, resolve(old)); assert.equal(target, resolve(target)); assert.notEqual(old, target)
  const output = structuredClone(config)
  for (const [parent, key, relative] of [[output, 'database', 'state/controller.sqlite'], [output, 'credentials', 'agents.json'], [output.management, 'credentials', 'principals.json'], [output.authz, 'credentials', 'callers.json']]) {
    assert.equal(parent?.[key], join(old, relative), 'unexpected private instance path')
    parent[key] = join(target, relative)
  }
  if (output.browser?.oidc_config !== undefined && output.browser.oidc_config !== '') {
    assert.equal(output.browser.oidc_config, join(old, 'identity/provider.json'), 'unexpected private OIDC provider path')
    output.browser.oidc_config = join(target, 'identity/provider.json')
  }
  return output
}

// Explicit second layer: relocating controller.yaml alone leaves a stale nested
// binding reference. Only this fixture's complete in-instance state is supported.
export function relocateOIDCProvider(provider, old, target) {
  assert.equal(old, resolve(old)); assert.equal(target, resolve(target)); assert.notEqual(old, target)
  assert.equal(provider.bindings, join(old, 'identity/bindings.json'), 'unexpected private OIDC bindings path')
  return { ...structuredClone(provider), bindings: join(target, 'identity/bindings.json') }
}
