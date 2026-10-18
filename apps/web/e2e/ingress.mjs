// SPDX-License-Identifier: Apache-2.0
// Actual Nginx auth_request → real local decision listener → real HTTP application.
import assert from 'node:assert/strict'
import http from 'node:http'
import { createServer } from 'node:net'
import { once } from 'node:events'
import { writeFile, readFile } from 'node:fs/promises'
import { join } from 'node:path'
import { spawn, spawnSync } from 'node:child_process'

export async function freePort(host = '127.0.0.1') {
  const s = createServer(); s.listen(0, host); await once(s, 'listening')
  const p = s.address().port; await new Promise(r => s.close(r)); return p
}
export async function startIngress({ root, temp, authzPort, token }) {
  const binary = process.env.PICKETX_NGINX ?? 'nginx'
  const version = spawnSync(binary, ['-V'], { encoding: 'utf8' })
  assert.equal(version.status, 0, `Nginx is required, never skipped: ${version.stderr}`)
  assert.match(version.stderr, /--with-http_auth_request_module/)
  console.log(version.stderr.split('\n')[0])
  let count = 0
  const app = http.createServer((req, res) => { count++; res.end('protected real HTTP application') })
  app.listen(0, '127.0.0.1'); await once(app, 'listening')
  const appPort = app.address().port, port = await freePort()
  const template = await readFile(join(root, 'examples/http/nginx.conf.template'), 'utf8')
  const config = template.replaceAll('__PREFIX__', temp).replaceAll('__PORT__', String(port)).replaceAll('__APP_PORT__', String(appPort)).replaceAll('__AUTHZ_PORT__', String(authzPort)).replaceAll('__TOKEN__', token)
  const file = join(temp, 'nginx.conf'); await writeFile(file, config, { mode: 0o600 })
  let logs = ''
  const nginx = spawn(binary, ['-p', `${temp}/`, '-c', file, '-g', 'daemon off;'], { stdio: ['ignore', 'pipe', 'pipe'] })
  nginx.stdout.on('data', x => { logs += x }); nginx.stderr.on('data', x => { logs += x })
  const agent = new http.Agent({ keepAlive: true, maxSockets: 1 })
  let connection
  const request = ({ path = '/allowed', method = 'GET', host = `127.0.0.1:${port}`, headers = {}, source } = {}) => new Promise((resolve, reject) => {
    const req = http.request({ hostname: '127.0.0.1', port, method, path, agent: source ? false : agent, localAddress: source, headers: { Host: host, ...headers } }, res => {
      const socket = res.socket; let body = ''; res.on('data', x => { body += x }); res.on('end', () => resolve({ status: res.statusCode, body, socket }))
    }); req.on('error', reject); req.end()
  })
  for (let n = 0; n < 100; n++) {
    if (nginx.exitCode !== null) throw new Error(`Nginx exited: ${logs}`)
    try { await request(); break } catch (e) { if (n === 99) throw e }
    await new Promise(r => setTimeout(r, 50))
  }
  return {
    port, url: `http://127.0.0.1:${port}`, count: () => count,
    async verifyAllowedAndIsolation() {
      let r = await request(); assert.equal(r.status, 200, logs); assert.equal(r.body, 'protected real HTTP application'); connection = r.socket
      const before = count
      for (const input of [{ path: '/private', headers: { 'X-PicketX-Route': 'route-v1' } }, { method: 'POST' }, { host: 'other.example' }, { path: '/%61llowed' }, { source: '127.0.0.2', headers: { 'X-Forwarded-For': '127.0.0.1', 'X-PicketX-Source': '127.0.0.1', 'X-Real-IP': '127.0.0.1' } }]) {
        r = await request(input); assert(r.status >= 400, JSON.stringify({ input, status: r.status })); if (!input.source) assert.equal(r.socket, connection, 'Must reevaluate a reused HTTP connection')
      }
      assert.equal(count, before, 'Denied request reached application')
      r = await request(); assert.equal(r.status, 200); assert.equal(r.socket, connection)
      console.log('PASS real Nginx keepalive target/method/path/source-header isolation; denied calls never reached application')
    },
    async verifyDenied() { const before = count; const r = await request(); assert.equal(r.status, 403, logs); assert.equal(count, before); assert.equal(r.socket, connection) },
    async verifyUnavailable() { const before = count; const r = await request(); assert(r.status >= 500); assert.equal(count, before) },
    async verifyCallerRevoked() { const before = count; const r = await request(); assert.equal(r.status, 401); assert.equal(count, before) },
    async close() { agent.destroy(); if (nginx.exitCode === null) { const exited = once(nginx, 'exit'); nginx.kill('SIGTERM'); await exited }; await new Promise(r => app.close(r)); assert.equal(nginx.exitCode, 0, logs) },
  }
}
