// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict'
import { readFile, writeFile } from 'node:fs/promises'
import { join } from 'node:path'
import { spawnSync } from 'node:child_process'

export async function accessRequests({ browser, origin, principalPath, root, evidence }) {
 const password = 'independent request password!'
 const result = spawnSync('go', ['run', './cmd/picketx', 'password-hash'], { cwd: root, input: password, encoding: 'utf8' })
 assert.equal(result.status, 0, result.stderr)
 await writeFile(principalPath, JSON.stringify({ principals: [], users: [
  { id: 'person:requester', username: 'requester', password_hash: result.stdout.trim(), roles: ['role:requester'] },
  { id: 'person:approver', username: 'approver', password_hash: result.stdout.trim(), roles: ['role:admin'] },
 ] }), { mode: 0o600 })
 const contexts = await Promise.all([browser.newContext(), browser.newContext()])
 const [requester, admin] = await Promise.all(contexts.map(c => c.newPage()))
 const errors = []; for (const page of [requester, admin]) page.on('pageerror', e => errors.push(e.message))
 const login = async (page, user) => {
  await page.goto(`${origin}/console/`)
  await page.getByLabel('Sign-in method', { exact: true }).selectOption('password')
  await page.getByLabel('Username', { exact: true }).fill(user)
  await page.getByLabel('Password', { exact: true }).fill(password)
  await page.getByRole('button', { name: 'Sign in', exact: true }).click()
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
 await login(requester, 'requester')
 await requester.getByRole('heading', { name: 'Access requests', exact: true }).waitFor({ timeout: 5000 })
 assert.equal(await requester.getByLabel('Policy YAML').count(), 0)
 assert.equal(await requester.evaluate(async () => (await fetch('/console/api/v1/publication')).status), 403)
 await requester.getByLabel('Source IP or CIDR', { exact: true }).fill('2001:db8::42')
 await requester.getByLabel('Requested Resource', { exact: true }).selectOption('resource-gitea-v1')
 await requester.getByRole('checkbox', { name: /^https / }).check()
 await requester.getByRole('checkbox', { name: /^ssh / }).check()
 await requester.getByLabel('Request reason', { exact: true }).fill('Browser independent request')
 await requester.getByRole('button', { name: 'Send access request', exact: true }).click()
 await requester.getByText('Request saved. No access has been granted.', { exact: true }).waitFor()
 const requestRow = requester.locator('.request-row').filter({ hasText: 'Browser independent request' })
 await requestRow.getByText('Pending', { exact: true }).waitFor()
 const original = await requester.evaluate(async () => (await (await fetch('/console/api/v1/access-requests')).json()).requests.find(r => r.reason === 'Browser independent request'))
 assert.equal(original.requester, 'person:requester'); assert.equal(original.source, '2001:db8::42/128')
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
 assert.equal(approved.approver, 'person:approver'); assert.equal(approved.expires_at, original.expires_at)
 assert(new Date(approved.approved_expires_at) <= new Date(original.expires_at)); assert(approved.grant_uid && approved.revision)
 const proof = await admin.evaluate(async () => ({ grants: await (await fetch('/console/api/v1/grants?limit=100')).json(), audit: await (await fetch('/console/api/v1/audit?limit=100')).json() }))
 const grant = proof.grants.grants.find(g => g.uid === approved.grant_uid)
 assert.equal(grant.source, '2001:db8::42/128'); assert.deepEqual(grant.scopes[0].endpoint_names, ['https'])
 assert.equal(grant.expires_at, approved.approved_expires_at)
 assert(proof.audit.events.some(e => e.actor === 'person:approver' && e.reason.includes(approved.uid)))
 for (const width of [375, 1440]) { await requester.setViewportSize({ width, height: 1000 }); assert(await requester.evaluate(() => document.documentElement.scrollWidth <= innerWidth)); await requester.screenshot({ path: join(evidence, `access-requests-${width}.png`), fullPage: true }) }
 // A separate real request is rejected; no publication revision is consumed.
 await requester.getByLabel('Source IP or CIDR', { exact: true }).fill('192.0.2.42')
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
 assert.equal(revoke, 200); assert.deepEqual(errors, [])
 for (const c of contexts) await c.close()
 console.log('PASS real human requester → independent approval/rejection → atomic Grant/SQLite/audit; IPv6 named narrowing, absolute deadline, requester isolation and revoke')
}
