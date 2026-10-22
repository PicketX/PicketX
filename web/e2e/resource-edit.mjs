// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict'
import { join } from 'node:path'
import { rejectedMutationReceipt } from './mutation-receipts.mjs'

export async function resourceNetworkEditing({ operator: p, view, origin, token, evidence }) {
  const headers = { Authorization: `Bearer ${token}` }
  const inventory = async () => (await fetch(`${origin}/api/v1/inventory`, { headers })).json()
  const write = async (path, body, method = 'PUT') => {
    const before = await fetch(`${origin}/api/v1/publication`, { headers })
    return fetch(`${origin}/api/v1${path}`, { method, headers: { ...headers, 'Content-Type': 'application/json', 'If-Match': before.headers.get('etag') }, body: JSON.stringify(body) })
  }
  const settled = () => p.waitForFunction(() => document.querySelector('.resources-section')?.getAttribute('aria-busy') === 'false')
  const save = () => p.getByRole('button', { name: 'Save network parameters', exact: true })
  const ack = () => p.getByRole('checkbox', { name: 'I reviewed the latest target and network changes', exact: true })
  const rebase = async () => {
    p.once('dialog', d => d.accept())
    await p.getByRole('button', { name: 'Use latest resource revision', exact: true }).click(); await settled()
  }
  await p.getByRole('button', { name: 'Resources', exact: true }).click(); await settled()
  // The previous workflow intentionally retained a read-failure draft.
  p.once('dialog', d => d.accept())
  await p.getByRole('button', { name: 'Clear resource draft', exact: true }).click()
  const endpoints = [{ name: 'web', protocol: 'TCP', port: 9401, addresses: ['192.0.2.90', '2001:db8::90'] }]
  const created = await write('/resources', { name: 'network-edit-target', endpoints, reason: 'Prepare network edit' }, 'POST')
  assert.equal(created.status, 201); const target = await created.json()
  await p.getByRole('button', { name: 'Use latest resource revision', exact: true }).click(); await settled()
  const open = uid => p.locator(`[data-resource-id="${uid}"]`).getByRole('button', { name: 'Edit network parameters', exact: true }).click()
  await open(target.uid)
  assert.equal(await p.getByLabel('Resource name', { exact: true }).inputValue(), 'network-edit-target')
  assert(await p.getByLabel('Resource name', { exact: true }).evaluate(e => e.readOnly))
  assert(await p.getByLabel('Endpoint 1 name', { exact: true }).evaluate(e => e.readOnly))
  assert.equal(await p.getByRole('button', { name: 'Add Endpoint', exact: true }).count(), 0)
  await p.getByLabel('Endpoint 1 port', { exact: true }).fill('9402')
  await p.getByRole('button', { name: 'Cancel network edit', exact: true }).click()
  assert.equal((await inventory()).resources.find(r => r.uid === target.uid).endpoints[0].port, 9401)
  await open(target.uid)
  await p.getByLabel('Resource change reason', { exact: true }).fill('Unchanged network')
  await ack().check(); assert(await save().isDisabled()); await ack().uncheck()
  await p.getByLabel('Endpoint 1 port', { exact: true }).fill('9443')
  await p.getByLabel('Resource change reason', { exact: true }).fill('Browser network correction')
  assert(await save().isDisabled()); await ack().check()
  const path = `/resources/${target.uid}/network`
  const concurrent = await write(path, { endpoints: [{ ...endpoints[0], port: 9403 }], reason: 'Concurrent network correction' })
  assert.equal(concurrent.status, 200)
  await p.getByRole('button', { name: 'Refresh resources', exact: true }).click(); await settled()
  assert.equal(await p.getByLabel('Endpoint 1 port', { exact: true }).inputValue(), '9443')
  const stale = p.waitForResponse(r => r.url().endsWith(`/console/api/v1${path}`))
  await save().click(); assert.equal((await stale).status(), 412); await settled()
  await rebase(); assert.equal(await ack().isChecked(), false)
  assert.equal(await p.getByLabel('Endpoint 1 port', { exact: true }).inputValue(), '9443')
  await p.locator('.network-edit-current').getByText(/TCP\/9403/).waitFor()
  await ack().check()
  await rejectedMutationReceipt({ p, path, section: '.resources-section', submit: 'Save network parameters', labels: ['Endpoint 1 port', 'Resource change reason'] })
  await rebase(); await ack().check()
  await p.route(`**/console/api/v1${path}`, r => r.fulfill({ status: 200, contentType: 'application/json', body: '{}' }))
  await save().click(); await p.locator('.resources-section [role="alert"]').waitFor(); await settled()
  assert(await save().isDisabled()); assert.equal(await p.getByLabel('Endpoint 1 port', { exact: true }).inputValue(), '9443')
  await p.unroute(`**/console/api/v1${path}`); await rebase()
  await p.setViewportSize({ width: 375, height: 812 })
  assert(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
  await p.screenshot({ path: join(evidence, 'resource-network-edit-mobile.png'), fullPage: true })
  await ack().check()
  const accepted = p.waitForResponse(r => r.url().endsWith(`/console/api/v1${path}`))
  await save().click(); assert.equal((await accepted).status(), 200); await settled()
  assert.deepEqual((await inventory()).resources.find(r => r.uid === target.uid).endpoints, [{ ...endpoints[0], port: 9443 }])
  const audit = await (await fetch(`${origin}/api/v1/audit?limit=100`, { headers })).json()
  assert(audit.events.some(e => e.reason === 'Browser network correction' && e.actor === 'principal:admin'))
  await p.setViewportSize({ width: 1440, height: 1000 })
  await p.screenshot({ path: join(evidence, 'resource-network-edit-desktop.png'), fullPage: true })
  // Referenced targets remain immutable even when visible/editable in stale inventory.
  await open('resource-gitea-v1')
  await p.getByLabel('Endpoint 1 port', { exact: true }).fill('9444')
  await p.getByLabel('Resource change reason', { exact: true }).fill('Do not retarget existing authorization')
  await ack().check()
  const denied = p.waitForResponse(r => r.url().endsWith('/resources/resource-gitea-v1/network'))
  await save().click(); assert.equal((await denied).status(), 409); await settled()
  assert.equal(await p.getByLabel('Resource change reason', { exact: true }).inputValue(), 'Do not retarget existing authorization')
  await p.getByRole('button', { name: 'Cancel network edit', exact: true }).click()
  await open(target.uid); await p.getByLabel('Resource change reason', { exact: true }).fill('Retain missing identity')
  const removed = await write(`/resources/${target.uid}/remove`, { reason: 'External removal', acknowledge_removal: true }, 'POST')
  assert.equal(removed.status, 200)
  const replacement = await write('/resources', { name: 'network-edit-target', endpoints, reason: 'Same name different identity' }, 'POST')
  assert.equal(replacement.status, 201); assert.notEqual((await replacement.json()).uid, target.uid)
  await rebase(); await ack().check(); assert(await save().isDisabled())
  await p.getByText('This Resource is absent from the draft revision. Cancel and inspect the latest inventory; a same-name replacement is a different target.', { exact: true }).waitFor()
  await p.getByRole('button', { name: 'Cancel network edit', exact: true }).click()
  // A full-publication writer may reorder or change membership. Identity is
  // the Endpoint name, never its array position in a retained edit draft.
  const pair = [{ name: 'a', protocol: 'TCP', port: 80 }, { name: 'b', protocol: 'TCP', port: 443 }]
  const paired = await write('/resources', { name: 'network-edit-pair', endpoints: pair, reason: 'Prepare named comparison' }, 'POST')
  assert.equal(paired.status, 201); const pairTarget = await paired.json()
  await p.getByRole('button', { name: 'Use latest resource revision', exact: true }).click(); await settled()
  await open(pairTarget.uid)
  await p.getByLabel('Resource change reason', { exact: true }).fill('Compare immutable Endpoint names')
  const replacePair = async change => {
    const current = await (await fetch(`${origin}/api/v1/publication`, { headers })).json()
    let count = 0
    const policy = current.policy.split('\n').map(line => {
      if (!line.startsWith('{')) return line
      const doc = JSON.parse(line)
      if (doc.kind !== 'Resource' || doc.metadata.uid !== pairTarget.uid) return line
      count++; change(doc.spec.endpoints); return JSON.stringify(doc)
    }).join('\n')
    assert.equal(count, 1)
    const result = await write('/publication', { policy, facts: current.facts, reason: 'Concurrent Endpoint layout change' })
    assert.equal(result.status, 200)
  }
  await replacePair(entries => entries.reverse())
  await rebase(); await ack().check()
  assert(await save().isDisabled(), 'Reordering must not turn an unchanged named draft into a change')
  await p.getByLabel('Endpoint 1 port', { exact: true }).fill('443')
  await p.getByLabel('Endpoint 2 port', { exact: true }).fill('80')
  await ack().check()
  assert(await save().isEnabled(), 'Named network changes must remain saveable after a concurrent reorder')
  const pairSaved = p.waitForResponse(r => r.url().endsWith(`/resources/${pairTarget.uid}/network`))
  await save().click(); assert.equal((await pairSaved).status(), 200); await settled()
  assert.deepEqual((await inventory()).resources.find(r => r.uid === pairTarget.uid).endpoints,
    [{ ...pair[1], port: 80 }, { ...pair[0], port: 443 }])
  await open(pairTarget.uid)
  await p.getByLabel('Endpoint 1 port', { exact: true }).fill('81')
  await p.getByLabel('Resource change reason', { exact: true }).fill('Do not overwrite changed membership')
  await replacePair(entries => entries.push({ name: 'c', network: { protocol: 'TCP', port: 8443 } }))
  await rebase(); await ack().check()
  assert(await save().isDisabled(), 'Incompatible Endpoint membership requires cancelling the old draft')
  await p.getByText('Endpoint membership changed in the latest revision. Cancel this draft and reopen the Resource before editing.', { exact: true }).waitFor()
  assert.equal(await p.getByLabel('Endpoint 1 port', { exact: true }).inputValue(), '81')
  await p.getByRole('button', { name: 'Cancel network edit', exact: true }).click()
  await view.getByRole('button', { name: 'Resources', exact: true }).click()
  assert.equal(await view.getByRole('button', { name: 'Edit network parameters', exact: true }).count(), 0)
  const forbidden = await view.evaluate(async () => {
    const { csrf } = await (await fetch('/console/api/v1/session')).json()
    return (await fetch('/console/api/v1/resources/resource-gitea-v1/network', { method: 'PUT', headers: { 'X-CSRF-Token': csrf, 'Content-Type': 'application/json', 'If-Match': '"1"' }, body: '{}' })).status
  }); assert.equal(forbidden, 403)
  console.log('PASS actual browser Resource network edit → SQLite/audit; cancel, reference refusal, stale CAS, latest preview, retained draft, incomplete receipts, missing UID and viewer; not Agent application')
}
