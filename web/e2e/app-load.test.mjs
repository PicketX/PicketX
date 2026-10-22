// SPDX-License-Identifier: Apache-2.0
// Execute the actual parent load function without mounting unrelated editors.
// Actual browser integration remains a separate mandatory gate.
import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { stripTypeScriptTypes } from 'node:module'

const source = await readFile(new URL('../src/App.tsx', import.meta.url), 'utf8')
const start = source.indexOf('  async function load('), end = source.indexOf('  const bootstrap =', start)
assert(start > 0 && end > start, 'actual parent load boundary')
const body = `return (${stripTypeScriptTypes(source.slice(start, end).trim())})`
function harness(operations, selected = 'audit', failPublication = false) {
  const state = { tab: selected, identity: null, audit: ['old-private-record'] }, epoch = { current: 0 }
  const setters = ['Identity', 'Accepted', 'Draft', 'Audit', 'Cursor', 'HasMore', 'Tab', 'LastRead', 'ReadFailed', 'Reason', 'Conflict']
  const request = async path => {
    if (path === '/me') return { data: { id: 'operator', operations, roles: [] } }
    if (path === '/publication') { if (failPublication) throw new Error('dependent unavailable'); return { data: {}, etag: '"1"' } }
    if (path === '/audit?limit=50') return { data: { events: [], next_after: 0 } }
    throw new Error('unexpected request')
  }
  const values = setters.map(name => value => { const key = name[0].toLowerCase() + name.slice(1); state[key] = typeof value === 'function' ? value(state[key]) : value })
  return { state, load: new Function('request', 'epoch', ...setters.map(x => 'set' + x), body)(request, epoch, ...values) }
}
test('observed audit permission removal applies before a dependent publication failure', async () => {
  const { state, load } = harness(['publication:read'], 'audit', true)
  await assert.rejects(load(), /dependent unavailable/)
  assert.deepEqual(state.identity?.operations, ['publication:read'])
  assert.equal(state.tab, 'publication')
})
for (const capability of ['request:read-own', 'account:read']) {
  test(`preserves authorized Audit during refresh with ${capability} but no publication`, async () => {
    const { state, load } = harness(['audit:read', capability])
    await load(); assert.equal(state.tab, 'audit')
  })
}
test('initial unauthorized view falls back to an authorized view', async () => {
  const { state, load } = harness(['audit:read', 'request:read-own'], 'publication')
  await load(); assert.equal(state.tab, 'requests')
})
