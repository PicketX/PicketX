// SPDX-License-Identifier: Apache-2.0
// Success assertions read the actual Controller/SQLite, not response fixtures.
import assert from 'node:assert/strict'
import { join } from 'node:path'
import { readFile, writeFile } from 'node:fs/promises'
import { randomBytes, createHash } from 'node:crypto'

export async function initialHTTPAudit({ page, evidence }) {
  await page.getByRole('button', { name: 'HTTP decisions', exact: true }).click({ timeout: 5000 })
  const section = page.locator('.http-audit-section')
  await section.getByRole('heading', { name: 'HTTP authorization decisions', exact: true }).waitFor()
  await section.getByText('No HTTP decisions have been recorded.', { exact: true }).waitFor()
  assert.equal(await section.locator('[data-http-decision-id]').count(), 0)
  for (const width of [1440, 375]) {
    await page.setViewportSize({ width, height: 1000 })
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
    await page.screenshot({ path: join(evidence, `http-audit-empty-${width}.png`), fullPage: true })
  }
  await page.getByRole('button', { name: 'Publication history', exact: true }).click()
}

export async function populatedHTTPAudit({ browser, origin, authzPort, ingressPort, ingressToken, principalPath, evidence }) {
  const credential = randomBytes(32).toString('hex'), id = 'principal:audit-workflow'
  const principals = JSON.parse(await readFile(principalPath, 'utf8'))
  principals.principals.push({ id, sha256: createHash('sha256').update(credential).digest('hex'), roles: ['role:viewer'] })
  await writeFile(principalPath, JSON.stringify(principals), { mode: 0o600 })
  const context = await browser.newContext({ viewport: { width: 1440, height: 1000 }, reducedMotion: 'reduce' })
  const page = await context.newPage(), errors = []
  let projectedOperations = ['audit:read']
  page.on('pageerror', e => errors.push(e.message))
  const path = '**/console/api/v1/http-decisions?*'
  try {
    await page.clock.install()
    // UI capability projection fault only: real viewer credential/session and
    // actual audit endpoint remain unchanged; do not call this RBAC acceptance.
    await page.route('**/console/api/v1/me', async route => {
      const response = await route.fetch(), body = await response.json()
      await route.fulfill({ response, json: { ...body, operations: projectedOperations } })
    })
    await page.goto(origin + '/console/')
    await page.getByLabel('Access key', { exact: true }).fill(credential)
    await page.getByRole('button', { name: 'Sign in', exact: true }).click()
    await page.getByRole('button', { name: 'Audit', exact: true }).waitFor()
    assert.equal(await page.getByRole('button', { name: 'Resources', exact: true }).count(), 0)
    let reads = 0
    page.on('request', r => { if (r.url().includes('/console/api/v1/http-decisions?')) reads++ })
    await page.evaluate(() => Object.defineProperty(document, 'hidden', { configurable: true, get: () => true }))
    await page.getByRole('button', { name: 'HTTP decisions', exact: true }).click()
    const section = page.locator('.http-audit-section')
    await section.getByText('Reading HTTP decisions…', { exact: true }).waitFor()
    assert.equal(reads, 0, 'hidden initial mount must not read')
    await page.evaluate(() => { delete document.hidden; document.dispatchEvent(new Event('visibilitychange')) })
    const ids = () => section.locator('[data-http-decision-id]').evaluateAll(rows => rows.map(r => Number(r.dataset.httpDecisionId)))
    const readPage = async after => page.evaluate(async after => (await fetch(`/console/api/v1/http-decisions?after=${after}&limit=50`)).json(), after)
    const first = await readPage(0)
    assert(first.events.length > 0, 'existing real Nginx/application decisions required')
    await section.locator(`[data-http-decision-id="${first.events.at(-1).id}"]`).waitFor()
    assert.deepEqual(await ids(), first.events.map(e => e.id))
    // Capability projections exercise parent UI transitions, not server RBAC.
    const refresh = async () => {
      const me = page.waitForResponse(r => r.url().endsWith('/console/api/v1/me'))
      await page.getByRole('button', { name: 'Refresh status', exact: true }).click(); await me
      await page.waitForFunction(() => document.querySelector('.workspace')?.getAttribute('aria-busy') === 'false')
    }
    for (const extra of ['request:read-own', 'account:read']) {
      projectedOperations = ['audit:read', extra]; await refresh()
      await section.locator(`[data-http-decision-id="${first.events.at(-1).id}"]`).waitFor()
      const observed = page.waitForResponse(r => r.url().endsWith('/console/api/v1/me'))
      await page.getByRole('button', { name: 'Audit', exact: true }).click(); await observed
      await page.waitForFunction(() => document.querySelector('.workspace')?.getAttribute('aria-busy') === 'false')
      assert(await section.isVisible(), 'mixed-capability Audit navigation/refresh stays selected')
    }
    let releasePermission, enteredPermission, permissionDone
    const permissionGate = new Promise(r => { releasePermission = r }), permissionEntered = new Promise(r => { enteredPermission = r }), permissionComplete = new Promise(r => { permissionDone = r })
    await page.route(path, async route => {
      const response = await route.fetch(); enteredPermission(); await permissionGate
      try { await route.fulfill({ response }) } catch { /* revoked mount aborts */ } finally { permissionDone() }
    })
    await section.getByRole('button', { name: 'Read next decisions', exact: true }).click(); await permissionEntered
    projectedOperations = ['publication:read']
    await page.route('**/console/api/v1/publication', r => r.fulfill({ status: 503, json: { code: 'unavailable' } }))
    await refresh()
    assert.equal(await section.count(), 0, 'observed audit removal unmounts despite dependent read failure')
    assert.equal(await page.getByRole('button', { name: 'Audit', exact: true }).count(), 0)
    releasePermission(); await permissionComplete
    assert.equal(await page.locator('[data-http-decision-id]').count(), 0, 'late response cannot restore observed-revoked history')
    await page.unroute(path); await page.unroute('**/console/api/v1/publication')
    projectedOperations = ['audit:read']; await refresh()
    await section.locator(`[data-http-decision-id="${first.events.at(-1).id}"]`).waitFor()
    let actual = [...first.events], cursor = first.next_after, last = first
    const next = async () => {
      const response = page.waitForResponse(r => r.url().includes('/console/api/v1/http-decisions?') && r.request().method() === 'GET')
      await section.getByRole('button', { name: 'Read next decisions', exact: true }).click()
      const result = await response; assert.equal(result.status(), 200)
      const body = await result.json()
      if (body.events.length) await section.locator(`[data-http-decision-id="${body.events.at(-1).id}"]`).waitFor()
      else await section.getByText('No newer HTTP decisions at the last read.', { exact: true }).waitFor()
      return body
    }
    while (last.events.length === 50) { last = await next(); actual.push(...last.events); cursor = last.next_after }
    assert.deepEqual(await ids(), actual.slice(-200).map(e => e.id))
    for (const item of [actual.find(e => e.allowed && e.grant_ids.length), actual.find(e => e.reason === 'explicit-deny'), actual.find(e => e.source === '::1')]) {
      assert(item, 'real allow/deny/IPv6 records required')
      const row = section.locator(`[data-http-decision-id="${item.id}"]`)
      await row.getByText(item.allowed ? 'Allowed' : 'Denied', { exact: true }).waitFor()
      await row.getByText(item.source, { exact: true }).waitFor()
      await row.getByText('Decision details', { exact: true }).click()
      for (const contributing of [...item.grant_ids, ...item.deny_ids]) await row.getByText(contributing, { exact: true }).waitFor()
    }
    // Actual authenticated decisions create real durable pagination data. These
    // are API decisions, NOT claims of205 forwarded application requests.
    for (let n = 0; n < 205; n++) {
      const response = await fetch(`http://127.0.0.1:${authzPort}/authz/v1/decisions`, { method: 'POST',
        headers: { Authorization: `Bearer ${ingressToken}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ route: 'route-v1', source: '2001:db8::1', host: `127.0.0.1:${ingressPort}`, method: 'GET', uri: '/allowed' }) })
      assert.equal(response.status, 200); assert.equal((await response.json()).allowed, false)
    }
    do { last = await next(); actual.push(...last.events); cursor = last.next_after } while (last.events.length === 50)
    assert.equal((await ids()).length, 200)
    assert.deepEqual(await ids(), actual.slice(-200).map(e => e.id))
    const retained = await ids()
    await next(); assert.deepEqual(await ids(), retained, 'empty tail preserves history')
    for (const width of [1440, 768, 375]) {
      await page.setViewportSize({ width, height: 1000 })
      assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
      await page.screenshot({ path: join(evidence, `http-audit-records-${width}.png`), fullPage: false })
    }
    // Controlled transport/schema failures only. Prior successes above are real.
    for (const fault of [{ status: 503, json: { code: 'unavailable' } }, { status: 202, json: { events: [], next_after: cursor } }, { status: 200, json: { events: [], next_after: 0 } }]) {
      await page.route(path, r => r.fulfill(fault))
      await section.getByRole('button', { name: 'Read next decisions', exact: true }).click()
      await section.getByText('HTTP decision read failed. Earlier records are retained and may be behind newer decisions.', { exact: true }).waitFor()
      assert.deepEqual(await ids(), retained); assert.equal(Number(await section.getAttribute('data-after')), cursor)
      await page.unroute(path)
    }
    await next(); assert.deepEqual(await ids(), retained)
    let release, entered
    const gate = new Promise(r => { release = r }), waiting = new Promise(r => { entered = r })
    await page.route(path, async route => { entered(); await gate; await route.continue() })
    await section.getByRole('button', { name: 'Read next decisions', exact: true }).click(); await waiting
    await section.getByText('Reading HTTP decisions…', { exact: true }).waitFor()
    // Advance wall time without firing the separate15s transport timeout.
    // This is controlled clock suspension, not a61s live network acceptance.
    await page.clock.setSystemTime(await page.evaluate(() => Date.now() + 61000))
    release(); await page.waitForFunction(() => document.querySelector('.http-audit-section')?.getAttribute('aria-busy') === 'false')
    assert.equal(await section.locator('[role="alert"]').count(), 0, 'held read completed, not transport timeout')
    await section.getByText('No newer HTTP decisions at the last read.', { exact: true }).waitFor()
    await section.getByText('Loaded history may be behind newer decisions. Read again to update it.', { exact: true }).waitFor()
    await page.unroute(path)
    await section.getByRole('button', { name: 'Start from oldest', exact: true }).click()
    await section.locator(`[data-http-decision-id="${first.events.at(-1).id}"]`).waitFor()
    assert.deepEqual(await ids(), first.events.map(e => e.id))
    let releaseOld, oldEntered, oldDone, blocked = false
    const oldGate = new Promise(r => { releaseOld = r }), oldWaiting = new Promise(r => { oldEntered = r }), oldComplete = new Promise(r => { oldDone = r })
    await page.route(path, async route => {
      if (blocked) { await route.continue(); return }
      blocked = true
      const response = await route.fetch(); oldEntered(); await oldGate
      try { await route.fulfill({ response }) } catch { /* old mount explicitly aborts its request */ }
      finally { oldDone() }
    })
    await section.getByRole('button', { name: 'Read next decisions', exact: true }).click(); await oldWaiting
    await page.getByRole('button', { name: 'Publication history', exact: true }).click()
    await page.getByRole('button', { name: 'HTTP decisions', exact: true }).click()
    await section.locator(`[data-http-decision-id="${first.events.at(-1).id}"]`).waitFor()
    releaseOld(); await oldComplete
    await page.evaluate(() => new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r))))
    assert.deepEqual(await ids(), first.events.map(e => e.id), 'late old-page response cannot update the new mount')
    await page.unroute(path)
    await section.getByRole('button', { name: 'Read next decisions', exact: true }).focus()
    assert(await section.getByRole('button', { name: 'Read next decisions', exact: true }).evaluate(e => e === document.activeElement))
    await page.getByRole('button', { name: 'Language', exact: true }).click()
    await section.getByRole('heading', { name: 'HTTP 授权决策', exact: true }).waitFor()
    await page.getByRole('button', { name: '语言', exact: true }).click()
    await page.route(path, r => r.fulfill({ status: 403, json: { code: 'forbidden' } }))
    await section.getByRole('button', { name: 'Read next decisions', exact: true }).click()
    await section.getByText('Permission to read HTTP decisions is unavailable. Retry after refreshing your session.', { exact: true }).waitFor()
    assert.equal(await section.locator('[data-http-decision-id]').count(), 0)
    await page.unroute(path); await next()
    // Valid near-limit display stress is injected UI data, NOT real audit proof.
    const stressID = Number(await section.getAttribute('data-after')) + 10000
    const grantIDs = Array.from({ length: 4096 }, (_, n) => `grant-${n}-${'g'.repeat(235)}`)
    const denyIDs = Array.from({ length: 4096 }, (_, n) => `deny-${n}-${'d'.repeat(235)}`)
    await page.route(path, r => r.fulfill({ status: 200, json: { events: [{ ...first.events[0], id: stressID, grant_ids: grantIDs, deny_ids: denyIDs }], next_after: stressID } }))
    await next(); await page.unroute(path)
    const stress = section.locator(`[data-http-decision-id="${stressID}"]`)
    assert.equal(await stress.locator('li').count(), 0, 'collapsed near-limit record has no ID DOM')
    await stress.getByText('Decision details', { exact: true }).click()
    await stress.getByText(grantIDs[0], { exact: true }).waitFor()
    assert.equal(await stress.locator('li').count(), 100)
    const grantList = stress.locator('.http-contributions').first()
    let offset = 0
    while (!(await grantList.getByRole('button', { name: 'Next IDs', exact: true }).isDisabled())) {
      await grantList.getByRole('button', { name: 'Next IDs', exact: true }).click(); offset += 50
      await grantList.getByText(grantIDs[offset], { exact: true }).waitFor()
      assert((await stress.locator('li').count()) <= 100)
    }
    await grantList.getByText(grantIDs.at(-1), { exact: true }).waitFor()
    await page.setViewportSize({ width: 375, height: 1000 })
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
    await page.screenshot({ path: join(evidence, 'http-audit-contributions-375.png'), fullPage: false })
    await stress.getByText('Decision details', { exact: true }).click()
    await page.waitForFunction(id => document.querySelector(`[data-http-decision-id="${id}"]`)?.querySelectorAll('li').length === 0, stressID)
    // Actual credential removal, not a mocked401: private history must disappear.
    const current = JSON.parse(await readFile(principalPath, 'utf8'))
    current.principals = current.principals.filter(p => p.id !== id)
    await writeFile(principalPath, JSON.stringify(current), { mode: 0o600 })
    await section.getByRole('button', { name: 'Read next decisions', exact: true }).click()
    await page.getByLabel('Access key', { exact: true }).waitFor()
    assert.equal(await page.locator('[data-http-decision-id]').count(), 0)
    assert.equal(await page.evaluate(() => localStorage.length + sessionStorage.length), 0)
    assert.deepEqual(errors, [])
    console.log('PASS actual HTTP decision audit: real Nginx/SQLite allow-deny-IPv6-contributing IDs,205 realAPI decisions, keyset/window/emptytail, loading/failure/age, language/mobile and credential removal; controlled mixed-capability/observed-revocation/4096-ID UI stress separately')
  } finally {
    await context.close()
    const current = JSON.parse(await readFile(principalPath, 'utf8'))
    current.principals = current.principals.filter(p => p.id !== id)
    await writeFile(principalPath, JSON.stringify(current), { mode: 0o600 })
  }
}
