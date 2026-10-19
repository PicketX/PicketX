// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict'
import { join } from 'node:path'
import { rejectedMutationReceipt } from './mutation-receipts.mjs'

export async function bindingEndpointEditing({ operator: p, view, origin, token, evidence }) {
  const headers = { Authorization: `Bearer ${token}` }
  const inventory = async () => (await fetch(`${origin}/api/v1/inventory`, { headers })).json()
  const write = async (path, body, method = 'PUT') => {
    const before = await fetch(`${origin}/api/v1/publication`, { headers })
    return fetch(`${origin}/api/v1${path}`, { method, headers: { ...headers, 'Content-Type': 'application/json', 'If-Match': before.headers.get('etag') }, body: JSON.stringify(body) })
  }
  const settled = () => p.waitForFunction(() => document.querySelector('.resources-section')?.getAttribute('aria-busy') === 'false')
  const save = () => p.getByRole('button', { name: 'Save protected Endpoints', exact: true })
  const ack = () => p.getByRole('checkbox', { name: 'I reviewed the latest scope and protection changes', exact: true })
  const choice = name => p.locator('#binding-edit-endpoints').getByRole('checkbox', { name: new RegExp(`^${name} `) })
  const rebase = async () => {
    p.once('dialog', d => d.accept())
    await p.getByRole('button', { name: 'Use latest resource revision', exact: true }).click(); await settled()
  }
  await p.getByRole('button', { name: 'Resources', exact: true }).click(); await settled()
  const endpoints = [{ name: 'web', protocol: 'TCP', port: 9501 }, { name: 'ssh', protocol: 'TCP', port: 9502 }]
  const created = await write('/resources', { name: 'binding-edit-resource', endpoints, reason: 'Prepare binding edit' }, 'POST')
  assert.equal(created.status, 201); const resource = await created.json()
  const added = await write('/bindings', { name: 'binding-edit-target', resource_name: 'binding-edit-resource', resource_uid: resource.uid, endpoint_names: ['web'], agent_name: 'agent-a', context_id: 'host-v1', reason: 'Prepare explicit protection' }, 'POST')
  assert.equal(added.status, 201); const target = await added.json()
  const path = `/bindings/${target.uid}/endpoints`
  const initial = (await inventory()).bindings.find(b => b.uid === target.uid)
  await p.getByRole('button', { name: 'Use latest resource revision', exact: true }).click(); await settled()
  const open = () => p.locator(`[data-binding-id="${target.uid}"]`).getByRole('button', { name: 'Edit protected Endpoints', exact: true }).click()
  await open(); await choice('ssh').check()
  await p.getByRole('button', { name: 'Cancel Binding edit', exact: true }).click()
  assert.deepEqual((await inventory()).bindings.find(b => b.uid === target.uid), initial)
  await open(); await ack().check(); assert(await save().isDisabled()); await ack().uncheck()
  await choice('ssh').check()
  await p.getByLabel('Binding edit reason', { exact: true }).fill('Browser protection adjustment')
  assert(await save().isDisabled()); await ack().check()
  // A concurrent writer changes named protection, not the fixed scope.
  const concurrent = await write(path, { endpoint_names: ['ssh'], reason: 'Concurrent protection selection', acknowledge_scope_change: true })
  assert.equal(concurrent.status, 200)
  await p.getByRole('button', { name: 'Refresh resources', exact: true }).click(); await settled()
  assert(await choice('web').isChecked()); assert(await choice('ssh').isChecked())
  const stale = p.waitForResponse(r => r.url().endsWith(`/console/api/v1${path}`))
  await save().click(); assert.equal((await stale).status(), 412); await settled()
  await rebase(); assert.equal(await ack().isChecked(), false)
  assert(await choice('web').isChecked()); assert(await choice('ssh').isChecked())
  await p.locator('.binding-edit-current').getByText('ssh', { exact: true }).waitFor()
  await ack().check()
  await rejectedMutationReceipt({ p, path, section: '.resources-section', submit: 'Save protected Endpoints', labels: ['Binding edit reason'] })
  await rebase(); await ack().check()
  await p.route(`**/console/api/v1${path}`, r => r.fulfill({ status: 200, contentType: 'application/json', body: '{}' }))
  await save().click(); await p.locator('.resources-section [role="alert"]').waitFor(); await settled()
  assert(await save().isDisabled()); assert.equal(await p.getByLabel('Binding edit reason', { exact: true }).inputValue(), 'Browser protection adjustment')
  await p.unroute(`**/console/api/v1${path}`); await rebase()
  await p.setViewportSize({ width: 375, height: 812 })
  assert(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
  await p.screenshot({ path: join(evidence, 'binding-endpoint-edit-mobile.png'), fullPage: true })
  await ack().check()
  const accepted = p.waitForResponse(r => r.url().endsWith(`/console/api/v1${path}`))
  await save().click(); assert.equal((await accepted).status(), 200); await settled()
  const saved = (await inventory()).bindings.find(b => b.uid === target.uid)
  assert.deepEqual(saved.endpoint_edit.endpoint_names, ['web', 'ssh'])
  assert.equal(saved.name, initial.name); assert.equal(saved.agents, initial.agents); assert.equal(saved.enforcement, initial.enforcement)
  const audit = await (await fetch(`${origin}/api/v1/audit?limit=100`, { headers })).json()
  assert(audit.events.some(e => e.reason === 'Browser protection adjustment' && e.actor === 'principal:admin'))
  await p.setViewportSize({ width: 1440, height: 1000 })
  await open()
  await p.screenshot({ path: join(evidence, 'binding-endpoint-edit-desktop.png'), fullPage: true })
  await choice('web').uncheck()
  await p.getByLabel('Binding edit reason', { exact: true }).fill('Never move protection silently')
  const replacement = await write('/resources', { name: 'binding-edit-other-resource', endpoints, reason: 'Prepare concurrent pinned Resource change' }, 'POST')
  assert.equal(replacement.status, 201); const other = await replacement.json()
  const current = await (await fetch(`${origin}/api/v1/publication`, { headers })).json()
  let count = 0
  const policy = current.policy.split('\n').map(line => {
    if (!line.startsWith('{')) return line
    const doc = JSON.parse(line)
    if (doc.kind !== 'EnforcementBinding' || doc.metadata.uid !== target.uid) return line
    count++; doc.spec.resources[0].resourceRef = { name: 'binding-edit-other-resource', uid: other.uid }; return JSON.stringify(doc)
  }).join('\n')
  assert.equal(count, 1)
  assert.equal((await write('/publication', { policy, facts: current.facts, reason: 'Concurrent execution scope change' })).status, 200)
  await rebase(); await ack().check(); assert(await save().isDisabled())
  await p.getByText('This Binding’s fixed scope or available Endpoints changed. Cancel and reopen it before editing; the retained draft will not move protection.', { exact: true }).waitFor()
  assert.equal(await p.getByLabel('Binding edit reason', { exact: true }).inputValue(), 'Never move protection silently')
  await p.getByRole('button', { name: 'Cancel Binding edit', exact: true }).click()
  await open(); await choice('web').uncheck()
  assert.equal((await write(`/bindings/${target.uid}/remove`, { reason: 'External Binding retirement', acknowledge_unprotection: true }, 'POST')).status, 200)
  await rebase(); await ack().check(); assert(await save().isDisabled())
  await p.getByRole('button', { name: 'Cancel Binding edit', exact: true }).click()
  await view.getByRole('button', { name: 'Resources', exact: true }).click()
  assert.equal(await view.getByRole('button', { name: 'Edit protected Endpoints', exact: true }).count(), 0)
  const forbidden = await view.evaluate(async path => {
    const { csrf } = await (await fetch('/console/api/v1/session')).json()
    return (await fetch(`/console/api/v1${path}`, { method: 'PUT', headers: { 'X-CSRF-Token': csrf, 'Content-Type': 'application/json', 'If-Match': '"1"' }, body: '{}' })).status
  }, path); assert.equal(forbidden, 403)
  console.log('PASS actual browser Binding Endpoint edit → API/SQLite/audit; cancel, fixed-scope preservation, stale CAS/rebase, incomplete receipts, changed scope/missing target lock and viewer; not Agent application')
}
