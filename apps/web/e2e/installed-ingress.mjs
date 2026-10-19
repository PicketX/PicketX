// SPDX-License-Identifier: Apache-2.0
// Installed binaries, real database-authoritative Controller, real Nginx and app.
import assert from 'node:assert/strict'
import { mkdtemp, mkdir, readFile, writeFile, rm } from 'node:fs/promises'
import { join, resolve } from 'node:path'
import { tmpdir } from 'node:os'
import { spawn, spawnSync } from 'node:child_process'
import { once } from 'node:events'
import http from 'node:http'
import { freePort } from './ingress.mjs'

const bundle = resolve(process.argv[2] ?? '/missing-bundle')
const temp = await mkdtemp(join(tmpdir(), 'picketx-installed-ingress-'))
const cli = join(bundle, 'bin/picketx'), core = join(bundle, 'bin/picketx-controller')
const env = Object.fromEntries(Object.entries(process.env).filter(([k]) => !k.startsWith('PICKETX_')))
env.PATH = ''
let keys = [], children = [], apps = [], clients = [], logs = '', completed = false
const redact = v => keys.reduce((s, k) => s.replaceAll(k, '[REDACTED]'), String(v))
const command = args => {
  const r = spawnSync(cli, args, { env, cwd: temp, encoding: 'utf8', timeout: 10000 })
  assert.equal(r.status, 0, 'installed PATH-empty CLI command failed')
  for (const key of keys) assert(!(r.stdout + r.stderr).includes(key), 'CLI disclosed credential')
}
const start = (bin, args, options = {}) => {
  const c = spawn(bin, args, { stdio: ['ignore', 'pipe', 'pipe'], ...options })
  c.stdout.on('data', b => { logs += b }); c.stderr.on('data', b => { logs += b })
  c.on('error', () => {}); children.push(c); return c
}
const stop = async c => {
  if (c.exitCode !== null) return
  const exited = once(c, 'exit'); c.kill('SIGTERM')
  const timer = setTimeout(() => c.kill('SIGKILL'), 10000)
  try { assert.deepEqual(await exited, [0, null], 'owned process did not stop cleanly') } finally { clearTimeout(timer) }
}
try {
  const proofs = []
  for (const callbackIP of ['127.0.0.1', '::1']) {
    const slot = join(temp, callbackIP === '::1' ? 'ipv6' : 'ipv4'); await mkdir(slot, { mode: 0o700 })
    const instance = join(slot, 'private instance'), httpPort = await freePort(), agentPort = await freePort()
    const authzPort = await freePort(callbackIP), nginxPort = await freePort(), origin = `http://127.0.0.1:${httpPort}`
    const address = callbackIP === '::1' ? `[::1]:${authzPort}` : `127.0.0.1:${authzPort}`
    command(['controller-init', '--directory', instance, '--assets', join(bundle, 'web'), '--origin', origin,
      '--listen', `127.0.0.1:${httpPort}`, '--agent-listen', `127.0.0.1:${agentPort}`, '--authz-listen', address])
    const admin = (await readFile(join(instance, 'bootstrap-key'), 'utf8')).trim()
    const token = (await readFile(join(instance, 'ingress-token'), 'utf8')).trim(); keys.push(admin, token)
    const includes = []
    for (const route of ['route-web', 'route-second']) {
      const dir = join(slot, `private includes ${route}`); includes.push(dir)
      command(['nginx-init', '--directory', dir, '--token-file', join(instance, 'ingress-token'), '--route-id', route, '--authz-address', address])
    }
    const registryPath = join(instance, 'callers.json')
    const registry = JSON.parse(await readFile(registryPath, 'utf8'))
    registry.callers[0].routes.push('route-second')
    await writeFile(registryPath, JSON.stringify(registry), { mode: 0o600 })
    const c = start(core, ['-config', join(instance, 'controller.yaml')], { cwd: slot, env })
    let ready = false
    for (let n = 0; n < 150; n++) {
      assert.equal(c.exitCode, null, 'installed Controller exited')
      try { ready = (await fetch(origin + '/readyz')).ok } catch { /* actual startup */ }
      if (ready) break
      await new Promise(r => setTimeout(r, 100))
    }
    assert(ready, 'real Controller readiness deadline')
    const version = spawnSync(process.env.PICKETX_NGINX ?? 'nginx', ['-V'], { encoding: 'utf8' })
    assert.equal(version.status, 0, 'real Nginx required, never skipped'); assert.match(version.stderr, /--with-http_auth_request_module/)
    let hits = 0
    const app = http.createServer((req, res) => { hits++; res.end('installed protected app') }); apps.push(app)
    app.listen(0, '127.0.0.1'); await once(app, 'listening')
    const appPort = app.address().port
    const config = `worker_processes 1;\npid "${slot}/nginx.pid";\nerror_log stderr error;\nevents { worker_connections 128; }\nhttp { access_log off; server {\nlisten 127.0.0.1:${nginxPort}; listen [::1]:${nginxPort};\n# Deliberate inherited bypass: generated protected include must force all.\nsatisfy any; allow all;\ninclude ${JSON.stringify(join(includes[0], 'server.conf'))};\ninclude ${JSON.stringify(join(includes[1], 'server.conf'))};\nlocation / { include ${JSON.stringify(join(includes[0], 'location.conf'))}; proxy_pass http://127.0.0.1:${appPort}; }\nlocation = /private { include ${JSON.stringify(join(includes[1], 'location.conf'))}; proxy_pass http://127.0.0.1:${appPort}; }\n} }\n`
    const configPath = join(slot, 'nginx.conf'); await writeFile(configPath, config, { mode: 0o600 })
    const nginx = start(process.env.PICKETX_NGINX ?? 'nginx', ['-p', slot + '/', '-c', configPath, '-g', 'daemon off;'])
    const keepAlive = new http.Agent({ keepAlive: true, maxSockets: 1 }); clients.push(keepAlive)
    const request = ({ ip = '127.0.0.1', path = '/allowed', method = 'GET', host = 'app.example.test', headers = {}, source } = {}) => new Promise((resolve, reject) => {
      const req = http.request({ hostname: ip, port: nginxPort, path, method, agent: source ? false : keepAlive, localAddress: source, headers: { Host: host, ...headers } }, res => {
        const socket = res.socket; let body = ''; res.on('data', b => { body += b }); res.on('end', () => resolve({ status: res.statusCode, body, socket }))
      }); req.on('error', reject); req.end()
    })
    let listening = false
    for (let n = 0; n < 100; n++) {
      assert.equal(nginx.exitCode, null, 'Nginx exited loading generated includes')
      try { assert.equal((await request()).status, 403, 'no initial Grant'); listening = true; break } catch (err) { if (err.code !== 'ECONNREFUSED') throw err }
      await new Promise(r => setTimeout(r, 50))
    }
    assert(listening); assert.equal(hits, 0, 'generation granted access')
    const getPublication = () => fetch(origin + '/api/v1/publication', { headers: { Authorization: `Bearer ${admin}` } })
    const initial = await getPublication(), before = await initial.json()
    const resource = `apiVersion: picketx.io/v1alpha1\nkind: Resource\nmetadata: {name: app, uid: app-v1}\nspec: {endpoints: [{name: web, network: {protocol: TCP, port: 443}}]}\n`
    const routes = `---\napiVersion: picketx.io/v1alpha1\nkind: HTTPRoute\nmetadata: {name: web, uid: route-web}\nspec: {resourceRef: {name: app, uid: app-v1}, endpointName: web, host: app.example.test, methods: [GET], paths: [/allowed]}\n---\napiVersion: picketx.io/v1alpha1\nkind: HTTPRoute\nmetadata: {name: second, uid: route-second}\nspec: {resourceRef: {name: app, uid: app-v1}, endpointName: web, host: app.example.test, methods: [POST], paths: [/private]}\n`
    const grants = ['127.0.0.1/32', '::1/128'].map((cidr, i) => `---\napiVersion: picketx.io/v1alpha1\nkind: Grant\nmetadata: {name: allow${i}, uid: allow${i}-v1}\nspec: {subject: {type: source, cidr: '${cidr}'}, scopes: [{resourceRef: {name: app, uid: app-v1}, endpointNames: [web]}], validity: {mode: permanent}}\n`).join('')
    const publish = async (policy, etag) => {
      const r = await fetch(origin + '/api/v1/publication', { method: 'PUT', headers: { Authorization: `Bearer ${admin}`, 'Content-Type': 'application/json', 'If-Match': etag }, body: JSON.stringify({ policy, facts: before.facts, reason: 'Installed trusted ingress acceptance' }) })
      assert.equal(r.status, 200, 'actual managed publication failed'); return r.headers.get('etag')
    }
    const etag = await publish(resource + routes + grants, initial.headers.get('etag'))
    let connection, allowed = 0, denied = 1
    const allow = async input => { const before = hits, r = await request(input); assert.equal(r.status, 200); assert.equal(r.body, 'installed protected app'); assert.equal(hits, before + 1); if (!input?.ip && connection) assert.equal(r.socket, connection, 'same live connection must reevaluate route/method'); if (!input?.ip) connection = r.socket; allowed++ }
    const deny = async (input, status) => { const before = hits, r = await request(input); if (status) assert.equal(r.status, status); else assert(r.status >= 400); assert.equal(hits, before, 'denied call reached application'); denied++ }
    await allow(); await allow({ path: '/private', method: 'POST' }); await allow()
    await allow({ ip: '::1' }); await allow({ ip: '::1', path: '/private', method: 'POST' })
    for (const input of [{ method: 'POST' }, { path: '/private' }, { path: '/wrong' }, { path: '/%61llowed' }, { host: 'evil.example' }, { source: '127.0.0.2', headers: { 'X-PicketX-Source': '127.0.0.1', 'X-Forwarded-For': '127.0.0.1', 'X-PicketX-Route': 'route-web', Authorization: `Bearer ${admin}` } }]) await deny(input)
    const internal = (await readFile(join(includes[0], 'location.conf'), 'utf8')).match(/auth_request (\/[^;]+);/)[1]
    await deny({ path: internal }, 404)
    await publish(resource + routes, etag)
    await deny({}, 403); await deny({ ip: '::1' }, 403); await deny({ path: '/private', method: 'POST' }, 403)
    await writeFile(registryPath, '{"callers":[]}', { mode: 0o600 }); await deny({}, 401)
    await stop(c); await deny({}, 500)
    await stop(nginx); keepAlive.destroy(); await new Promise(r => app.close(r))
    proofs.push({ callbackIP, status: 'pass', allowed, denied, privateIncludes: 4, methods: ['GET', 'POST'], sourceFamilies: ['IPv4', 'IPv6'], noInitialGrant: true, fixedRoutes: true, headerIsolation: true, denyNeverForwarded: true, revoked: true, callerRevoked: true, outageFailClosed: true, emptyRuntimePATH: true })
  }
  for (const key of keys) assert(!logs.includes(key), 'process logged secret')
  const dir = process.env.PICKETX_INGRESS_EVIDENCE_DIR
  if (dir) { await mkdir(dir, { recursive: true }); await writeFile(join(dir, 'installed-ingress-report.json'), JSON.stringify({ result: 'pass', source: JSON.parse(await readFile(join(bundle, 'SOURCE.json'), 'utf8')), proofs }, null, 2) + '\n') }
  console.log('PASS installed empty-PATH CLI → real Nginx includes → actual Controller → real application: IPv4/IPv6 callback/source, two fixed routes/methods, revoke/caller/outage fail closed')
  completed = true
} catch (err) { console.error(redact(err?.stack ?? err)); console.error(redact(logs)); process.exitCode = 1 }
finally {
  for (const client of clients) client.destroy()
  for (const c of children) if (c.exitCode === null) { const exited = once(c, 'exit'); c.kill('SIGKILL'); await exited }
  for (const app of apps) if (app.listening) await new Promise(r => app.close(r))
  await rm(temp, { recursive: true, force: true })
}
assert(completed || process.exitCode === 1)
