// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict'
import { writeFile } from 'node:fs/promises'
import { join } from 'node:path'
import { createHash } from 'node:crypto'
import { freePort, startIngress } from './ingress.mjs'
import { httpRouteRemoval } from './http-route-removal.mjs'

export async function httpRouteManagement({ operator, view, origin, token, root, temp, authzPort, ingressToken, callersPath, evidence }) {
  const ingressPort = await freePort()
  const api = async (path, options = {}) => {
    const response = await fetch(`${origin}/api/v1${path}`, { ...options, headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json', ...options.headers } })
    return { response, data: await response.json() }
  }
  await operator.getByRole('button', { name: 'HTTP routes', exact: true }).click()
  await operator.getByRole('button', { name: 'Use latest route revision', exact: true }).click()
  await operator.getByText('No HTTP routes yet. Select a TCP endpoint and define exact request selectors.', { exact: true }).waitFor()
  await operator.getByRole('button', { name: 'Create HTTP route', exact: true }).click()
  await operator.waitForFunction(() => document.querySelector('.http-routes-section [role="alert"]') === document.activeElement)
  await operator.getByLabel('Route name', { exact: true }).fill('browser-route')
  await operator.getByLabel('Route Resource', { exact: true }).selectOption('resource-gitea-v1')
  await operator.getByLabel('TCP endpoint', { exact: true }).selectOption('https')
  await operator.getByLabel('Exact request Host', { exact: true }).fill(`127.0.0.1:${ingressPort}`)
  await operator.getByRole('checkbox', { name: 'GET', exact: true }).check()
  await operator.getByLabel('Exact paths, one per line', { exact: true }).fill('/allowed')
  await operator.getByLabel('Route creation reason', { exact: true }).fill('Browser exact route')
  const before = await api('/publication')
  // Real competing writer changes publication content, not just its reason.
  const competing = await api('/resources', { method: 'POST', headers: { 'If-Match': before.response.headers.get('etag') }, body: JSON.stringify({ name: 'route-concurrent', reason: 'Concurrent route writer', endpoints: [{ name: 'web', protocol: 'TCP', port: 9443 }] }) })
  assert.equal(competing.response.status, 201)
  let pending = operator.waitForResponse(r => r.url().endsWith('/http-routes') && r.request().method() === 'POST')
  await operator.getByRole('button', { name: 'Create HTTP route', exact: true }).click(); assert.equal((await pending).status(), 412)
  assert.equal(await operator.getByLabel('Route name', { exact: true }).inputValue(), 'browser-route')
  assert(await operator.getByRole('button', { name: 'Create HTTP route', exact: true }).isDisabled())
  operator.once('dialog', d => d.accept())
  await operator.getByRole('button', { name: 'Use latest route revision', exact: true }).click()
  pending = operator.waitForResponse(r => r.url().endsWith('/http-routes') && r.request().method() === 'POST')
  await operator.getByRole('button', { name: 'Create HTTP route', exact: true }).click()
  const created = await pending; assert.equal(created.status(), 201); const route = await created.json()
  await operator.locator(`[data-route-id="${route.uid}"]`).waitFor()
  assert.equal(await operator.getByLabel('Route name', { exact: true }).inputValue(), '')
  const publication = await api('/publication'); assert.equal(publication.data.facts, before.data.facts)
  const audit = await api('/audit?limit=100'); assert(audit.data.events.some(e => e.reason === 'Browser exact route'))
  await operator.screenshot({ path: join(evidence, 'http-routes-desktop.png'), fullPage: true })
  // The operator must configure this explicit scope independently. No mock
  // authorization responses or implicit client-controlled route are involved.
  await writeFile(callersPath, JSON.stringify({ callers: [{ id: 'nginx', sha256: createHash('sha256').update(ingressToken).digest('hex'), routes: [route.uid] }] }), { mode: 0o600 })
  const ingress = await startIngress({ root, temp: join(temp, 'route-ingress'), authzPort, token: ingressToken, routeID: route.uid, listenPort: ingressPort })
  try {
    await ingress.verifyExactRoute(false)
    const grantBase = await api('/publication')
    const grant = await api('/grants', { method: 'POST', headers: { 'If-Match': grantBase.response.headers.get('etag') }, body: JSON.stringify({ name: 'route-source', resource_name: 'gitea', resource_uid: 'resource-gitea-v1', endpoint_names: ['https'], source: '127.0.0.1', duration_seconds: 900, reason: 'Route source access' }) })
    assert.equal(grant.response.status, 201, JSON.stringify(grant.data))
    await ingress.verifyExactRoute(true)
    const application = await operator.context().newPage()
    await application.goto(`${ingress.url}/allowed`)
    assert.equal(await application.locator('body').textContent(), 'protected real HTTP application'); await application.close()
    const revokeBase = await api('/publication')
    const revoke = await api(`/grants/${grant.data.uid}/revoke`, { method: 'POST', headers: { 'If-Match': revokeBase.response.headers.get('etag') }, body: JSON.stringify({ reason: 'Route source revoked' }) })
    assert.equal(revoke.response.status, 200, JSON.stringify(revoke.data)); await ingress.verifyExactRoute(false)
  } finally { await ingress.close() }
  await operator.getByLabel('Route name', { exact: true }).fill('preserved-route-draft')
  await operator.route('**/console/api/v1/inventory', r => r.fulfill({ status: 503, contentType: 'application/json', body: '{"code":"management_unavailable"}' }))
  await operator.getByRole('button', { name: 'Refresh routes', exact: true }).click()
  await operator.locator('.http-routes-section .stale').waitFor()
  assert.equal(await operator.getByLabel('Route name', { exact: true }).inputValue(), 'preserved-route-draft')
  await operator.unroute('**/console/api/v1/inventory')
  await operator.setViewportSize({ width: 375, height: 812 })
  assert(await operator.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
  await operator.screenshot({ path: join(evidence, 'http-routes-mobile.png'), fullPage: true })
  await view.getByRole('button', { name: 'HTTP routes', exact: true }).click()
  await view.getByRole('button', { name: 'Refresh routes', exact: true }).click()
  await view.locator(`[data-route-id="${route.uid}"]`).waitFor()
  assert.equal(await view.getByRole('button', { name: 'Create HTTP route', exact: true }).count(), 0)
  assert(await view.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
  await view.screenshot({ path: join(evidence, 'http-routes-viewer-mobile.png'), fullPage: true })
  await httpRouteRemoval({ operator, view, api, route, ingressPort, root, temp, authzPort, ingressToken, callersPath, evidence })
  console.log('PASS real HTTPRoute browser creation → CAS/audit → explicit caller/Nginx → application; source deny/grant/revoke, exact selectors, preserved draft, failed read, viewer and mobile')
}
