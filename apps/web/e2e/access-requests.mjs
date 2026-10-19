// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict'
import { readFile, writeFile } from 'node:fs/promises'
import { join } from 'node:path'
import { spawnSync } from 'node:child_process'

export async function accessRequests({ browser, origin, principalPath, root, evidence, databasePath }) {
 const password = 'independent request password!'
 const result = spawnSync('go', ['run', './cmd/picketx', 'password-hash'], { cwd: root, input: password, encoding: 'utf8' })
 assert.equal(result.status, 0, result.stderr)
 await writeFile(principalPath, JSON.stringify({ principals: [], users: [
  { id: 'principal:requester', username: 'requester', password_hash: result.stdout.trim(), roles: ['role:requester'] },
  { id: 'principal:approver', username: 'approver', password_hash: result.stdout.trim(), roles: ['role:admin'] },
  { id: 'principal:creator', username: 'creator', password_hash: result.stdout.trim(), roles: ['role:create-only'] },
  { id: 'principal:reviewer', username: 'reviewer', password_hash: result.stdout.trim(), roles: ['role:approve-only'] },
 ] }), { mode: 0o600 })
 // Explicit fixture provisioning through the real SQLite Casbin adapter. Wait
 // for configured policy reload below; do not mock current-user operations.
 const policies = spawnSync('python3', ['-c', `import sqlite3,sys
c=sqlite3.connect(sys.argv[1])
for role,op in [('role:create-only','request:create'),('role:approve-only','request:review'),('role:approve-only','publication:write')]:
 c.execute("INSERT INTO casbin_rule(ptype,v0,v1,v2,v3,v4,v5) VALUES('p',?,?,'allow','','','')",(role,op))
c.commit()`, databasePath], { encoding: 'utf8' })
 assert.equal(policies.status, 0, policies.stderr)
 const contexts = await Promise.all([browser.newContext(), browser.newContext()])
 const [requester, admin] = await Promise.all(contexts.map(c => c.newPage()))
 const errors = []; for (const page of [requester, admin]) page.on('pageerror', e => errors.push(e.message))
 const login = async (page, user) => {
  await page.goto(`${origin}/console/`)
  await page.getByLabel('Sign-in method', { exact: true }).selectOption('password')
  for (let attempt = 0; attempt < 3; attempt++) {
   await page.getByLabel('Username', { exact: true }).fill(user)
   await page.getByLabel('Password', { exact: true }).fill(password)
   const response = page.waitForResponse(r => r.url().endsWith('/console/api/v1/login/password') && r.request().method() === 'POST')
   await page.getByRole('button', { name: 'Sign in', exact: true }).click()
   const result = await response
   if (result.status() === 200) return
   assert.equal(result.status(), 429, `real ${user} login: ${result.status()} ${await result.text()}`)
   // The preceding human-login suite legitimately consumes this same local
   // peer's 5-token burst. Honor the production 0.1/s refill, never reset/bypass it.
   console.log(`Observed real password-login429 for ${user}; waiting for configured refill`)
   await page.getByRole('alert').waitFor()
   await new Promise(resolve => setTimeout(resolve, 10100))
  }
  assert.fail(`password-login rate did not refill for ${user}`)
 }
 await login(admin, 'approver')
 await admin.getByLabel('Policy YAML').waitFor()
 // Use actual existing pinned Resource, independent of preceding suites' inventory.
 const policy = await readFile(join(root, 'examples/static/gitea.yaml'), 'utf8')
 const facts = await readFile(join(root, 'examples/static/facts.json'), 'utf8')
 await admin.getByLabel('Policy YAML').fill(policy)
 await admin.getByLabel('Facts JSON').fill(facts)
 await admin.getByLabel('Change reason', { exact: true }).fill('request workflow inventory')
 await admin.getByRole('button', { name: 'Save desired configuration', exact: true }).click()
 await admin.getByText('Desired configuration saved. Node application is not confirmed.', { exact: true }).waitFor()
 // Hold the real initial read at its transport boundary. Leaving this page must
 // cancel that read and release UI busy state, not disable logout indefinitely.
 let releaseRead, readStarted
 const held = new Promise(resolve => { readStarted = resolve })
 await admin.route('**/console/api/v1/access-requests?limit=50', async route => {
  readStarted(); await new Promise(resolve => { releaseRead = resolve }); await route.continue().catch(() => {})
 })
 await admin.getByRole('button', { name: 'Access requests', exact: true }).click()
 await held
 await admin.getByRole('button', { name: 'Desired configuration', exact: true }).click()
 try { await admin.waitForFunction(() => ![...document.querySelectorAll('button')].find(b => b.textContent === 'Sign out')?.disabled, null, { timeout: 5000 }); assert.equal(await admin.getByRole('button', { name: 'Sign out', exact: true }).isDisabled(), false, 'leaving cancelled request read releases busy state') }
 finally { releaseRead(); await admin.unroute('**/console/api/v1/access-requests?limit=50') }
 await login(requester, 'requester')
 await requester.getByRole('heading', { name: 'Access requests', exact: true }).waitFor({ timeout: 5000 })
 assert.equal(await requester.getByLabel('Policy YAML').count(), 0)
 assert.equal(await requester.evaluate(async () => (await fetch('/console/api/v1/publication')).status), 403)
 await requester.getByLabel('Requested source IP or CIDR', { exact: true }).fill('2001:db8::42')
 await requester.getByLabel('Requested Resource', { exact: true }).selectOption('resource-gitea-v1')
 await requester.getByRole('checkbox', { name: /^https / }).check()
 await requester.getByRole('checkbox', { name: /^ssh / }).check()
 await requester.getByLabel('Request reason', { exact: true }).fill('Browser independent request')
 await requester.getByRole('button', { name: 'Send access request', exact: true }).click()
 await requester.getByText('Request saved. No access has been granted.', { exact: true }).waitFor()
 const requestRow = requester.locator('.request-row').filter({ hasText: 'Browser independent request' })
 await requestRow.getByText('Pending', { exact: true }).waitFor()
 const original = await requester.evaluate(async () => (await (await fetch('/console/api/v1/access-requests')).json()).requests.find(r => r.reason === 'Browser independent request'))
 assert.equal(original.requester, 'principal:requester'); assert.equal(original.source, '2001:db8::42/128')
 await admin.getByRole('button', { name: 'Access requests', exact: true }).click()
 const review = admin.locator('.request-row').filter({ hasText: 'Browser independent request' })
 await review.getByRole('button', { name: 'Review request', exact: true }).click()
 await admin.getByRole('checkbox', { name: 'Approve ssh', exact: true }).uncheck()
 await admin.getByLabel('Decision reason', { exact: true }).fill('Browser narrowed approval')
 await admin.getByLabel('Review confirmed', { exact: true }).check()
 await admin.getByRole('button', { name: 'Approve and publish Grant', exact: true }).click()
 await admin.getByText('Decision saved. Approval is not confirmed application.', { exact: true }).waitFor()
 await requester.getByRole('button', { name: 'Refresh requests', exact: true }).click()
 await requestRow.getByText('Approved', { exact: true }).waitFor()
 const approved = await requester.evaluate(async () => (await (await fetch('/console/api/v1/access-requests')).json()).requests.find(r => r.reason === 'Browser independent request'))
 assert.deepEqual(approved.endpoint_names, ['https', 'ssh']); assert.deepEqual(approved.approved_endpoints, ['https'])
 assert.equal(approved.approver, 'principal:approver'); assert.equal(approved.expires_at, original.expires_at)
 assert(new Date(approved.approved_expires_at) <= new Date(original.expires_at)); assert(approved.grant_uid && approved.revision)
 const proof = await admin.evaluate(async () => ({ grants: await (await fetch('/console/api/v1/grants?limit=100')).json(), audit: await (await fetch('/console/api/v1/audit?limit=100')).json() }))
 const grant = proof.grants.grants.find(g => g.uid === approved.grant_uid)
 assert.equal(grant.source, '2001:db8::42/128'); assert.deepEqual(grant.scopes[0].endpoint_names, ['https'])
 assert.equal(grant.expires_at, approved.approved_expires_at)
 assert(proof.audit.events.some(e => e.actor === 'principal:approver' && e.reason.includes(approved.uid)))
 for (const width of [375, 1440]) { await requester.setViewportSize({ width, height: 1000 }); assert(await requester.evaluate(() => document.documentElement.scrollWidth <= innerWidth)); await requester.screenshot({ path: join(evidence, `access-requests-${width}.png`), fullPage: true }) }
 // A separate real request is rejected; no publication revision is consumed.
 // Plain refresh intentionally preserves its old CAS base after the approval.
 await requester.getByRole('button', { name: 'Use latest request catalog', exact: true }).click()
 await requester.getByLabel('Requested source IP or CIDR', { exact: true }).fill('192.0.2.42')
 await requester.getByLabel('Requested Resource', { exact: true }).selectOption('resource-gitea-v1')
 await requester.getByRole('checkbox', { name: /^https / }).check()
 await requester.getByLabel('Request reason', { exact: true }).fill('Browser rejected request')
 await requester.getByRole('button', { name: 'Send access request', exact: true }).click()
 await requester.getByText('Request saved. No access has been granted.', { exact: true }).waitFor()
 await admin.getByRole('button', { name: 'Refresh requests', exact: true }).click()
 await admin.locator('.request-row').filter({ hasText: 'Browser rejected request' }).getByRole('button', { name: 'Review request', exact: true }).click()
 await admin.getByLabel('Decision reason', { exact: true }).fill('Browser rejection')
 await admin.getByRole('button', { name: 'Reject request', exact: true }).click()
 await admin.getByText('Decision saved. Approval is not confirmed application.', { exact: true }).waitFor()
 await requester.getByRole('button', { name: 'Refresh requests', exact: true }).click()
 await requester.locator('.request-row').filter({ hasText: 'Browser rejected request' }).getByText('Rejected', { exact: true }).waitFor()
 assert.equal((await admin.evaluate(async () => (await (await fetch('/console/api/v1/publication')).json()).revision)), approved.revision)
 const revoke = await admin.evaluate(async r => {
  const session = await (await fetch('/console/api/v1/session')).json()
  return (await fetch(`/console/api/v1/grants/${r.grant_uid}/revoke`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': session.csrf, 'If-Match': `"${r.revision}"` }, body: JSON.stringify({ reason: 'Revoke approved request contribution' }) })).status
 }, approved)
 assert.equal(revoke, 200)
 await requester.getByRole('button', { name: 'Refresh requests', exact: true }).click()
 await requestRow.getByText('Desired Grant: revoked', { exact: true }).waitFor()
 await admin.getByRole('button', { name: 'Use latest request catalog', exact: true }).click()
 // A new read after navigating back must not unlock an in-flight POST. Hold
 // only transport, then let the actual Controller persist exactly one request.
 await admin.getByLabel('Requested source IP or CIDR', { exact: true }).fill('192.0.2.88')
 await admin.getByLabel('Requested Resource', { exact: true }).selectOption('resource-gitea-v1')
 await admin.getByRole('checkbox', { name: /^https / }).check()
 await admin.getByLabel('Request reason', { exact: true }).fill('Held write navigation regression')
 let releaseWrite, writeStarted, finishWrite, writes = 0
 const writing = new Promise(resolve => { writeStarted = resolve })
 const continued = new Promise(resolve => { finishWrite = resolve })
 await admin.route('**/console/api/v1/access-requests', async route => {
  if (route.request().method() !== 'POST') return route.continue()
  writes++; writeStarted()
  try { await new Promise(resolve => { releaseWrite = resolve }); await route.continue() }
  finally { finishWrite() }
 })
 await admin.getByRole('button', { name: 'Send access request', exact: true }).click()
 await writing
 try {
  await admin.getByRole('button', { name: 'Desired configuration', exact: true }).click()
  const reread = admin.waitForResponse(r => r.url().endsWith('/access-requests?limit=50') && r.request().method() === 'GET')
  await admin.getByRole('button', { name: 'Access requests', exact: true }).click()
  await (await reread).finished()
  await admin.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))))
  assert.equal(await admin.getByRole('button', { name: 'Send access request', exact: true }).isDisabled(), true, 'completed GET cannot unlock pending POST')
  assert.equal(await admin.getByRole('button', { name: 'Sign out', exact: true }).isDisabled(), true)
 } finally { releaseWrite(); await continued; await admin.unroute('**/console/api/v1/access-requests') }
 await admin.getByText('Request saved. No access has been granted.', { exact: true }).waitFor()
 await admin.waitForFunction(() => ![...document.querySelectorAll('button')].find(b => b.textContent === 'Sign out')?.disabled)
 const persisted = await admin.evaluate(async () => (await (await fetch('/console/api/v1/access-requests')).json()).requests.filter(r => r.reason === 'Held write navigation regression'))
 assert.equal(writes, 1); assert.equal(persisted.length, 1)
 // Independent custom capabilities: creation does not depend on history-read;
 // review/publication-write does not depend on creation or publication-read.
 const creatorContext = await browser.newContext(), creator = await creatorContext.newPage()
 const reviewerContext = await browser.newContext(), reviewer = await reviewerContext.newPage()
 for (const p of [creator, reviewer]) p.on('pageerror', e => errors.push(e.message))
 await login(creator, 'creator')
 await creator.waitForFunction(async () => (await (await fetch('/console/api/v1/me')).json()).operations?.includes('request:create'), null, { timeout: 45000 })
 await creator.reload()
 await creator.getByRole('heading', { name: 'Access requests', exact: true }).waitFor()
 assert.equal(await creator.getByLabel('Policy YAML').count(), 0)
 assert.equal(await creator.evaluate(async () => (await fetch('/console/api/v1/access-requests')).status), 403)
 await creator.getByLabel('Requested source IP or CIDR', { exact: true }).fill('192.0.2.66')
 await creator.getByLabel('Requested Resource', { exact: true }).selectOption('resource-gitea-v1')
 await creator.getByRole('checkbox', { name: /^https / }).check()
 await creator.getByLabel('Request reason', { exact: true }).fill('Create-only to approve-only capability regression')
 await creator.getByRole('button', { name: 'Send access request', exact: true }).click()
 await creator.getByText('Request saved. No access has been granted.', { exact: true }).waitFor()
 await login(reviewer, 'reviewer')
 await reviewer.getByRole('heading', { name: 'Access requests', exact: true }).waitFor()
 assert.equal(await reviewer.getByLabel('Policy YAML').count(), 0)
 assert.equal(await reviewer.getByLabel('Requested source IP or CIDR', { exact: true }).count(), 0)
 await reviewer.locator('.request-row').filter({ hasText: 'Create-only to approve-only capability regression' }).getByRole('button', { name: 'Review request', exact: true }).click()
 await reviewer.getByLabel('Decision reason', { exact: true }).fill('Independent custom capability approval')
 await reviewer.getByLabel('Review confirmed', { exact: true }).check()
 await reviewer.getByRole('button', { name: 'Approve and publish Grant', exact: true }).click()
 await reviewer.getByText('Decision saved. Approval is not confirmed application.', { exact: true }).waitFor()
 await reviewer.locator('.request-row').filter({ hasText: 'Create-only to approve-only capability regression' }).getByText('Approved', { exact: true }).waitFor()
 await creatorContext.close(); await reviewerContext.close()
 assert.deepEqual(errors, [])
 for (const c of contexts) await c.close()
 console.log('PASS real human requester → independent approval/rejection → atomic Grant/SQLite/audit; IPv6 named narrowing, absolute deadline, requester isolation and revoke')
}
