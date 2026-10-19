// SPDX-License-Identifier: Apache-2.0
// A separate real process isolates the sticky DB-user authority from file-user suites.
import assert from 'node:assert/strict'
import { mkdir, writeFile } from 'node:fs/promises'
import { join } from 'node:path'
import { randomBytes, createHash } from 'node:crypto'
import { spawn, spawnSync } from 'node:child_process'
import { once } from 'node:events'
import { freePort } from './ingress.mjs'

export async function managedAccounts({ browser, root, temp, evidence }) {
  const dir = join(temp, 'managed-accounts'); await mkdir(dir)
  const port = await freePort(), origin = `http://127.0.0.1:${port}`
  const token = randomBytes(32).toString('hex'), directoryKey = randomBytes(32).toString('hex')
  let password = randomBytes(24).toString('hex')
  const credentials = join(dir, 'keys.json'), config = join(dir, 'runtime.yaml')
  await writeFile(credentials, JSON.stringify({ principals: [{ id: 'principal:recovery', sha256: createHash('sha256').update(token).digest('hex'), roles: ['role:admin'] }, { id: 'principal:directory', sha256: createHash('sha256').update(directoryKey).digest('hex'), roles: ['role:directory'] }] }), { mode: 0o600 })
  await writeFile(join(dir, 'agents.json'), '{"credentials":[]}', { mode: 0o600 })
  await writeFile(config, `http: {listen: "127.0.0.1:${port}"}
database: ${JSON.stringify(join(dir, 'state/db.sqlite'))}
credentials: ${JSON.stringify(join(dir, 'agents.json'))}
source: {mode: database}
management: {credentials: ${JSON.stringify(credentials)}, users_mode: database, reload: 100ms}
agent: {enabled: true, listen: "127.0.0.1:0", authority: "accounts:test"}
browser: {enabled: true, origin: ${JSON.stringify(origin)}, assets: ${JSON.stringify(join(root, 'apps/web/dist'))}}
`, { mode: 0o600 })
  let logs = ''
  const env = Object.fromEntries(Object.entries(process.env).filter(([k]) => !k.startsWith('PICKETX_')))
  const child = spawn(join(temp, 'controller'), ['-config', config], { cwd: root, env, stdio: ['ignore', 'pipe', 'pipe'] })
  child.stdout.on('data', b => { logs += b }); child.stderr.on('data', b => { logs += b })
  const context = await browser.newContext(), human = await browser.newContext()
  const page = await context.newPage(), person = await human.newPage()
  try {
    let ready = false
    for (let n = 0; n < 100; n++) {
      assert.equal(child.exitCode, null, logs)
      try { if ((await fetch(`${origin}/readyz`)).ok) { ready = true; break } } catch { /* process startup */ }
      await new Promise(r => setTimeout(r, 100))
    }
    assert(ready, logs)
    await page.goto(`${origin}/console/`)
    await page.getByLabel('Access key', { exact: true }).fill(token)
    await page.getByRole('button', { name: 'Sign in', exact: true }).click()
    await page.getByRole('button', { name: 'Accounts', exact: true }).click({ timeout: 5000 })
    await page.getByText('No human accounts yet.', { exact: true }).waitFor()
    await page.getByLabel('Account username', { exact: true }).fill('alice')
    await page.getByLabel('Account password', { exact: true }).fill(password)
    await page.getByLabel('Account change reason', { exact: true }).fill('onboard Alice')
    await page.getByRole('button', { name: 'Create account', exact: true }).click()
    await page.getByText('Account saved. Previous sessions are invalid.', { exact: true }).waitFor()
    const row = page.locator('.account-row').filter({ has: page.getByRole('heading', { name: 'alice', exact: true }) })
    await row.waitFor()
    assert.equal(await page.getByLabel('Account password', { exact: true }).inputValue(), '')
    const directory = await page.evaluate(async () => (await fetch('/console/api/v1/accounts')).json())
    assert.equal(directory.revision, 1); assert.equal(directory.accounts[0].version, 1)
    assert(!JSON.stringify(directory).includes(password)); assert(!JSON.stringify(directory).includes('$2'))
    const id = directory.accounts[0].id
    async function signInHuman() {
      await person.goto(`${origin}/console/`)
      await person.getByLabel('Sign-in method', { exact: true }).selectOption('password')
      await person.getByLabel('Username', { exact: true }).fill('alice')
      await person.getByLabel('Password', { exact: true }).fill(password)
      await person.getByRole('button', { name: 'Sign in', exact: true }).click()
      await person.getByRole('heading', { name: 'Access requests', exact: true }).waitFor()
    }
    await signInHuman()
    const oldCookie = await human.cookies()
    await row.getByRole('button', { name: 'Edit account', exact: true }).click()
    await page.getByLabel('Account disabled', { exact: true }).check()
    await page.getByLabel('Account change reason', { exact: true }).fill('disable Alice')
    await page.getByLabel('Invalidate previous sessions', { exact: true }).check()
    await page.getByRole('button', { name: 'Save account', exact: true }).click()
    await row.getByText('Disabled', { exact: true }).waitFor()
    await row.getByRole('button', { name: 'Edit account', exact: true }).click()
    await page.getByLabel('Account disabled', { exact: true }).uncheck()
    await page.getByLabel('Account change reason', { exact: true }).fill('re-enable Alice')
    await page.getByLabel('Invalidate previous sessions', { exact: true }).check()
    await page.getByRole('button', { name: 'Save account', exact: true }).click()
    await row.getByText('Enabled', { exact: true }).waitFor()
    await human.addCookies(oldCookie)
    assert.equal((await human.request.get(`${origin}/console/api/v1/me`)).status(), 401)
    await signInHuman()
    assert.equal(await person.getByRole('button', { name: 'Accounts', exact: true }).count(), 0)
    assert.equal((await human.request.get(`${origin}/console/api/v1/accounts`)).status(), 403)
    // Refresh must not silently rebase an in-progress editor after an external write.
    await row.getByRole('button', { name: 'Edit account', exact: true }).click()
    await page.getByLabel('Account change reason', { exact: true }).fill('preserved stale draft')
    await page.getByLabel('Invalidate previous sessions', { exact: true }).check()
    const external = await fetch(`${origin}/api/v1/accounts/${id}`, { method: 'PUT', headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json', 'If-Match': '"3"' }, body: JSON.stringify({ roles: ['role:requester'], disabled: false, reason: 'other operator' }) })
    assert.equal(external.status, 200)
    await page.getByRole('button', { name: 'Refresh accounts', exact: true }).click()
    await page.getByRole('button', { name: 'Save account', exact: true }).click()
    await page.getByText('Inspect the latest directory, then clear the draft before retrying.', { exact: true }).waitFor()
    assert.equal(await page.getByLabel('Account change reason', { exact: true }).inputValue(), 'preserved stale draft')
    assert(await page.getByRole('button', { name: 'Save account', exact: true }).isDisabled())
    for (const width of [1440, 375]) {
      await page.setViewportSize({ width, height: 1000 })
      assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
      await page.screenshot({ path: join(evidence, `accounts-conflict-${width}.png`), fullPage: true })
    }
    page.once('dialog', dialog => dialog.accept())
    await page.getByRole('button', { name: 'Clear account draft', exact: true }).click()
    await page.getByLabel('Account change reason', { exact: true }).fill('')
    await signInHuman() // Fresh generation4 before reset, not an already stale cookie.
    await row.getByRole('button', { name: 'Edit account', exact: true }).click()
    const previousPassword = password; password = randomBytes(24).toString('hex')
    await page.getByLabel('Account password', { exact: true }).fill(password)
    await page.getByLabel('Account change reason', { exact: true }).fill('reset Alice')
    await page.getByLabel('Invalidate previous sessions', { exact: true }).check()
    await page.getByRole('button', { name: 'Save account', exact: true }).click()
    await page.getByText('Session generation 5', { exact: true }).waitFor()
    assert.equal((await human.request.get(`${origin}/console/api/v1/me`)).status(), 401)
    const bootstrap = await (await human.request.get(`${origin}/console/api/v1/session`)).json()
    const oldLogin = await human.request.post(`${origin}/console/api/v1/login/password`, { headers: { Origin: origin, 'X-CSRF-Token': bootstrap.csrf }, data: { username: 'alice', password: previousPassword } })
    assert.equal(oldLogin.status(), 401)
    await signInHuman()
    const afterReset = await page.evaluate(async () => (await fetch('/console/api/v1/accounts')).json())
    assert.equal(afterReset.revision, 5); assert.equal(afterReset.events.length, 5)
    assert.equal(afterReset.events[0].reason, 'reset Alice'); assert.equal(afterReset.events[0].actor, 'principal:recovery')
    await page.getByLabel('Account username', { exact: true }).fill('read-draft')
    await page.route('**/console/api/v1/accounts', r => r.request().method() === 'GET' ? r.abort() : r.continue())
    await page.getByRole('button', { name: 'Refresh accounts', exact: true }).click()
    await page.getByText('Last read failed. Displayed data may be stale.', { exact: true }).waitFor()
    assert.equal(await page.getByLabel('Account username', { exact: true }).inputValue(), 'read-draft')
    await page.unroute('**/console/api/v1/accounts')
    page.once('dialog', dialog => dialog.accept()); await page.getByRole('button', { name: 'Clear account draft', exact: true }).click()
    for (const fault of [202, 500, 'abort']) {
      await page.getByLabel('Account username', { exact: true }).fill('uncertain')
      await page.getByLabel('Account password', { exact: true }).fill(password)
      await page.getByLabel('Account change reason', { exact: true }).fill('unknown outcome')
      await page.route('**/console/api/v1/accounts', r => r.request().method() !== 'POST' ? r.continue() : fault === 'abort' ? r.abort() : r.fulfill({ status: fault, contentType: 'application/json', body: '{}' }))
      await page.getByRole('button', { name: 'Create account', exact: true }).click()
      await page.locator('.accounts-section [role="alert"]').waitFor()
      assert(await page.getByRole('button', { name: 'Create account', exact: true }).isDisabled())
      assert.equal(await page.getByLabel('Account password', { exact: true }).inputValue(), '')
      assert.equal(await page.getByLabel('Account change reason', { exact: true }).inputValue(), 'unknown outcome')
      assert.equal(await page.locator('.accounts-section .feedback.success').count(), 0)
      await page.unroute('**/console/api/v1/accounts')
      assert.equal((await page.evaluate(async () => (await fetch('/console/api/v1/accounts')).json())).revision, 5)
      page.once('dialog', dialog => dialog.accept()); await page.getByRole('button', { name: 'Clear account draft', exact: true }).click()
    }
    // Writer survives navigation; no client cancellation is mistaken for rollback.
    let release; const held = new Promise(r => { release = r }); let intercepted
    const started = new Promise(r => { intercepted = r })
    let releaseOldRead, capturedOldRead, holdNextRead = false
    const oldRead = new Promise(r => { releaseOldRead = r }), oldReadCaptured = new Promise(r => { capturedOldRead = r })
    await page.route('**/console/api/v1/accounts', async r => {
      if (r.request().method() === 'POST') { intercepted(); await held; return r.continue() }
      if (holdNextRead && r.request().method() === 'GET') {
        holdNextRead = false; const actual = await r.fetch(); capturedOldRead(); await oldRead; return r.fulfill({ response: actual })
      }
      return r.continue()
    })
    await page.getByLabel('Account username', { exact: true }).fill('bob')
    await page.getByLabel('Account password', { exact: true }).fill(password)
    await page.getByLabel('Account change reason', { exact: true }).fill('writer navigation')
    await page.getByRole('button', { name: 'Create account', exact: true }).click(); await started
    await page.getByRole('button', { name: 'Desired configuration', exact: true }).click()
    assert(await page.getByRole('button', { name: 'Sign out', exact: true }).isDisabled())
    holdNextRead = true
    await page.getByRole('button', { name: 'Accounts', exact: true }).click()
    await oldReadCaptured // Real pre-write directory5 held across committed directory6.
    release()
    await page.locator('.account-row').getByRole('heading', { name: 'bob', exact: true }).waitFor()
    releaseOldRead()
    await page.waitForFunction(() => document.querySelector('.accounts-section')?.getAttribute('aria-busy') === 'false')
    assert(await page.locator('.account-row').getByRole('heading', { name: 'bob', exact: true }).isVisible(), 'late pre-write GET must not replace committed account state')
    await page.unroute('**/console/api/v1/accounts')
    // Real directory-only capability without publication or request privileges.
    const policy = spawnSync('python3', ['-c', 'import sqlite3,sys\nc=sqlite3.connect(sys.argv[1])\nfor op in ("account:read","account:write"): c.execute("INSERT INTO casbin_rule(ptype,v0,v1,v2,v3,v4,v5) VALUES(?,?,?,?,?,?,?)",("p","role:directory",op,"allow","","",""))\nc.commit()', join(dir, 'state/db.sqlite')], { encoding: 'utf8' })
    assert.equal(policy.status, 0, policy.stderr)
    for (let n = 0; n < 100; n++) {
      const me = await fetch(`${origin}/api/v1/me`, { headers: { Authorization: `Bearer ${directoryKey}` } })
      if ((await me.json()).operations.includes('account:read')) break
      assert(n < 99); await new Promise(r => setTimeout(r, 50))
    }
    const only = await browser.newContext(), onlyPage = await only.newPage()
    try {
      const requested = []; onlyPage.on('request', r => requested.push(new URL(r.url()).pathname))
      await onlyPage.goto(`${origin}/console/`)
      await onlyPage.getByLabel('Access key', { exact: true }).fill(directoryKey)
      await onlyPage.getByRole('button', { name: 'Sign in', exact: true }).click()
      await onlyPage.getByRole('heading', { name: 'Accounts', exact: true }).waitFor()
      await onlyPage.getByRole('heading', { name: 'alice', exact: true }).waitFor()
      assert(!requested.some(p => p.endsWith('/publication') || p.includes('/access-request')))
      assert.equal(await onlyPage.getByRole('button', { name: 'Access requests', exact: true }).count(), 0)
      await onlyPage.getByRole('button', { name: 'Language', exact: true }).click()
      await onlyPage.getByRole('heading', { name: '人员账户目录', exact: true }).waitFor()
      for (const width of [375, 1440]) {
        await onlyPage.setViewportSize({ width, height: 1000 }); assert(await onlyPage.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
        await onlyPage.screenshot({ path: join(evidence, `accounts-zh-${width}.png`), fullPage: true })
      }
    } finally { await only.close() }
    assert(!logs.includes(password)); assert(!logs.includes('$2'))
    console.log('PASS actual managed accounts: browser create, audit/CAS, disable/re-enable permanent revocation, reset password, requester denial, stale draft/read failure, 202/500/transport unknown outcomes, held writer navigation, directory-only capability, EN/ZH 375/1440 layout')
  } finally {
    await page.screenshot({ path: join(evidence, 'accounts-last.png'), fullPage: true }).catch(() => {})
    await context.close(); await human.close()
    if (child.exitCode === null) { const exited = once(child, 'exit'); child.kill('SIGTERM'); await exited }
    assert.equal(child.exitCode, 0, logs)
  }
}
