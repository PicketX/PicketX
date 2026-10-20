// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict'
import { test, before, after } from 'node:test'
import { mkdtemp, readFile, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import https from 'node:https'
import * as helper from './real-idp-https.mjs'
let dir, tls, server, origin
before(async () => {
  dir = await mkdtemp(join(tmpdir(), 'picketx-idp-tls-test-'))
  tls = await helper.createTestCertificate(dir)
  server = https.createServer({ key: await readFile(tls.key), cert: await readFile(tls.cert) }, (req, res) => {
    if (req.url === '/slow') return
    if (req.url === '/large') return res.end('PRIVATE-BODY'.repeat(110000))
    if (req.url === '/invalid') return res.end('PRIVATE-INVALID-JSON')
    res.setHeader('Content-Type', 'application/json'); res.end('{"identity":"real-TLS-response"}')
  })
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve)); origin = `https://127.0.0.1:${server.address().port}`
})
after(async () => { if (server) { server.closeAllConnections(); await new Promise(r => server.close(r)) }; if (dir) await rm(dir, { recursive: true, force: true }) })
test('normal verified HTTPS returns the actual response with explicit CA', async () => {
  const response = await helper.requestHTTPS(origin, { ca: await readFile(tls.ca) })
  assert.equal(response.status, 200); assert.deepEqual(response.json(), { identity: 'real-TLS-response' })
})
test('unknown CA cannot establish trusted evidence', async () => {
  await assert.rejects(helper.requestHTTPS(origin), { message: 'HTTPS observation failed' })
})
test('trusted CA does not disable server hostname verification', async () => {
  await assert.rejects(helper.requestHTTPS(origin, { ca: await readFile(tls.ca), servername: 'wrong.example.test' }), { message: 'HTTPS observation failed' })
})
test('HTTP is refused without echoing private URL data', async () => {
  await assert.rejects(helper.requestHTTPS('http://127.0.0.1/PRIVATE-URL'), { message: 'HTTPS observation failed' })
})
test('stalled HTTPS observation has a bounded deadline', async () => {
  const start = Date.now()
  await assert.rejects(helper.requestHTTPS(origin + '/slow', { ca: await readFile(tls.ca), timeout: 100 }), { message: 'HTTPS observation failed' })
  assert(Date.now() - start < 2000)
})
test('oversized HTTPS body is refused without returning private bytes', async () => {
  await assert.rejects(helper.requestHTTPS(origin + '/large', { ca: await readFile(tls.ca) }), { message: 'HTTPS observation failed' })
})
test('invalid successful JSON never appears in diagnostics', async () => {
  const response = await helper.requestHTTPS(origin + '/invalid', { ca: await readFile(tls.ca) })
  assert.throws(() => response.json(), { message: 'HTTPS response JSON invalid' })
})
