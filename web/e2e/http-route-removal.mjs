// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict'
import { writeFile } from 'node:fs/promises'
import { join } from 'node:path'
import { createHash } from 'node:crypto'
import { freePort, startIngress } from './ingress.mjs'

export async function httpRouteRemoval({ operator, view, api, route, ingressPort, root, temp, authzPort, ingressToken, callersPath, evidence }) {
  const button = name => operator.getByRole('button', { name, exact: true })
  const confirm = button('Confirm HTTP route removal')
  const reason = operator.getByLabel('Route removal reason', { exact: true })
  const ack = operator.getByRole('checkbox', { name: 'I understand new requests for this route will be denied', exact: true })
  const rebase = async () => {
    operator.once('dialog', d => d.accept())
    await button('Use latest route revision').click()
    await operator.waitForFunction(() => document.querySelector('.http-routes-section')?.getAttribute('aria-busy') === 'false')
  }
  const mutate = async (path, body) => {
    const base = await api('/publication')
    return api(path, { method: 'POST', headers: { 'If-Match': base.response.headers.get('etag') }, body: JSON.stringify(body) })
  }
  operator.once('dialog', d => d.accept())
  await button('Clear route draft').click()
  const siblingPort = await freePort()
  const siblingBody = { name: 'removal-sibling', resource_name: 'gitea', resource_uid: 'resource-gitea-v1', endpoint_name: 'https', host: `127.0.0.1:${siblingPort}`, methods: ['GET'], paths: ['/allowed'], reason: 'Independent sibling' }
  const sibling = await mutate('/http-routes', siblingBody); assert.equal(sibling.response.status, 201)
  const grant = await mutate('/grants', { name: 'removal-live-source', resource_name: 'gitea', resource_uid: 'resource-gitea-v1', endpoint_names: ['https'], source: '127.0.0.1', duration_seconds: 900, reason: 'Keep network access during route removal' })
  assert.equal(grant.response.status, 201)
  await writeFile(callersPath, JSON.stringify({ callers: [{ id: 'nginx', sha256: createHash('sha256').update(ingressToken).digest('hex'), routes: [route.uid, sibling.data.uid] }] }), { mode: 0o600 })
  const ingress = await startIngress({ root, temp: join(temp, 'removed-route-ingress'), authzPort, token: ingressToken, routeID: route.uid, listenPort: ingressPort })
  const other = await startIngress({ root, temp: join(temp, 'sibling-route-ingress'), authzPort, token: ingressToken, routeID: sibling.data.uid, listenPort: siblingPort })
  try {
    await ingress.verifyExactRoute(true); await other.verifyExactRoute(true)
    // First missing-control failure on the test-only head is the UI RED gate.
    await button('Use latest route revision').click()
    const open = uid => operator.locator(`[data-route-id="${uid}"]`).getByRole('button', { name: 'Remove HTTP route', exact: true }).click()
    await open(route.uid)
    assert(await confirm.isDisabled())
    await reason.fill('Retire browser route')
    await ack.check()
    await button('Cancel route removal').click()
    await ingress.verifyExactRoute(true)
    await open(route.uid); await reason.fill('Retire browser route'); await ack.check()
    // Ambiguous success is not a receipt and must not erase intent or repeat a write.
    for (const status of [200, 204, 202]) {
      const current = await api('/publication')
      const next = current.data.revision + 1
      let attempts = 0
      await operator.route(`**/console/api/v1/http-routes/${route.uid}/remove`, r => {
        attempts++
        return r.fulfill({ status, ...(status === 200 ? { contentType: 'application/json', body: '{}' } : status === 202 ? { contentType: 'application/json', headers: { ETag: `"${next}"` }, body: JSON.stringify({ uid: route.uid, revision: next }) } : {}) })
      })
      const pending = operator.waitForResponse(r => r.url().endsWith(`/${route.uid}/remove`))
      await confirm.click(); assert.equal((await pending).status(), status)
      await operator.waitForFunction(() => document.querySelector('.http-routes-section')?.getAttribute('aria-busy') === 'false')
      assert.equal(await confirm.count(), 1, `HTTP ${status} must retain removal intent`)
      assert.equal(await operator.locator('.route-result').count(), 0, 'Ambiguous response must not announce completed removal')
      assert.equal(attempts, 1, 'No automatic mutation retry')
      assert.equal(await reason.inputValue(), 'Retire browser route'); assert(await confirm.isDisabled())
      await operator.unroute(`**/console/api/v1/http-routes/${route.uid}/remove`)
      await rebase(); assert(!(await ack.isChecked())); await ack.check()
    }
    const concurrent = await mutate('/resources', { name: 'http-route-removal-concurrent', endpoints: [{ name: 'web', protocol: 'TCP', port: 9444 }], reason: 'Stale removal base' })
    assert.equal(concurrent.response.status, 201, JSON.stringify(concurrent.data))
    await button('Refresh routes').click()
    let pending = operator.waitForResponse(r => r.url().endsWith(`/${route.uid}/remove`))
    await confirm.click(); assert.equal((await pending).status(), 412)
    assert.equal(await reason.inputValue(), 'Retire browser route')
    await rebase(); assert(!(await ack.isChecked())); await ack.check()
    await operator.route('**/console/api/v1/inventory', r => r.fulfill({ status: 503, contentType: 'application/json', body: '{"code":"management_unavailable"}' }))
    await button('Refresh routes').click(); await operator.locator('.http-routes-section .stale').waitFor()
    assert.equal(await reason.inputValue(), 'Retire browser route')
    await operator.unroute('**/console/api/v1/inventory')
    await operator.setViewportSize({ width: 375, height: 812 })
    assert(await operator.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
    await operator.screenshot({ path: join(evidence, 'http-route-removal-mobile.png'), fullPage: true })
    const before = await api('/publication')
    await operator.route('**/console/api/v1/inventory', r => r.fulfill({ status: 503, contentType: 'application/json', body: '{"code":"management_unavailable"}' }))
    pending = operator.waitForResponse(r => r.url().endsWith(`/${route.uid}/remove`))
    await confirm.click(); assert.equal((await pending).status(), 200)
    await operator.locator('.route-result').waitFor()
    await operator.waitForFunction(() => document.querySelector('.http-routes-section')?.getAttribute('aria-busy') === 'false')
    assert.equal(await confirm.count(), 0, 'Successful removal must not remain retryable after a failed observation')
    assert.equal(await operator.getByLabel('Route name', { exact: true }).inputValue(), '')
    await operator.unroute('**/console/api/v1/inventory')
    await button('Use latest route revision').click()
    await operator.locator(`[data-route-id="${route.uid}"]`).waitFor({ state: 'detached' })
    await ingress.verifyExactRoute(false); await other.verifyExactRoute(true)
    const after = await api('/publication')
    assert.equal(after.data.facts, before.data.facts)
    const grants = await api('/grants'); assert(grants.data.grants.some(g => g.uid === grant.data.uid && !g.revoked))
    const audit = await api('/audit?limit=100'); assert(audit.data.events.some(e => e.reason === 'Retire browser route'))
    await operator.setViewportSize({ width: 1440, height: 1000 })
    await operator.screenshot({ path: join(evidence, 'http-route-removal-desktop.png'), fullPage: true })
    await open(sibling.data.uid); await reason.fill('Must retain missing identity'); await ack.check()
    const removed = await mutate(`/http-routes/${sibling.data.uid}/remove`, { reason: 'External removal', acknowledge_removal: true }); assert.equal(removed.response.status, 200)
    const replacement = await mutate('/http-routes', siblingBody); assert.equal(replacement.response.status, 201); assert.notEqual(replacement.data.uid, sibling.data.uid)
    await rebase(); assert(!(await ack.isChecked())); assert(await confirm.isDisabled())
    await operator.getByText('This HTTP route is absent from the draft revision. Cancel and inspect the latest inventory; a same-name replacement is a different target.', { exact: true }).waitFor()
    assert.equal(await reason.inputValue(), 'Must retain missing identity')
    await button('Cancel route removal').click()
    await view.getByRole('button', { name: 'Refresh routes', exact: true }).click()
    assert.equal(await view.getByRole('button', { name: 'Remove HTTP route', exact: true }).count(), 0)
  } finally { await ingress.close(); await other.close() }
  console.log('PASS HTTPRoute removal: real browser/API/audit/CAS; active Grant preserved, actual Nginx denies removed UID while sibling permits; cancel, invalid receipts, rebase, missing UID, viewer/mobile')
}
