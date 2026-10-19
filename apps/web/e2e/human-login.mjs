// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict'
import { writeFile } from 'node:fs/promises'
import { join } from 'node:path'
import { spawnSync } from 'node:child_process'

export async function humanLogin({ browser, origin, principalPath, root, evidence }) {
  const password = 'browser human password 1!', rotated = 'browser rotated password 2!'
  const hash = secret => {
    const r = spawnSync('go', ['run', './cmd/picketx', 'password-hash'], { cwd: root, input: `${secret}\n`, encoding: 'utf8' })
    assert.equal(r.status, 0, r.stderr); assert.match(r.stdout.trim(), /^\$2a\$12\$/)
    return r.stdout.trim()
  }
  let passwordHash = hash(password)
  const provision = async (roles, disabled = false) => writeFile(principalPath, JSON.stringify({ principals: [], users: [{ id: 'principal:alice', username: 'alice', password_hash: passwordHash, roles, disabled }] }), { mode: 0o600 })
  await provision(['role:admin'])
  const context = await browser.newContext(), page = await context.newPage()
  const errors = []; page.on('pageerror', e => errors.push(e.message))
  await page.goto(`${origin}/console/`)
  await page.getByLabel('Sign-in method', { exact: true }).selectOption('password', { timeout: 5000 })
  assert.equal(await page.getByLabel('Username', { exact: true }).getAttribute('autocomplete'), 'username')
  assert.equal(await page.getByLabel('Password', { exact: true }).getAttribute('autocomplete'), 'current-password')
  await page.getByLabel('Username', { exact: true }).fill('alice')
  await page.getByLabel('Password', { exact: true }).fill('wrong human password')
  await page.getByRole('button', { name: 'Sign in', exact: true }).click()
  await page.getByRole('alert').getByText('Sign-in details not accepted. Check them or contact your administrator.', { exact: true }).waitFor()
  assert.equal(await page.getByLabel('Password', { exact: true }).inputValue(), '')
  await page.getByLabel('Password', { exact: true }).fill(password)
  for (const width of [375, 1440]) {
    await page.setViewportSize({ width, height: 1000 })
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
    await page.screenshot({ path: join(evidence, `human-login-${width}.png`), fullPage: true })
  }
  await page.getByRole('button', { name: 'Sign in', exact: true }).click()
  await page.getByRole('heading', { name: 'Desired configuration', exact: true }).waitFor()
  await page.getByText('principal:alice', { exact: true }).waitFor()
  const cookies = await context.cookies(), cookie = cookies.find(c => c.name === 'picketx_session')
  assert(cookie.httpOnly && cookie.sameSite === 'Strict')
  assert(!cookies.some(c => c.value.includes(password) || c.value.includes(passwordHash)))
  assert.equal(await page.evaluate(() => localStorage.length + sessionStorage.length), 0)
  const prior = await page.evaluate(async () => (await fetch('/console/api/v1/publication')).json())
  await page.getByLabel('Policy YAML', { exact: true }).fill(`${prior.policy}\n# human identity publication\n`)
  await page.getByLabel('Change reason', { exact: true }).fill('Human login actual publication')
  await page.getByRole('button', { name: 'Save desired configuration', exact: true }).click()
  await page.getByText('Desired configuration saved. Node application is not confirmed.', { exact: true }).waitFor()
  const recorded = await page.evaluate(async () => {
    const publication = await (await fetch('/console/api/v1/publication')).json()
    const events = []; let after = 0, part
    do {
      part = await (await fetch(`/console/api/v1/audit?limit=100&after=${after}`)).json()
      events.push(...part.events); after = part.next_after
    } while (part.events.length === 100)
    return { publication, audit: { events } }
  })
  assert.equal(recorded.publication.revision, prior.revision + 1)
  assert(recorded.audit.events.some(e => e.actor === 'principal:alice' && e.reason === 'Human login actual publication'))
  await provision(['role:viewer'])
  await page.getByRole('button', { name: 'Refresh status', exact: true }).click()
  await page.getByText('Read-only access. Your current role cannot publish changes.', { exact: true }).waitFor()
  assert.equal(await page.evaluate(async () => (await fetch('/console/api/v1/publication', { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: '{}' })).status), 403)
  await provision(['role:admin'])
  await page.getByRole('button', { name: 'Sign out', exact: true }).click()
  await page.getByLabel('Access key', { exact: true }).waitFor()
  // Logout destroys the old cookie server-side, not just in the React state.
  assert.equal((await fetch(`${origin}/console/api/v1/me`, { headers: { Cookie: `picketx_session=${cookie.value}` } })).status, 401)
  await page.getByLabel('Sign-in method', { exact: true }).selectOption('password')
  await page.getByLabel('Username', { exact: true }).fill('alice')
  await page.getByLabel('Password', { exact: true }).fill(password)
  await page.getByRole('button', { name: 'Sign in', exact: true }).click()
  await page.getByRole('heading', { name: 'Desired configuration', exact: true }).waitFor()
  passwordHash = hash(rotated); await provision(['role:admin'])
  await page.getByRole('button', { name: 'Refresh status', exact: true }).click()
  await page.getByLabel('Sign-in method', { exact: true }).waitFor()
  await page.getByRole('alert').getByText('Your session expired or access was revoked. Sign in again.', { exact: true }).waitFor()
  await page.getByLabel('Sign-in method', { exact: true }).selectOption('password')
  await page.getByLabel('Username', { exact: true }).fill('alice')
  await page.getByLabel('Password', { exact: true }).fill(rotated)
  await page.getByRole('button', { name: 'Sign in', exact: true }).click()
  await page.getByRole('heading', { name: 'Desired configuration', exact: true }).waitFor()
  await provision(['role:admin'], true)
  await page.getByRole('button', { name: 'Refresh status', exact: true }).click()
  await page.getByLabel('Access key', { exact: true }).waitFor()
  assert.equal(await page.getByLabel('Sign-in method', { exact: true }).count(), 0)
  assert.deepEqual(errors, [])
  await context.close()
  console.log('PASS actual human password → SCS/Casbin → publication/SQLite/audit; viewer denial, logout, password rotation, disable, 375/1440 layout')
}
