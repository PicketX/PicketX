// SPDX-License-Identifier: Apache-2.0
// Real browser → built React → real Controller → SQLite. No API fixtures.
import assert from 'node:assert/strict'
import { mkdtemp, mkdir, readFile, writeFile, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { resolve, join } from 'node:path'
import { createHash, randomBytes } from 'node:crypto'
import { createServer } from 'node:net'
import { spawn, spawnSync } from 'node:child_process'
import { once } from 'node:events'
import { chromium } from 'playwright'
import { freePort, startIngress } from './ingress.mjs'

const root = resolve(import.meta.dirname, '../../..')
const temp = await mkdtemp(join(tmpdir(), 'picketx-browser-'))
const evidence = process.env.PICKETX_BROWSER_EVIDENCE_DIR ?? join(temp, 'screenshots')
await mkdir(evidence, { recursive: true })
const token = randomBytes(32).toString('hex'), viewer = randomBytes(32).toString('hex')
const hash = x => createHash('sha256').update(x).digest('hex')
let child, browser, ingress, logs = ''
try {
  const portServer = createServer()
  portServer.listen(0, '127.0.0.1'); await once(portServer, 'listening')
  const port = portServer.address().port; await new Promise(r => portServer.close(r))
  const origin = `http://127.0.0.1:${port}`
  const authzPort = await freePort(), ingressToken = randomBytes(32).toString('hex')
  const callersPath = join(temp, 'callers.json')
  await writeFile(callersPath, JSON.stringify({ callers: [{ id: 'nginx', sha256: hash(ingressToken), routes: ['route-v1', 'route-private-v1'] }] }), { mode: 0o600 })
  const principalPath = join(temp, 'principals.json')
  await writeFile(principalPath, JSON.stringify({ principals: [
    { id: 'principal:admin', sha256: hash(token), roles: ['role:admin'] },
    { id: 'principal:viewer', sha256: hash(viewer), roles: ['role:viewer'] },
  ] }), { mode: 0o600 })
  await writeFile(join(temp, 'agent.json'), '{"credentials":[]}', { mode: 0o600 })
  await writeFile(join(temp, 'runtime.yaml'), `http: {listen: "127.0.0.1:${port}"}
database: ${JSON.stringify(join(temp, 'state/db.sqlite'))}
credentials: ${JSON.stringify(join(temp, 'agent.json'))}
source: {mode: database}
management: {credentials: ${JSON.stringify(principalPath)}}
agent: {enabled: true, listen: "127.0.0.1:0", authority: "static:example-v1"}
browser: {enabled: true, origin: ${JSON.stringify(origin)}, assets: ${JSON.stringify(join(root, 'apps/web/dist'))}}
authz: {enabled: true, network: tcp, address: "127.0.0.1:${authzPort}", credentials: ${JSON.stringify(callersPath)}}
`, { mode: 0o600 })
  const built = spawnSync('go', ['build', '-o', join(temp, 'controller'), './cmd/picketx-controller'], { cwd: root, encoding: 'utf8' })
  assert.equal(built.status, 0, built.stderr)
  const runtimeEnv = Object.fromEntries(Object.entries(process.env).filter(([name]) => !name.startsWith('PICKETX_')))
  child = spawn(join(temp, 'controller'), ['-config', join(temp, 'runtime.yaml')], { cwd: root, env: runtimeEnv, stdio: ['ignore', 'pipe', 'pipe'] })
  child.stdout.on('data', x => { logs += x }); child.stderr.on('data', x => { logs += x })
  let ready = false
  for (let n = 0; n < 100; n++) {
    if (child.exitCode !== null) throw new Error(`Controller exited: ${logs}`)
    try { if ((await fetch(`${origin}/readyz`)).ok) { ready = true; break } } catch { /* startup */ }
    await new Promise(r => setTimeout(r, 100))
  }
  assert(ready, logs)
  ingress = await startIngress({ root, temp, authzPort, token: ingressToken })
  browser = await chromium.launch({ headless: true })
  const context = await browser.newContext()
  const page = await context.newPage()
  const errors = []; page.on('pageerror', e => errors.push(e.message))
  await page.goto(`${origin}/console/`)
  await page.getByLabel('Access key', { exact: true }).fill(token)
  await page.getByRole('button', { name: 'Sign in', exact: true }).click()
  await page.getByRole('heading', { name: 'Desired configuration', exact: true }).waitFor()
  await page.getByLabel('Policy YAML').waitFor()
  const original = await page.getByLabel('Policy YAML').inputValue()
  const originalFacts = await page.getByLabel('Facts JSON').inputValue()
  const policy = await readFile(join(root, 'examples/static/gitea.yaml'), 'utf8')
  const facts = await readFile(join(root, 'examples/static/facts.json'), 'utf8')
  await page.getByLabel('Policy YAML').fill(policy)
  await page.getByLabel('Facts JSON').fill(facts)
  await page.getByLabel('Change reason').fill('Browser actual publish')
  await page.getByRole('button', { name: 'Save desired configuration', exact: true }).click()
  await page.getByText('Desired configuration saved. Node application is not confirmed.', { exact: true }).waitFor()
  const accepted = await fetch(`${origin}/api/v1/publication`, { headers: { Authorization: `Bearer ${token}` } })
  const document = await accepted.json(); assert.equal(document.policy, policy); assert.equal(document.revision, 2)
  // New structured workflow must be a real navigable page, not only an API.
  await page.getByRole('button', { name: 'Grants', exact: true }).click({ timeout: 5000 })
  await page.getByRole('heading', { name: 'Grant access', exact: true }).waitFor()
  await page.getByLabel('Source IP or CIDR', { exact: true }).waitFor()
  await page.getByRole('button', { name: 'Desired configuration', exact: true }).click()
  assert.equal(await page.evaluate(() => localStorage.length + sessionStorage.length), 0)
  const cookies = await context.cookies(); const cookie = cookies.find(c => c.name === 'picketx_session')
  assert(cookie.httpOnly && cookie.sameSite === 'Strict' && cookie.path === '/console/')
  assert(!cookies.some(c => c.value.includes(token)))
  const draft = `${policy}\n# retained unsaved draft\n`
  await page.getByLabel('Policy YAML').fill(draft)
  await page.getByLabel('Change reason').fill('Will conflict')
  const external = await fetch(`${origin}/api/v1/publication`, { method: 'PUT', headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json', 'If-Match': accepted.headers.get('etag') }, body: JSON.stringify({ policy: original, facts: originalFacts, reason: 'Concurrent external update' }) })
  assert.equal(external.status, 200)
  await page.getByRole('button', { name: 'Save desired configuration', exact: true }).click()
  await page.getByText('The revision changed. Your draft is preserved; reload explicitly before editing again.', { exact: true }).waitFor()
  assert.equal(await page.getByLabel('Policy YAML').inputValue(), draft)
  // Failed reads preserve the draft and expose stale state (fault injection,
  // not an acceptance substitute for the real successful flow above).
  await page.route('**/console/api/v1/publication', r => r.abort())
  await page.getByRole('button', { name: 'Refresh status', exact: true }).click()
  await page.getByText('Last read failed. Displayed data may be stale.', { exact: true }).waitFor()
  assert.equal(await page.getByLabel('Policy YAML').inputValue(), draft)
  await page.unroute('**/console/api/v1/publication')
  await page.getByRole('button', { name: 'Audit', exact: true }).click()
  await page.getByText('Browser actual publish', { exact: true }).waitFor()
  await page.getByText('Concurrent external update', { exact: true }).waitFor()
  await page.setViewportSize({ width: 1440, height: 1000 })
  await page.screenshot({ path: join(evidence, 'audit-desktop.png'), fullPage: true })
  const longPolicy = `${policy}\n${'# readonly scrolling\n'.repeat(100)}`
  const readonlyDocument = await fetch(`${origin}/api/v1/publication`, { method: 'PUT', headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json', 'If-Match': external.headers.get('etag') }, body: JSON.stringify({ policy: longPolicy, facts, reason: 'Readonly long-document test' }) })
  assert.equal(readonlyDocument.status, 200)
  const viewerContext = await browser.newContext({ viewport: { width: 375, height: 812 }, reducedMotion: 'reduce' })
  const view = await viewerContext.newPage(); await view.goto(`${origin}/console/`)
  await view.getByLabel('Access key', { exact: true }).fill(viewer)
  await view.getByRole('button', { name: 'Sign in', exact: true }).click()
  await view.getByLabel('Policy YAML').waitFor()
  assert(await view.getByLabel('Policy YAML').isEnabled(), 'Viewer must be able to focus, read and copy documents')
  assert(await view.getByLabel('Policy YAML').evaluate(el => el.readOnly))
  await view.getByLabel('Policy YAML').focus()
  await view.keyboard.press('ControlOrMeta+A')
  assert(await view.getByLabel('Policy YAML').evaluate(el => document.activeElement === el && el.selectionStart === 0 && el.selectionEnd === el.value.length))
  await view.keyboard.press('ArrowDown')
  await view.keyboard.press('ControlOrMeta+End')
  const geometry = await view.getByLabel('Policy YAML').evaluate(el => ({ scrollHeight: el.scrollHeight, clientHeight: el.clientHeight, focused: document.activeElement === el }))
  assert(geometry.focused && geometry.scrollHeight > geometry.clientHeight, JSON.stringify(geometry))
  // Native keyboard scrolling completes asynchronously. Observe its result;
  // never assign scrollTop or replace the keyboard action with JS scrolling.
  await view.waitForFunction(() => document.querySelector('#policy')?.scrollTop > 0, null, { timeout: 3000 })
  assert(await view.getByLabel('Policy YAML').evaluate(el => el.scrollTop > 0))
  await view.keyboard.type('must-not-edit')
  assert.equal(await view.getByLabel('Policy YAML').inputValue(), longPolicy)
  assert(await view.getByLabel('Facts JSON').evaluate(el => el.readOnly && !el.disabled))
  assert(await view.getByRole('button', { name: 'Save desired configuration', exact: true }).isDisabled())
  assert(await view.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
  await view.screenshot({ path: join(evidence, 'viewer-mobile.png'), fullPage: true })
  const forbidden = await view.evaluate(async () => {
    const { csrf } = await (await fetch('/console/api/v1/session')).json()
    return (await fetch('/console/api/v1/publication', { method: 'PUT', headers: { 'X-CSRF-Token': csrf, 'Content-Type': 'application/json', 'If-Match': '"3"' }, body: '{}' })).status
  }); assert.equal(forbidden, 403)
  // A separate real browser publishes HTTP intent, then visits the real ingress.
  const operator = await context.newPage(); await operator.goto(`${origin}/console/`)
  await operator.getByLabel('Policy YAML').waitFor()
  const httpPolicy = (await readFile(join(root, 'examples/http/policy.yaml'), 'utf8')).replaceAll('__HOST__', `127.0.0.1:${ingress.port}`)
  const httpFacts = JSON.stringify({ authorityID: 'static:example-v1', agents: [], contexts: [] })
  const publishHTTP = async (value, reason) => {
    await operator.getByLabel('Policy YAML').fill(value); await operator.getByLabel('Facts JSON').fill(httpFacts)
    await operator.getByLabel('Change reason').fill(reason)
    const response = operator.waitForResponse(r => r.url().endsWith('/console/api/v1/publication') && r.request().method() === 'PUT')
    await operator.getByRole('button', { name: 'Save desired configuration', exact: true }).click()
    assert.equal((await response).status(), 200)
    await operator.getByText('Desired configuration saved. Node application is not confirmed.', { exact: true }).waitFor()
  }
  await publishHTTP(httpPolicy, 'Browser HTTP authorization')
  await ingress.verifyAllowedAndIsolation()
  const application = await context.newPage(); await application.goto(`${ingress.url}/allowed`)
  assert.equal(await application.locator('body').textContent(), 'protected real HTTP application')
  await application.screenshot({ path: join(evidence, 'http-application.png'), fullPage: true })
  const ipv6 = await fetch(`http://127.0.0.1:${authzPort}/authz/v1/decisions`, { method: 'POST', headers: { Authorization: `Bearer ${ingressToken}`, 'Content-Type': 'application/json' }, body: JSON.stringify({ route: 'route-v1', source: '::1', host: `127.0.0.1:${ingress.port}`, method: 'GET', uri: '/allowed' }) })
  assert.equal(ipv6.status, 200); assert.equal((await ipv6.json()).allowed, true)
  const explicitDeny = `${httpPolicy}\n---
apiVersion: picketx.io/v1alpha1
kind: NetworkDeny
metadata: {name: http-block, uid: http-block-v1}
spec:
  subject: {type: source, cidr: 127.0.0.1/32}
  scopes: [{resourceRef: {name: http-app, uid: http-app-v1}, endpointNames: [web]}]
  validity: {mode: permanent}
`
  await publishHTTP(explicitDeny, 'Browser explicit HTTP deny wins'); await ingress.verifyDenied()
  await publishHTTP(httpPolicy.replace('subject: {type: source, cidr: 127.0.0.1/32}', 'revoked: true\n  subject: {type: source, cidr: 127.0.0.1/32}'), 'Browser revoke HTTP source')
  await ingress.verifyDenied(); const denied = await application.reload(); assert.equal(denied.status(), 403)
  const expiry = new Date(Date.now() + 2500)
  await publishHTTP(httpPolicy.replace('validity: {mode: permanent}', `validity: {mode: bounded, expiresAt: '${expiry.toISOString()}'}`), 'Browser absolute HTTP deadline')
  await ingress.verifyAllowedAndIsolation()
  while (Date.now() < expiry.getTime()) await new Promise(r => setTimeout(r, 50))
  await ingress.verifyDenied()
  const decisionAudit = await operator.evaluate(async () => (await fetch('/console/api/v1/http-decisions?limit=100')).json())
  assert(decisionAudit.events.some(e => e.allowed && e.grant_ids.includes('loopback4-v1')))
  assert(decisionAudit.events.some(e => !e.allowed && e.reason === 'no-active-grant'))
  assert(decisionAudit.events.some(e => !e.allowed && e.reason === 'explicit-deny' && e.deny_ids.includes('http-block-v1')))
  assert(decisionAudit.events.every(e => !('uri' in e) && !('host' in e)))
  await writeFile(callersPath, '{"callers":[]}', { mode: 0o600 }); await ingress.verifyCallerRevoked()
  await writeFile(callersPath, '{"callers":null}', { mode: 0o600 }); await ingress.verifyUnavailable()
  console.log('PASS actual browser publication → source authorization → Nginx → application; revoke/deadline/audit/IPv6/scoped caller removal')
  // Removal invalidates the next real request even when the browser holds a cookie.
  await writeFile(principalPath, '{"principals":[]}', { mode: 0o600 })
  await page.getByRole('button', { name: 'Refresh status', exact: true }).click()
  await page.getByLabel('Access key', { exact: true }).waitFor()
  assert.deepEqual(errors, [])
  console.log('PASS real Chromium → React → SCS/CSRF → management API → SQLite: publish, audit, conflict/draft, read failure, viewer, mobile, credential revocation')
  console.log(`Screenshots: ${evidence}`)
} finally {
  if (browser?.contexts()[0]?.pages()[0]) await browser.contexts()[0].pages()[0].screenshot({ path: join(evidence, 'last-state.png'), fullPage: true }).catch(() => {})
  await browser?.close()
  await ingress?.close()
  if (child && child.exitCode === null) { const exited = once(child, 'exit'); child.kill('SIGTERM'); await exited }
  if (child) assert.equal(child.exitCode, 0, logs)
  // Preserve evidence on request; otherwise securely remove generated credentials.
  if (process.env.PICKETX_KEEP_BROWSER_EVIDENCE !== '1') await rm(temp, { recursive: true, force: true })
}
