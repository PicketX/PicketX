// SPDX-License-Identifier: Apache-2.0
// Actual upstream Keycloak + installed Controller, not a protocol fixture.
// Run ONLY in disposable CI: the owned temporary CA is added to user NSS trust.
import assert from 'node:assert/strict'
import { mkdtemp, mkdir, readFile, writeFile, rm } from 'node:fs/promises'
import { tmpdir, homedir } from 'node:os'
import { join, resolve } from 'node:path'
import { spawn, spawnSync } from 'node:child_process'
import { once } from 'node:events'
import { randomBytes } from 'node:crypto'
import { chromium } from 'playwright'
import { freePort } from './ingress.mjs'
import { createTestCertificate, requestHTTPS } from '../../../scripts/real-idp-https.mjs'

const bundle = resolve(process.argv[2] ?? '/missing-bundle'), keycloak = resolve(process.argv[3] ?? '/missing-keycloak')
const archiveSHA256 = '4e2740b181faa2ea623a1c2ecf00855935311e546046e2f857615a47ed24cbe8'
let phase = 'setup', directory, browser, context, trustLabel, realmFile, activePage, siteOrigins, completed = false
const children = [], observations = { s256: false, nonce: false }
const nss = join(homedir(), '.pki/nssdb')
const command = (bin, args, options = {}) => {
  const result = spawnSync(bin, args, { encoding: 'utf8', timeout: 15000, ...options })
  assert.equal(result.status, 0, 'owned deployment command failed')
  return result.stdout
}
const start = (bin, args, env) => {
  const child = spawn(bin, args, { env, stdio: ['ignore', 'ignore', 'ignore'] })
  child.on('error', () => {}); children.push(child); return child
}
async function stop(child, clean = false) {
  if (child.exitCode !== null || child.signalCode !== null) { if (clean) assert.equal(child.exitCode, 0); return }
  const exit = once(child, 'exit'); child.kill('SIGTERM')
  const timer = setTimeout(() => child.kill('SIGKILL'), 10000)
  try { const [code] = await exit; if (clean) assert.equal(code, 0, 'installed Controller clean shutdown') } finally { clearTimeout(timer) }
}
async function ready(child, probe, milliseconds = 120000) {
  const deadline = Date.now() + milliseconds
  while (Date.now() < deadline) {
    assert(child.exitCode === null && child.signalCode === null, 'owned process exited before ready')
    try { if (await probe()) return } catch { /* bounded actual startup only */ }
    await new Promise(resolve => setTimeout(resolve, 200))
  }
  throw new Error('owned deployment readiness deadline')
}
try {
  assert(process.env.PICKETX_REAL_IDP_TESTS === '1' && process.env.GITHUB_ACTIONS === 'true', 'disposable CI opt-in required')
  directory = await mkdtemp(join(tmpdir(), 'picketx-real-idp-')); await mkdir(join(directory, 'private'), { mode: 0o700 })
  const privateDir = join(directory, 'private'), certificate = await createTestCertificate(privateDir), ca = await readFile(certificate.ca)
  const publicPort = await freePort(), backendPort = await freePort(), providerPort = await freePort()
  const origin = `https://127.0.0.1:${publicPort}`, providerOrigin = `https://127.0.0.2:${providerPort}`, issuer = providerOrigin + '/realms/picketx-acceptance'
  siteOrigins = { controller: origin, provider: providerOrigin }
  const secret = randomBytes(32).toString('hex'), password = randomBytes(24).toString('hex'), subject = '9f4101d6-0400-4000-8000-111111111111'
  const source = JSON.parse(await readFile(join(bundle, 'SOURCE.json'), 'utf8'))
  assert.match(source.commit, /^[a-f0-9]{40}$/); assert.match(source.tree, /^[a-f0-9]{40}$/)
  phase = 'provider-version'
  assert(command(join(keycloak, 'bin/kc.sh'), ['--version']).includes('26.8.1'), 'pinned actual provider version')
  await mkdir(join(keycloak, 'data/import'), { recursive: true, mode: 0o700 })
  const importPath = join(keycloak, 'data/import/picketx-acceptance-realm.json')
  await writeFile(importPath, JSON.stringify({
    realm: 'picketx-acceptance', enabled: true, sslRequired: 'all', registrationAllowed: false,
    users: [{ id: subject, username: 'picketx-operator', enabled: true, emailVerified: true, firstName: 'Test', lastName: 'Operator', email: 'operator@example.test', credentials: [{ type: 'password', value: password, temporary: false }] }],
    clients: [{ clientId: 'picketx-installed', enabled: true, protocol: 'openid-connect', publicClient: false, secret, standardFlowEnabled: true, directAccessGrantsEnabled: false, redirectUris: [origin + '/console/api/v1/oidc/callback'], webOrigins: [origin], attributes: { 'pkce.code.challenge.method': 'S256' } }],
  }), { mode: 0o600, flag: 'wx' }); realmFile = importPath
  phase = 'provider-start'
  const provider = start(join(keycloak, 'bin/kc.sh'), ['start', '--db=dev-file', '--http-enabled=false', '--http-host=127.0.0.2', `--https-port=${providerPort}`, `--hostname=${providerOrigin}`, `--https-certificate-file=${certificate.cert}`, `--https-certificate-key-file=${certificate.key}`, '--import-realm'], { ...process.env, JAVA_OPTS_APPEND: '-Xms128m -Xmx512m' })
  await ready(provider, async () => (await requestHTTPS(issuer + '/.well-known/openid-configuration', { ca })).status === 200)
  const discovery = (await requestHTTPS(issuer + '/.well-known/openid-configuration', { ca })).json()
  assert.equal(discovery.issuer, issuer); assert(discovery.authorization_endpoint.startsWith(providerOrigin + '/'))
  await assert.rejects(requestHTTPS(issuer + '/.well-known/openid-configuration'), { message: 'HTTPS observation failed' })
  await assert.rejects(requestHTTPS(issuer + '/.well-known/openid-configuration', { ca, servername: 'wrong.example.test' }), { message: 'HTTPS observation failed' })
  phase = 'installed-init'
  const runtimeEnv = Object.fromEntries(Object.entries(process.env).filter(([key]) => !key.startsWith('PICKETX_')))
  runtimeEnv.PATH = ''; runtimeEnv.SSL_CERT_FILE = certificate.ca
  const instance = join(privateDir, 'instance'), cli = join(bundle, 'bin/picketx')
  command(cli, ['controller-init', '--directory', instance, '--assets', join(bundle, 'web'), '--origin', origin, '--listen', `127.0.0.1:${backendPort}`, '--agent-listen', `127.0.0.1:${await freePort()}`, '--authz-listen', `127.0.0.1:${await freePort()}`], { env: runtimeEnv })
  const token = (await readFile(join(instance, 'bootstrap-key'), 'utf8')).trim(), bindings = join(instance, 'bindings.json'), oidc = join(instance, 'provider.json')
  await writeFile(bindings, '{"bindings":[]}', { mode: 0o600 })
  await writeFile(oidc, JSON.stringify({ issuer, client_id: 'picketx-installed', client_secret: secret, bindings }), { mode: 0o600 })
  const configuration = JSON.parse(await readFile(join(instance, 'controller.yaml'), 'utf8')); configuration.browser.oidc_config = oidc
  await writeFile(join(instance, 'controller.yaml'), JSON.stringify(configuration), { mode: 0o600 })
  const core = start(join(bundle, 'bin/picketx-controller'), ['-config', join(instance, 'controller.yaml')], runtimeEnv)
  const nginxConfig = join(privateDir, 'nginx.conf')
  await writeFile(nginxConfig, `daemon off;\npid ${JSON.stringify(join(privateDir, 'nginx.pid'))};\nerror_log ${JSON.stringify(join(privateDir, 'nginx.log'))};\nevents {}\nhttp { access_log off; server { listen 127.0.0.1:${publicPort} ssl; server_name 127.0.0.1; ssl_certificate ${JSON.stringify(certificate.cert)}; ssl_certificate_key ${JSON.stringify(certificate.key)}; location / { proxy_set_header Host $http_host; proxy_pass http://127.0.0.1:${backendPort}; } } }\n`, { mode: 0o600 })
  const nginx = start('nginx', ['-p', directory, '-c', nginxConfig], process.env)
  phase = 'installed-ready'
  await ready(core, async () => (await requestHTTPS(origin + '/readyz', { ca })).status === 200)
  const api = (path, method = 'GET', data, revision) => requestHTTPS(origin + '/api/v1' + path, { ca, method, headers: { Authorization: `Bearer ${token}`, ...(data ? { 'Content-Type': 'application/json' } : {}), ...(revision === undefined ? {} : { 'If-Match': `"${revision}"` }) }, ...(data ? { body: JSON.stringify(data) } : {}) })
  const created = await api('/accounts', 'POST', { username: 'idp-operator', password: randomBytes(24).toString('hex'), roles: ['role:admin'], reason: 'real IdP provisioning' }, 0)
  assert.equal(created.status, 201); const account = created.json()
  phase = 'browser-trust'
  await mkdir(nss, { recursive: true })
  try { await readFile(join(nss, 'cert9.db')) } catch (error) { if (error.code !== 'ENOENT') throw error; command('certutil', ['-N', '--empty-password', '-d', `sql:${nss}`]) }
  trustLabel = 'picketx-real-idp-' + randomBytes(8).toString('hex')
  command('certutil', ['-A', '-d', `sql:${nss}`, '-n', trustLabel, '-t', 'C,,', '-i', certificate.ca])
  browser = await chromium.launch({ headless: true }); context = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  const page = await context.newPage(), pageErrors = []
  activePage = page
  page.on('pageerror', () => pageErrors.push('browser error'))
  page.on('request', request => { const url = new URL(request.url()); if (url.origin === providerOrigin && url.pathname.endsWith('/protocol/openid-connect/auth')) { observations.s256 = url.searchParams.get('code_challenge_method') === 'S256' && Boolean(url.searchParams.get('code_challenge')); observations.nonce = Boolean(url.searchParams.get('nonce')) } })
  const me = () => page.evaluate(async () => { const response = await fetch('/console/api/v1/me'); return { status: response.status, ...(response.ok ? { identity: await response.json() } : {}) } })
  async function begin() { await page.goto(origin + '/console/'); await page.getByRole('button', { name: 'Continue with organization', exact: true }).click() }
  async function signedIn() { await page.getByRole('heading', { name: 'Desired configuration', exact: true }).waitFor({ timeout: 15000 }); const identity = await me(); assert.equal(identity.status, 200); assert.equal(identity.identity.id, account.id); return identity.identity }
  phase = 'real-password-unbound'
  await begin(); phase = 'provider-login-form'; await page.locator('#username').waitFor()
  phase = 'correlation-cookie'
  const correlation = (await context.cookies()).find(cookie => cookie.name === 'picketx_oidc')
  assert(correlation?.secure && correlation.httpOnly && correlation.sameSite === 'Lax', 'HTTPS correlation cookie boundary')
  phase = 'wrong-password'
  await page.locator('#username').fill('picketx-operator'); await page.locator('#password').fill('deliberately-incorrect-password'); await page.locator('#kc-login').click(); await page.locator('#input-error').waitFor()
  phase = 'valid-unbound-callback'
  await page.locator('#password').fill(password); await page.locator('#kc-login').click()
  await page.getByRole('alert').getByText('Organization sign-in failed. Try again or contact your administrator.', { exact: true }).waitFor()
  assert.equal((await me()).status, 401)
  phase = 'bound-login'
  const bind = () => writeFile(bindings, JSON.stringify({ bindings: [{ subject, principal_id: account.id, generation: randomBytes(16).toString('hex') }] }), { mode: 0o600 })
  await bind(); await begin(); const identity = await signedIn(); assert.deepEqual(identity.roles, ['role:admin'])
  const cookie = (await context.cookies()).find(item => item.name === 'picketx_session')
  assert(cookie?.secure && cookie.httpOnly && cookie.sameSite === 'Strict'); assert(!(await context.cookies()).some(item => item.name === 'picketx_oidc'))
  assert(observations.s256 && observations.nonce); assert.equal(await page.evaluate(() => localStorage.length + sessionStorage.length), 0)
  phase = 'human-publication'
  const policy = 'apiVersion: picketx.io/v1alpha1\nkind: Resource\nmetadata: {name: real-idp-proof, uid: real-idp-proof-v1}\nspec:\n  endpoints:\n  - name: web\n    network: {protocol: TCP, port: 9443}\n'
  await page.getByLabel('Policy YAML', { exact: true }).fill(policy); await page.getByLabel('Change reason', { exact: true }).fill('real Keycloak human publication'); await page.getByRole('button', { name: 'Save desired configuration', exact: true }).click()
  await page.getByText('Desired configuration saved. Node application is not confirmed.', { exact: true }).waitFor()
  const published = (await api('/publication')).json(); assert.equal(published.revision, 2); assert.equal(published.policy, policy)
  assert((await api('/audit')).json().events.some(event => event.actor === account.id && event.reason === 'real Keycloak human publication'))
  const evidence = process.env.PICKETX_REAL_IDP_EVIDENCE_DIR; assert(evidence, 'explicit public evidence directory required'); await mkdir(evidence, { recursive: true })
  await page.screenshot({ path: join(evidence, 'real-idp-controller-1440.png'), fullPage: true })
  phase = 'current-local-roles'
  assert.equal((await api(`/accounts/${account.id}`, 'PUT', { roles: ['role:viewer'], disabled: false, reason: 'real IdP local role revocation' }, 1)).status, 200)
  assert.equal((await requestHTTPS(origin + '/console/api/v1/me', { ca, headers: { Cookie: `picketx_session=${cookie.value}` } })).status, 401)
  await begin(); const viewer = await signedIn(); assert.deepEqual(viewer.roles, ['role:viewer']); assert(!viewer.operations.includes('publication:write'))
  const writeStatus = await page.evaluate(async () => { const session = await (await fetch('/console/api/v1/session')).json(); return (await fetch('/console/api/v1/publication', { method: 'PUT', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': session.csrf, 'If-Match': '"2"' }, body: JSON.stringify({ policy: '', facts: '{}', reason: 'viewer cannot write' }) })).status })
  assert.equal(writeStatus, 403); assert.equal((await api('/publication')).json().revision, 2)
  phase = 'binding-revoke-logout'
  const viewerCookie = (await context.cookies()).find(item => item.name === 'picketx_session')
  await writeFile(bindings, '{"bindings":[]}', { mode: 0o600 }); assert.equal((await me()).status, 401)
  await bind(); assert.equal((await requestHTTPS(origin + '/console/api/v1/me', { ca, headers: { Cookie: `picketx_session=${viewerCookie.value}` } })).status, 401)
  await begin(); await signedIn(); const logoutCookie = (await context.cookies()).find(item => item.name === 'picketx_session')
  await page.getByRole('button', { name: 'Sign out', exact: true }).click(); await page.getByRole('heading', { name: 'Sign in', exact: true }).waitFor()
  assert.equal((await requestHTTPS(origin + '/console/api/v1/me', { ca, headers: { Cookie: `picketx_session=${logoutCookie.value}` } })).status, 401)
  assert.equal(pageErrors.length, 0)
  await context.close(); context = undefined; await browser.close(); browser = undefined
  await stop(core, true); await stop(nginx); await stop(provider)
  phase = 'cleanup-trust'
  command('certutil', ['-D', '-d', `sql:${nss}`, '-n', trustLabel]); trustLabel = undefined
  await writeFile(join(evidence, 'real-idp-report.json'), JSON.stringify({ result: 'pass', source, provider: { kind: 'real-keycloak', version: '26.8.1', archiveSHA256, disposable: true }, runtimePATH: 'empty', proofs: { normalVerifiedTLS: true, unknownCARejected: true, wrongSANRejected: true, realPasswordRejected: true, validUnboundRejected: true, explicitSubjectBinding: true, s256: true, nonce: true, secureLaxCorrelation: true, secureStrictSession: true, correlationCleared: true, noBrowserTokenStorage: true, sameLocalPrincipal: true, humanPublicationAndAudit: true, currentLocalRoles: true, oldCookieRejected: true, viewerWriteDenied: true, bindingWithdrawalDenied: true, localLogout: true, ownedTrustRemoved: true, cleanControllerShutdown: true } }, null, 2) + '\n')
  console.log('PASS actual Keycloak26.8.1 → normal CA/SAN HTTPS → installed Controller/React/SCS/Casbin/SQLite: password/unbound denial, Secure cookies, current roles, publication/audit and revoke/logout; disposable real provider NOT production deployment or Agent conjunction')
  completed = true
} catch {
  console.error('Real IdP acceptance failed during ' + phase)
  // Classification only, never URL/query/body/credential/exception text.
  if (activePage) {
    try {
      const actualOrigin = new URL(activePage.url()).origin
      const flags = { atProvider: actualOrigin === siteOrigins.provider, atController: actualOrigin === siteOrigins.controller, usernamePresent: await activePage.locator('#username').count() > 0, passwordPresent: await activePage.locator('#password').count() > 0, providerErrorPresent: await activePage.locator('#kc-error-message').count() > 0, inputErrorPresent: await activePage.locator('#input-error').count() > 0, controllerAlertPresent: await activePage.getByRole('alert').count() > 0 }
      console.error('Real IdP failure classification ' + JSON.stringify(flags))
    } catch { console.error('Real IdP failure classification unavailable') }
  }
  process.exitCode = 1
}
finally {
  await context?.close().catch(() => {}); await browser?.close().catch(() => {})
  for (const child of children.toReversed()) { try { await stop(child) } catch { console.error('Real IdP owned-process cleanup failed'); process.exitCode = 1 } }
  if (trustLabel) { try { command('certutil', ['-D', '-d', `sql:${nss}`, '-n', trustLabel]) } catch { console.error('Real IdP owned-trust cleanup failed'); process.exitCode = 1 } }
  if (realmFile) await rm(realmFile).catch(() => { console.error('Real IdP owned-import cleanup failed'); process.exitCode = 1 })
  if (directory) await rm(directory, { recursive: true, force: true }).catch(() => { console.error('Real IdP owned-state cleanup failed'); process.exitCode = 1 })
}
assert(completed || process.exitCode === 1)
