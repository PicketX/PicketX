// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict'
import { test } from 'node:test'
import { chmod, lstat, mkdir, mkdtemp, readFile, rm, symlink, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { spawnSync } from 'node:child_process'
import { copyPrivateState, snapshotPrivateTree, relocateInstanceConfig } from './controller-state-copy.mjs'
import * as stateCopy from './controller-state-copy.mjs'

for (const kind of ['binding bytes', 'provider object']) test(`private ${kind} mismatch rejects without disclosing either document`, () => {
  const markers = ['SUBJECT-PRIVATE-A', 'PRINCIPAL-PRIVATE-A', 'GENERATION-PRIVATE-A', 'ISSUER-PRIVATE-A', 'CLIENT-PRIVATE-A', 'SECRET-PRIVATE-A', 'SUBJECT-PRIVATE-B', 'PRINCIPAL-PRIVATE-B', 'GENERATION-PRIVATE-B', 'ISSUER-PRIVATE-B', 'CLIENT-PRIVATE-B', 'SECRET-PRIVATE-B']
  const document = offset => Object.fromEntries(['subject', 'principal', 'generation', 'issuer', 'client_id', 'client_secret'].map((key, i) => [key, markers[offset + i]]))
  const actual = kind === 'binding bytes' ? JSON.stringify(document(0)) : document(0)
  const expected = kind === 'binding bytes' ? JSON.stringify(document(6)) : document(6)
  stateCopy.assertPrivateStateEqual(actual, structuredClone(actual))
  let failure
  try { stateCopy.assertPrivateStateEqual(actual, expected) } catch (error) { failure = error }
  assert(failure instanceof assert.AssertionError, 'unequal private state must reject')
  // The runner prints this stack after literal credential redaction. Neither
  // complete document nor unknown private fields may reach that public boundary.
  for (const marker of markers) assert(!failure.stack.includes(marker), 'private marker reached public failure output')
  assert.equal(failure.message, 'private restored state differs')
})

async function fixture(t) {
  const dir = await mkdtemp(join(tmpdir(), 'picketx-state-test-'))
  t.after(() => rm(dir, { recursive: true, force: true }))
  const source = join(dir, 'source'), target = join(dir, 'backup')
  await mkdir(source, { mode: 0o700 }); await mkdir(join(source, 'state'), { mode: 0o700 })
  await writeFile(join(source, 'key'), 'PRIVATE-FIXTURE\n', { mode: 0o600 })
  await writeFile(join(source, 'state/db'), Buffer.from([0, 1, 2, 255]), { mode: 0o600 })
  return { dir, source, target }
}
test('stopped copy retains complete bytes and private modes without reinit', async t => {
  const { source, target } = await fixture(t)
  const proof = await copyPrivateState(source, target)
  assert.deepEqual(Object.keys(proof), ['key', 'state/db'])
  assert.equal(proof.key, '36857d2cdb8326008a93b7921ea5816062f3afc44714ff7164f0d88e0b7336ab')
  assert.equal((await readFile(join(target, 'key'), 'utf8')), 'PRIVATE-FIXTURE\n')
  assert.deepEqual(await readFile(join(target, 'state/db')), Buffer.from([0, 1, 2, 255]))
  assert.equal((await lstat(target)).mode & 0o777, 0o700)
  assert.equal((await lstat(join(target, 'state/db'))).mode & 0o777, 0o600)
})
test('copy refuses existing target and preserves operator data', async t => {
  const { source, target } = await fixture(t); await writeFile(target, 'OPERATOR')
  await assert.rejects(copyPrivateState(source, target))
  assert.equal(await readFile(target, 'utf8'), 'OPERATOR')
})
test('copy refuses nested target before writing into source', async t => {
  const { source } = await fixture(t)
  await assert.rejects(copyPrivateState(source, join(source, 'backup')))
  assert.equal(await lstat(join(source, 'backup')).catch(() => null), null)
})
test('snapshot rejects symlinked private files', async t => {
  const { source } = await fixture(t); await symlink('key', join(source, 'alias'))
  await assert.rejects(snapshotPrivateTree(source))
})
test('snapshot rejects special private files', async t => {
  const { source } = await fixture(t)
  assert.equal(spawnSync('mkfifo', ['-m', '600', join(source, 'fifo')]).status, 0)
  await assert.rejects(snapshotPrivateTree(source))
})
for (const [entry, mode] of [['key', 0o644], ['state', 0o755], ['', 0o755]]) test('snapshot rejects loose '+(entry || 'root'), async t => {
  const { source } = await fixture(t); await chmod(join(source, entry), mode)
  await assert.rejects(snapshotPrivateTree(source))
})
const config = () => ({ database: '/private/old/state/controller.sqlite', credentials: '/private/old/agents.json', management: { credentials: '/private/old/principals.json', users_mode: 'database' }, authz: { credentials: '/private/old/callers.json' }, agent: { authority: 'authority-stays' }, browser: { assets: '/bundle/web', origin: 'http://127.0.0.1:8080' } })
test('relocation changes only four exact instance paths, retaining authority and source', () => {
  const original = config(), actual = relocateInstanceConfig(original, '/private/old', '/private/new')
  assert.deepEqual(actual, { ...config(), database: '/private/new/state/controller.sqlite', credentials: '/private/new/agents.json', management: { credentials: '/private/new/principals.json', users_mode: 'database' }, authz: { credentials: '/private/new/callers.json' } })
  assert.deepEqual(original, config())
})
test('relocation rejects a path outside original complete instance', () => {
  const c = config(); c.database = '/operator/live.sqlite'
  assert.throws(() => relocateInstanceConfig(c, '/private/old', '/private/new'))
})
test('optional OIDC runtime path follows the complete private instance', () => {
  const original = config(); original.browser.oidc_config = '/private/old/identity/provider.json'
  const actual = relocateInstanceConfig(original, '/private/old', '/private/new')
  assert.equal(actual.browser.oidc_config, '/private/new/identity/provider.json')
  assert.equal(actual.browser.assets, '/bundle/web'); assert.equal(actual.agent.authority, 'authority-stays')
  assert.equal(original.browser.oidc_config, '/private/old/identity/provider.json')
})
test('optional OIDC runtime relocation refuses unknown or external private paths', () => {
  for (const path of ['/operator/identity/provider.json', '/private/old/identity/../provider.json', 'identity/provider.json']) {
    const original = config(); original.browser.oidc_config = path
    assert.throws(() => relocateInstanceConfig(original, '/private/old', '/private/new'))
    assert.equal(original.browser.oidc_config, path)
  }
})
test('nested provider relocation preserves every nonpath identity field', () => {
  const original = { issuer: 'https://identity.example.test', client_id: 'picketx-fixture', client_secret: 'not-a-real-secret', bindings: '/private/old/identity/bindings.json' }
  const actual = stateCopy.relocateOIDCProvider(original, '/private/old', '/private/new')
  assert.deepEqual(actual, { issuer: 'https://identity.example.test', client_id: 'picketx-fixture', client_secret: 'not-a-real-secret', bindings: '/private/new/identity/bindings.json' })
  assert.equal(original.bindings, '/private/old/identity/bindings.json')
})
test('nested provider relocation refuses missing or external binding state', () => {
  for (const bindings of [undefined, '', '/operator/bindings.json', '/private/old/identity/../bindings.json']) {
    assert.throws(() => stateCopy.relocateOIDCProvider({ bindings }, '/private/old', '/private/new'))
  }
})
