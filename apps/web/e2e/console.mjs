// SPDX-License-Identifier: Apache-2.0
// Real browser → built React → real Controller → SQLite. No API fixtures.
import assert from 'node:assert/strict'
import { mkdtemp, readFile, writeFile, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { resolve, join } from 'node:path'
import { createHash, randomBytes } from 'node:crypto'
import { createServer } from 'node:net'
import { spawn, spawnSync } from 'node:child_process'
import { once } from 'node:events'
import { chromium } from 'playwright'

const root = resolve(import.meta.dirname, '../../..')
const temp = await mkdtemp(join(tmpdir(), 'picketx-browser-'))
const token = randomBytes(32).toString('hex'), viewer = randomBytes(32).toString('hex')
const hash = x => createHash('sha256').update(x).digest('hex')
let child, browser, logs = ''
try {
  const portServer = createServer()
  portServer.listen(0, '127.0.0.1'); await once(portServer, 'listening')
  const port = portServer.address().port; await new Promise(r => portServer.close(r))
  const origin = `http://127.0.0.1:${port}`
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
`, { mode: 0o600 })
  const built = spawnSync('go', ['build', '-o', join(temp, 'controller'), './cmd/picketx-controller'], { cwd: root, encoding: 'utf8' })
  assert.equal(built.status, 0, built.stderr)
  child = spawn(join(temp, 'controller'), ['-config', join(temp, 'runtime.yaml')], { cwd: root, stdio: ['ignore', 'pipe', 'pipe'] })
  child.stdout.on('data', x => { logs += x }); child.stderr.on('data', x => { logs += x })
  let ready = false
  for (let n = 0; n < 100; n++) {
    if (child.exitCode !== null) throw new Error(`Controller exited: ${logs}`)
    try { if ((await fetch(`${origin}/readyz`)).ok) { ready = true; break } } catch { /* startup */ }
    await new Promise(r => setTimeout(r, 100))
  }
  assert(ready, logs)
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
  await page.screenshot({ path: join(temp, 'audit-desktop.png'), fullPage: true })
  const viewerContext = await browser.newContext({ viewport: { width: 375, height: 812 }, reducedMotion: 'reduce' })
  const view = await viewerContext.newPage(); await view.goto(`${origin}/console/`)
  await view.getByLabel('Access key', { exact: true }).fill(viewer)
  await view.getByRole('button', { name: 'Sign in', exact: true }).click()
  await view.getByLabel('Policy YAML').waitFor()
  assert(await view.getByLabel('Policy YAML').isDisabled())
  assert(await view.getByRole('button', { name: 'Save desired configuration', exact: true }).isDisabled())
  assert(await view.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
  await view.screenshot({ path: join(temp, 'viewer-mobile.png'), fullPage: true })
  const forbidden = await view.evaluate(async () => {
    const { csrf } = await (await fetch('/console/api/v1/session')).json()
    return (await fetch('/console/api/v1/publication', { method: 'PUT', headers: { 'X-CSRF-Token': csrf, 'Content-Type': 'application/json', 'If-Match': '"3"' }, body: '{}' })).status
  }); assert.equal(forbidden, 403)
  // Removal invalidates the next real request even when the browser holds a cookie.
  await writeFile(principalPath, '{"principals":[]}', { mode: 0o600 })
  await page.getByRole('button', { name: 'Refresh status', exact: true }).click()
  await page.getByLabel('Access key', { exact: true }).waitFor()
  assert.deepEqual(errors, [])
  console.log('PASS real Chromium → React → SCS/CSRF → management API → SQLite: publish, audit, conflict/draft, read failure, viewer, mobile, credential revocation')
  console.log(`Screenshots: ${temp}`)
} finally {
  await browser?.close()
  if (child && child.exitCode === null) { const exited = once(child, 'exit'); child.kill('SIGTERM'); await exited; assert.equal(child.exitCode, 0, logs) }
  // Preserve evidence on request; otherwise securely remove generated credentials.
  if (process.env.PICKETX_KEEP_BROWSER_EVIDENCE !== '1') await rm(temp, { recursive: true, force: true })
}
