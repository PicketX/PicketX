// SPDX-License-Identifier: Apache-2.0
// Real packaged binaries + copied built assets, never build from this runner.
import assert from 'node:assert/strict'
import { chown, cp, mkdir, mkdtemp, readFile, readdir, rename, rm, stat, writeFile } from 'node:fs/promises'
import { join, resolve } from 'node:path'
import { tmpdir } from 'node:os'
import { createHash, randomBytes } from 'node:crypto'
import { spawn, spawnSync } from 'node:child_process'
import { once } from 'node:events'
import { createConnection } from 'node:net'
import { chromium } from 'playwright'
import { freePort } from './ingress.mjs'
import { createControllerService } from '../../../scripts/controller-service-fixture.mjs'
import { copyPrivateState, relocateInstanceConfig, snapshotPrivateTree } from '../../../scripts/controller-state-copy.mjs'

const input = resolve(process.argv[2] ?? '/missing-controller-bundle')
const serviceMode = process.argv[3] === '--systemd'
if (serviceMode) assert(process.getuid() === 0 && process.env.PICKETX_SERVICE_TESTS === '1', 'real system-manager opt-in required')
assert((await stat(join(input, 'bin/picketx')).catch(() => null))?.isFile(), 'standalone CLI bundle required')
const temp = await mkdtemp(serviceMode ? '/run/picketx-service-' : join(tmpdir(), 'picketx-install-'))
const bundle = join(temp, 'relocated bundle'), stateParent = join(temp, 'private state')
let instance = join(stateParent, 'private instance')
const evidence = process.env.PICKETX_INSTALL_EVIDENCE_DIR ?? join(temp, 'evidence')
await mkdir(evidence, { recursive: true })
const env = Object.fromEntries(Object.entries(process.env).filter(([k]) => !k.startsWith('PICKETX_')))
env.PATH = '' // Neither executable may invoke Go/npm/Python/shell helpers at runtime.
let child, browser, service, logs = '', keys = [], completed = false
const publicError = value => keys.reduce((s, key) => s.replaceAll(key, '[REDACTED]'), String(value))
try {
  await cp(input, bundle, { recursive: true, errorOnExist: true, force: false })
  await mkdir(stateParent, { mode: 0o700 })
  if (serviceMode) service = await createControllerService({ temp, bundle, stateParent })
  assert.equal(spawnSync('sha256sum', ['--check', 'SHA256SUMS'], { cwd: bundle }).status, 0, 'bundle content integrity')
  const cli = join(bundle, 'bin/picketx'), binary = join(bundle, 'bin/picketx-controller')
  const source = JSON.parse(await readFile(join(bundle, 'SOURCE.json'), 'utf8'))
  assert.match(source.commit, /^[a-f0-9]{40}$/); assert.match(source.tree, /^[a-f0-9]{40}$/)
  const http = await freePort(), agent = await freePort(), authz = await freePort(), origin = `http://127.0.0.1:${http}`
  const args = ['controller-init', '--directory', instance, '--assets', join(bundle, 'web'), '--origin', origin, '--listen', `127.0.0.1:${http}`, '--agent-listen', serviceMode ? `[::1]:${agent}` : `127.0.0.1:${agent}`, '--authz-listen', `127.0.0.1:${authz}`]
  const initOptions = { cwd: temp, env, encoding: 'utf8', timeout: 10000, ...(service?.identity ?? {}) }
  const init = spawnSync(cli, args, initOptions)
  assert.equal(init.status, 0, 'PATH-empty init failed')
  keys = await Promise.all(['bootstrap-key', 'agent-token', 'ingress-token'].map(async name => (await readFile(join(instance, name), 'utf8')).trim()))
  assert.equal(new Set(keys).size, 3)
  for (const key of keys) assert(!(init.stdout + init.stderr).includes(key), 'init output leaked a credential')
  const repeat = spawnSync(cli, args, initOptions)
  assert.notEqual(repeat.status, 0, 'existing instance overwritten')
  assert.equal((await readFile(join(instance, 'bootstrap-key'), 'utf8')).trim(), keys[0], 'recovery key rotated on rerun')
  const api = (path, key) => fetch(origin + '/api/v1/' + path, { headers: key ? { Authorization: `Bearer ${key}` } : {} })
  async function ready() {
    for (let n = 0; n < 150; n++) {
      if (child) assert.equal(child.exitCode, null, 'packaged Controller exited before ready')
      try { if ((await fetch(origin + '/readyz')).ok) return } catch { /* real process startup */ }
      await new Promise(r => setTimeout(r, 100))
    }
    throw new Error('packaged Controller readiness deadline')
  }
  async function start() {
    logs = ''
    if (service) await service.start(instance)
    else {
      child = spawn(binary, ['-config', join(instance, 'controller.yaml')], { cwd: temp, env, stdio: ['ignore', 'pipe', 'pipe'] })
      child.stdout.on('data', b => { logs += b }); child.stderr.on('data', b => { logs += b })
      child.on('error', () => {})
    }
    await ready()
  }
  async function stop() {
    if (service) { await service.stop(); logs = service.logs(); return }
    const exit = once(child, 'exit'); child.kill('SIGTERM')
    const timer = setTimeout(() => child.kill('SIGKILL'), 10000)
    try { assert.deepEqual(await exit, [0, null], 'packaged Controller did not shut down cleanly') } finally { clearTimeout(timer) }
    child = null
  }
  await start()
  if (service) {
    const socket = createConnection({ host: '::1', port: agent })
    socket.setTimeout(5000, () => socket.destroy(new Error('IPv6 listener deadline')))
    try { await once(socket, 'connect') } finally { socket.destroy() }
    service.proof.dualStackListeners = true // IPv4 HTTP + actual IPv6 gRPC TCP listener, not Grant proof.
  }
  assert.equal((await fetch(origin + '/healthz')).status, 200)
  assert.equal((await api('publication')).status, 401)
  for (const key of keys.slice(1)) assert.equal((await api('publication', key)).status, 401, 'wrong audience accepted by management')
  const before = await (await api('publication', keys[0])).json()
  assert.equal(before.revision, 1); assert(before.policy.includes('empty: true'))
  for (const key of keys.slice(0, 2)) assert.equal((await fetch(`http://127.0.0.1:${authz}/authz/v1/nginx`, { headers: { Authorization: `Bearer ${key}`, 'X-PicketX-Route': 'route-web', 'X-PicketX-Source': '127.0.0.1', 'X-PicketX-Host': 'app.example.test', 'X-PicketX-Method': 'GET', 'X-PicketX-URI': '/' } })).status, 401, 'wrong audience accepted by decision listener')
  browser = await chromium.launch({ headless: true })
  const admin = await browser.newContext(), human = await browser.newContext()
  const page = await admin.newPage(), person = await human.newPage()
  await page.goto(origin + '/console/')
  await page.getByLabel('Access key', { exact: true }).fill(keys[0])
  await page.getByRole('button', { name: 'Sign in', exact: true }).click()
  await page.getByRole('button', { name: 'Accounts', exact: true }).click()
  await page.getByText('No human accounts yet.', { exact: true }).waitFor()
  const password = randomBytes(24).toString('hex'); keys.push(password)
  await page.getByLabel('Account username', { exact: true }).fill('operator')
  await page.getByLabel('Account password', { exact: true }).fill(password)
  await page.getByLabel('Admin', { exact: true }).check()
  await page.getByLabel('Account change reason', { exact: true }).fill('First installed operator')
  await page.getByRole('button', { name: 'Create account', exact: true }).click()
  await page.getByText('Account saved. Previous sessions are invalid.', { exact: true }).waitFor()
  async function humanLogin() {
    await person.goto(origin + '/console/')
    await person.getByLabel('Sign-in method', { exact: true }).selectOption('password')
    await person.getByLabel('Username', { exact: true }).fill('operator')
    await person.getByLabel('Password', { exact: true }).fill(password)
    await person.getByRole('button', { name: 'Sign in', exact: true }).click()
    await person.getByLabel('Policy YAML').waitFor()
  }
  await humanLogin()
  await person.getByLabel('Policy YAML').fill(before.policy.replace('name: empty', 'name: installed'))
  await person.getByLabel('Change reason', { exact: true }).fill('Installed operator publication')
  await person.getByRole('button', { name: 'Save desired configuration', exact: true }).click()
  await person.getByText('Desired configuration saved. Node application is not confirmed.', { exact: true }).waitFor()
  assert.equal((await (await api('publication', keys[0])).json()).revision, 2)
  const cookie = await human.cookies()
  if (service) {
    await service.crashRestart(); await ready()
    assert.equal((await fetch(origin + '/console/api/v1/publication', { headers: { Cookie: cookie.map(c => `${c.name}=${c.value}`).join('; ') } })).status, 401, 'crash-restarted session survived')
  }
  await stop(); await start()
  const old = await fetch(origin + '/console/api/v1/publication', { headers: { Cookie: cookie.map(c => `${c.name}=${c.value}`).join('; ') } })
  assert.equal(old.status, 401, 'in-memory session survived process restart')
  await human.clearCookies(); await humanLogin()
  const publication = await (await api('publication', keys[0])).json()
  assert.equal(publication.revision, 2); assert(publication.policy.includes('name: installed'))
  const accounts = await (await api('accounts', keys[0])).json()
  assert.equal(accounts.accounts.length, 1); assert.equal(accounts.accounts[0].username, 'operator')
  const audit = await person.evaluate(async () => (await fetch('/console/api/v1/audit')).json())
  assert(JSON.stringify(audit).includes('Installed operator publication'), 'durable publication audit missing')
  if (service) {
    const views = async () => Object.fromEntries(await Promise.all(['publication', 'accounts', 'audit'].map(async path => {
      const response = await api(path, keys[0]); assert.equal(response.status, 200)
      return [path, await response.json()]
    })))
    const beforeRestore = await views(), oldConfig = JSON.parse(await readFile(join(instance, 'controller.yaml'), 'utf8'))
    const restoreCookie = await human.cookies()
    await stop()
    const backup = join(temp, 'stopped backup'), restored = join(stateParent, 'restored instance')
    const snapshot = await copyPrivateState(instance, backup)
    assert.deepEqual(await copyPrivateState(backup, restored), snapshot)
    const config = relocateInstanceConfig(oldConfig, instance, restored)
    await writeFile(join(restored, 'controller.yaml'), JSON.stringify(config, null, 2) + '\n', { mode: 0o600 })
    const restoredSnapshot = await snapshotPrivateTree(restored)
    for (const path of Object.keys(snapshot).filter(path => path !== 'controller.yaml')) assert.equal(restoredSnapshot[path], snapshot[path])
    async function restoreOwner(path) {
      await chown(path, service.identity.uid, service.identity.gid)
      if ((await stat(path)).isDirectory()) for (const name of await readdir(path)) await restoreOwner(join(path, name))
    }
    await restoreOwner(restored)
    await rename(instance, join(stateParent, 'original unavailable'))
    instance = restored
    await start()
    assert.equal((await fetch(origin + '/console/api/v1/publication', { headers: { Cookie: restoreCookie.map(c => `${c.name}=${c.value}`).join('; ') } })).status, 401, 'session survived restored process')
    assert.deepEqual(await views(), beforeRestore, 'restored durable views changed')
    await human.clearCookies(); await humanLogin()
    service.proof.restore = { completeFiles: Object.keys(snapshot).length, snapshotSHA256: createHash('sha256').update(JSON.stringify(snapshot)).digest('hex'), changedOnlyConfigPaths: true, originalUnavailable: true, durableViewsEqual: true, oldSessionRejected: true, freshHumanLogin: true, authorityRetained: config.agent.authority === oldConfig.agent.authority }
    service.proof.status = 'pass'
  }
  for (const width of [1440, 375]) {
    await person.setViewportSize({ width, height: 1000 })
    assert(await person.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
    await person.screenshot({ path: join(evidence, `installed-controller-${width}.png`), fullPage: true })
  }
  await stop()
  for (const key of keys) assert(!logs.includes(key), 'runtime logged a credential')
  const report = { result: 'pass', source, runtimePATH: 'empty', relocated: true, initialRevision: 1, recoveredRevision: 2, accounts: 1, oldSessionRejected: true, independentAudiences: true, cleanSIGTERM: true }
  if (service) report.service = service.proof
  if (serviceMode) assert(report.service?.status === 'pass', 'actual systemd and stopped restore proof required')
  await writeFile(join(evidence, 'install-report.json'), JSON.stringify(report, null, 2) + '\n')
  console.log('PASS detached static Controller: private init, real browser bootstrap/human publication, restart/audit, no runtime tools')
  if (service) console.log('PASS actual unprivileged systemd Controller: crash restart, clean stop, complete stopped backup and relocated restore')
  completed = true
} catch (err) {
  // Never propagate child process argument/env/cause or unredacted private logs.
  console.error(publicError(err?.stack ?? err)); console.error(publicError(logs)); process.exitCode = 1
} finally {
  if (browser) await browser.close()
  if (child?.exitCode === null) { const exit = once(child, 'exit'); child.kill('SIGKILL'); await exit }
  let safe = true
  if (service) try { await service.close() } catch { safe = false; process.exitCode = 1; console.error('Owned service cleanup failed; private fixture retained for inspection') }
  if (safe) await rm(temp, { recursive: true, force: true })
}
assert(completed || process.exitCode === 1)
