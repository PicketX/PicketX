// SPDX-License-Identifier: Apache-2.0
// Signed protocol fixture, NOT a production IdP. Controller/API/SQLite/React are real.
import assert from 'node:assert/strict'
import { mkdir, writeFile } from 'node:fs/promises'
import { join } from 'node:path'
import { randomBytes, createHash, generateKeyPairSync, sign } from 'node:crypto'
import { createServer } from 'node:http'
import { spawn } from 'node:child_process'
import { once } from 'node:events'
import { freePort } from './ingress.mjs'

export async function oidcLogin({ browser, root, temp, evidence }) {
  const dir = join(temp, 'oidc'); await mkdir(dir)
  const port = await freePort(), origin = `http://127.0.0.1:${port}`
  const token = randomBytes(32).toString('hex'), secret = randomBytes(32).toString('hex')
  const { privateKey, publicKey } = generateKeyPairSync('rsa', { modulusLength: 2048 })
  const jwk = { ...publicKey.export({ format: 'jwk' }), use: 'sig', alg: 'RS256', kid: 'fixture' }
  const codes = new Map(); let exchanges = 0, mode = 'valid', lastCallback = '', providerOrigin
  const issuer = createServer(async (req, res) => {
    try {
      const u = new URL(req.url, providerOrigin); res.setHeader('Cache-Control', 'no-store')
      const json = value => { res.setHeader('Content-Type', 'application/json'); res.end(JSON.stringify(value)) }
      if (u.pathname === '/.well-known/openid-configuration') return json({ issuer: providerOrigin, authorization_endpoint: `${providerOrigin}/authorize`, token_endpoint: `${providerOrigin}/token`, jwks_uri: `${providerOrigin}/keys`, id_token_signing_alg_values_supported: ['RS256'] })
      if (u.pathname === '/keys') return json({ keys: [jwk] })
      if (u.pathname === '/authorize') {
        assert.equal(u.searchParams.get('client_id'), 'browser-test'); assert.equal(u.searchParams.get('response_type'), 'code'); assert.equal(u.searchParams.get('scope'), 'openid')
        assert.equal(u.searchParams.get('redirect_uri'), `${origin}/console/api/v1/oidc/callback`); assert.equal(u.searchParams.get('code_challenge_method'), 'S256')
        const code = randomBytes(32).toString('hex'); codes.set(code, { nonce: u.searchParams.get('nonce'), challenge: u.searchParams.get('code_challenge') })
        const callback = new URL(`${origin}/console/api/v1/oidc/callback`); callback.searchParams.set('code', code); callback.searchParams.set('state', u.searchParams.get('state')); lastCallback = callback.href
        res.setHeader('Content-Type', 'text/html'); return res.end(`<title>Signed OIDC test issuer</title><h1>Signed OIDC test issuer</h1><p>Protocol fixture only, not production authentication.</p><a href="${callback.href.replaceAll('&', '&amp;')}">Continue as test user</a>`)
      }
      if (u.pathname === '/token') {
        let body = ''; for await (const chunk of req) body += chunk
        const q = new URLSearchParams(body), proof = codes.get(q.get('code')); codes.delete(q.get('code')); exchanges++
        assert(proof); assert.equal(req.headers.authorization, `Basic ${Buffer.from(`browser-test:${secret}`).toString('base64')}`)
        assert.equal(createHash('sha256').update(q.get('code_verifier')).digest('base64url'), proof.challenge)
        assert.equal(q.get('redirect_uri'), `${origin}/console/api/v1/oidc/callback`)
        const jwt = { iss: providerOrigin, aud: 'browser-test', sub: 'external-alice', iat: Math.floor(Date.now() / 1000), exp: Math.floor(Date.now() / 1000) + 600, nonce: mode === 'nonce' ? 'forged' : proof.nonce, email: 'administrator@example.test', roles: ['role:admin'] }
        const input = Buffer.from(JSON.stringify({ alg: 'RS256', kid: 'fixture' })).toString('base64url') + '.' + Buffer.from(JSON.stringify(jwt)).toString('base64url')
        return json({ access_token: 'fixture-not-persisted', token_type: 'Bearer', id_token: input + '.' + sign('RSA-SHA256', Buffer.from(input), privateKey).toString('base64url') })
      }
      res.writeHead(404).end()
    } catch { res.writeHead(400).end('Protocol fixture rejected request') }
  })
  // Different loopback IP gives an actual cross-site callback in Chromium.
  issuer.listen(0, '127.0.0.2'); await once(issuer, 'listening'); providerOrigin = `http://127.0.0.2:${issuer.address().port}`
  const bindings = join(dir, 'bindings.json'), provider = join(dir, 'provider.json'), runtime = join(dir, 'runtime.yaml')
  const bind = id => writeFile(bindings, JSON.stringify({ bindings: id ? [{ subject: 'external-alice', principal_id: id, generation: 'a'.repeat(32) }] : [] }), { mode: 0o600 })
  await bind(null)
  await writeFile(provider, JSON.stringify({ issuer: providerOrigin, client_id: 'browser-test', client_secret: secret, bindings }), { mode: 0o600 })
  await writeFile(join(dir, 'keys.json'), JSON.stringify({ principals: [{ id: 'principal:recovery', sha256: createHash('sha256').update(token).digest('hex'), roles: ['role:admin'] }] }), { mode: 0o600 })
  await writeFile(join(dir, 'agents.json'), '{"credentials":[]}', { mode: 0o600 })
  await writeFile(runtime, `http: {listen: "127.0.0.1:${port}"}
database: ${JSON.stringify(join(dir, 'state/db.sqlite'))}
credentials: ${JSON.stringify(join(dir, 'agents.json'))}
source: {mode: database}
management: {credentials: ${JSON.stringify(join(dir, 'keys.json'))}, users_mode: database}
agent: {enabled: true, listen: "127.0.0.1:0", authority: "oidc:test"}
browser: {enabled: true, origin: ${JSON.stringify(origin)}, assets: ${JSON.stringify(join(root, 'apps/web/dist'))}, oidc_config: ${JSON.stringify(provider)}}
`, { mode: 0o600 })
  const env = Object.fromEntries(Object.entries(process.env).filter(([k]) => !k.startsWith('PICKETX_')))
  const child = spawn(join(temp, 'controller'), ['-config', runtime], { cwd: root, env, stdio: ['ignore', 'pipe', 'pipe'] })
  let logs = ''; child.stdout.on('data', b => { logs += b }); child.stderr.on('data', b => { logs += b })
  const context = await browser.newContext(), page = await context.newPage(), errors = []; page.on('pageerror', e => errors.push(e.message))
  const api = async (path, method = 'GET', data, revision) => {
    const r = await fetch(`${origin}/api/v1${path}`, { method, headers: { Authorization: `Bearer ${token}`, ...(data ? { 'Content-Type': 'application/json' } : {}), ...(revision !== undefined ? { 'If-Match': `"${revision}"` } : {}) }, ...(data ? { body: JSON.stringify(data) } : {}) }); return r
  }
  try {
    let ready = false; for (let n = 0; n < 100; n++) { assert.equal(child.exitCode, null, 'OIDC Controller exited'); try { if ((await fetch(`${origin}/readyz`)).ok) { ready = true; break } } catch {} await new Promise(r => setTimeout(r, 100)) }; assert(ready, 'OIDC Controller readiness')
    const created = await api('/accounts', 'POST', { username: 'alice', password: randomBytes(24).toString('hex'), roles: ['role:admin'], reason: 'OIDC account provisioning' }, 0); assert.equal(created.status, 201); const account = await created.json(); await bind(account.id)
    await page.goto(`${origin}/console/`)
    await page.getByRole('button', { name: 'Continue with organization', exact: true }).waitFor({ timeout: 5000 })
    for (const width of [375, 1440]) { await page.setViewportSize({ width, height: 1000 }); assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)); await page.screenshot({ path: join(evidence, `oidc-login-${width}.png`), fullPage: true }) }
    await page.getByRole('button', { name: 'Language', exact: true }).click()
    await page.getByRole('button', { name: '使用组织账户登录', exact: true }).waitFor(); await page.screenshot({ path: join(evidence, 'oidc-login-zh-1440.png'), fullPage: true })
    await page.getByRole('button', { name: '语言', exact: true }).click()
    async function login() { await page.getByRole('button', { name: 'Continue with organization', exact: true }).click(); await page.getByRole('heading', { name: 'Signed OIDC test issuer' }).waitFor(); await page.getByRole('link', { name: 'Continue as test user' }).click() }
    // A stale signed-out tab can initiate replacement while another tab has a
    // session. The Strict cookie is absent at cross-site callback, so start must
    // revoke that original token under its existing same-session request lock.
    await page.getByLabel('Access key', { exact: true }).fill(token); await page.getByRole('button', { name: 'Sign in', exact: true }).click(); await page.getByRole('heading', { name: 'Desired configuration', exact: true }).waitFor()
    const originatingCookie = (await context.cookies()).find(c => c.name === 'picketx_session')
    const start = await page.evaluate(async () => (await fetch('/console/api/v1/oidc/start', { method: 'POST' })).json())
    assert.equal((await fetch(`${origin}/console/api/v1/me`, { headers: { Cookie: `picketx_session=${originatingCookie.value}` } })).status, 401)
    await page.goto(start.url); await page.getByRole('link', { name: 'Continue as test user' }).click(); await page.getByRole('heading', { name: 'Desired configuration', exact: true }).waitFor(); await page.getByText(account.id, { exact: true }).waitFor()
    const cookie = (await context.cookies()).find(c => c.name === 'picketx_session'); assert(cookie?.httpOnly && cookie.sameSite === 'Strict'); assert(!(await context.cookies()).some(c => c.name === 'picketx_oidc'))
    const me = await (await context.request.get(`${origin}/console/api/v1/me`)).json(); assert.equal(me.id, account.id); assert.deepEqual(me.roles, ['role:admin'])
    assert.equal(await page.evaluate(() => localStorage.length + sessionStorage.length), 0)
    const current = await (await api('/publication')).json(); assert.equal(current.revision, 1)
    // A new managed authority contains the explicit-empty marker. Replace it,
    // never append objects to it: mixed empty/nonempty candidates must still fail.
    const policy = 'apiVersion: picketx.io/v1alpha1\nkind: Resource\nmetadata: {name: oidc-proof, uid: oidc-proof-v1}\nspec:\n  endpoints:\n  - name: web\n    network: {protocol: TCP, port: 9443}\n'
    await page.getByLabel('Policy YAML', { exact: true }).fill(policy); await page.getByLabel('Change reason', { exact: true }).fill('OIDC verified human publication'); await page.getByRole('button', { name: 'Save desired configuration', exact: true }).click(); await page.getByText('Desired configuration saved. Node application is not confirmed.', { exact: true }).waitFor()
    assert((await (await api('/audit')).json()).events.some(e => e.actor === account.id && e.reason === 'OIDC verified human publication'))
    const persisted = await (await api('/publication')).json(); assert.equal(persisted.revision, 2); assert.equal(persisted.policy, policy)
    // Exact replay is consumed without another token exchange, even on real navigation.
    const priorExchanges = exchanges; await page.goto(lastCallback); assert.equal(exchanges, priorExchanges)
    assert.equal((await api(`/accounts/${account.id}`, 'PUT', { roles: ['role:viewer'], disabled: false, reason: 'OIDC role generation change' }, 1)).status, 200)
    assert.equal((await fetch(`${origin}/console/api/v1/me`, { headers: { Cookie: `picketx_session=${cookie.value}` } })).status, 401)
    await page.goto(`${origin}/console/`); await login(); await page.getByRole('heading', { name: 'Desired configuration', exact: true }).waitFor()
    const viewerMe = await (await context.request.get(`${origin}/console/api/v1/me`)).json(); assert.deepEqual(viewerMe.roles, ['role:viewer']); assert(!viewerMe.operations.includes('publication:write'))
    assert.equal(await page.evaluate(async () => (await fetch('/console/api/v1/publication', { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: '{}' })).status), 403)
    const viewerCookie = (await context.cookies()).find(c => c.name === 'picketx_session')
    await bind(null); assert.equal((await context.request.get(`${origin}/console/api/v1/me`)).status(), 401); await bind(account.id)
    assert.equal((await fetch(`${origin}/console/api/v1/me`, { headers: { Cookie: `picketx_session=${viewerCookie.value}` } })).status, 401)
    await page.goto(`${origin}/console/`); mode = 'nonce'; await login(); await page.getByRole('alert').getByText('Organization sign-in failed. Try again or contact your administrator.', { exact: true }).waitFor(); assert.equal(new URL(page.url()).search, ''); assert.equal((await context.request.get(`${origin}/console/api/v1/me`)).status(), 401)
    mode = 'valid'; await login(); await page.getByRole('heading', { name: 'Desired configuration', exact: true }).waitFor()
    const beforeDisable = (await context.cookies()).find(c => c.name === 'picketx_session')
    assert.equal((await api(`/accounts/${account.id}`, 'PUT', { roles: ['role:viewer'], disabled: true, reason: 'OIDC disable' }, 2)).status, 200)
    assert.equal((await api(`/accounts/${account.id}`, 'PUT', { roles: ['role:viewer'], disabled: false, reason: 'OIDC re-enable' }, 3)).status, 200)
    // No authenticated request between disable/re-enable: durable generation,
    // not an observed-cookie destruction, must permanently revoke the old login.
    assert.equal((await fetch(`${origin}/console/api/v1/me`, { headers: { Cookie: `picketx_session=${beforeDisable.value}` } })).status, 401)
    await page.goto(`${origin}/console/`); await login(); await page.getByRole('heading', { name: 'Desired configuration', exact: true }).waitFor()
    const logoutCookie = (await context.cookies()).find(c => c.name === 'picketx_session'); await page.getByRole('button', { name: 'Sign out', exact: true }).click()
    assert.equal((await fetch(`${origin}/console/api/v1/me`, { headers: { Cookie: `picketx_session=${logoutCookie.value}` } })).status, 401)
    assert.equal((await fetch(`${origin}/console/api/v1/me`, { headers: { Cookie: `picketx_session=${originatingCookie.value}` } })).status, 401)
    assert.deepEqual(errors, []); assert(!logs.includes(secret) && !logs.includes('fixture-not-persisted'))
    console.log('PASS actual cross-site Chromium → signed OIDC protocol fixture → Controller SCS/Casbin → SQLite publication/audit, fresh roles/binding/replay/nonce/logout; fixture is NOT a production IdP')
  } finally {
    await page.screenshot({ path: join(evidence, 'oidc-last.png'), fullPage: true }).catch(() => {}); await context.close()
    if (child.exitCode === null) { const exited = once(child, 'exit'); child.kill('SIGTERM'); await exited }; assert.equal(child.exitCode, 0, 'OIDC Controller shutdown')
    await new Promise(r => issuer.close(r))
  }
}
