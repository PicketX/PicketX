// SPDX-License-Identifier: Apache-2.0
// A separate real process isolates the sticky DB-user authority from file-user suites.
import assert from 'node:assert/strict'
import { mkdir, writeFile } from 'node:fs/promises'
import { join } from 'node:path'
import { randomBytes, createHash } from 'node:crypto'
import { spawn } from 'node:child_process'
import { once } from 'node:events'
import { freePort } from './ingress.mjs'

export async function managedAccounts({ browser, root, temp, evidence }) {
  const dir = join(temp, 'managed-accounts'); await mkdir(dir)
  const port = await freePort(), origin = `http://127.0.0.1:${port}`
  const token = randomBytes(32).toString('hex'), password = randomBytes(24).toString('hex')
  const credentials = join(dir, 'keys.json'), config = join(dir, 'runtime.yaml')
  await writeFile(credentials, JSON.stringify({ principals: [{ id: 'principal:recovery', sha256: createHash('sha256').update(token).digest('hex'), roles: ['role:admin'] }] }), { mode: 0o600 })
  await writeFile(join(dir, 'agents.json'), '{"credentials":[]}', { mode: 0o600 })
  await writeFile(config, `http: {listen: "127.0.0.1:${port}"}
database: ${JSON.stringify(join(dir, 'state/db.sqlite'))}
credentials: ${JSON.stringify(join(dir, 'agents.json'))}
source: {mode: database}
management: {credentials: ${JSON.stringify(credentials)}, users_mode: database}
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
    assert(!logs.includes(password)); assert(!logs.includes('$2'))
    console.log('PASS actual managed accounts: browser create, atomic audit/CAS, disable/re-enable permanent session invalidation, requester denial, stale draft, secret redaction, 375/1440 layout')
  } finally {
    await page.screenshot({ path: join(evidence, 'accounts-last.png'), fullPage: true }).catch(() => {})
    await context.close(); await human.close()
    if (child.exitCode === null) { const exited = once(child, 'exit'); child.kill('SIGTERM'); await exited }
    assert.equal(child.exitCode, 0, logs)
  }
}
