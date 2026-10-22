// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict'
import { createServer } from 'node:http'
import { once } from 'node:events'
import { setTimeout as pause } from 'node:timers/promises'
import test from 'node:test'
import { waitForOperation } from './wait-for-operation.mjs'
import { readIdentity } from './read-identity.mjs'

async function fixture(t, respond) {
 let requests = 0, reads = 0
 const server = createServer((request, response) => respond(response, ++requests, request))
 server.listen(0, '127.0.0.1'); await once(server, 'listening')
 t.after(async () => { server.closeAllConnections(); await new Promise(resolve => server.close(resolve)) })
 const url = `http://127.0.0.1:${server.address().port}/me`
 const read = async signal => {
  reads++
  return readIdentity({ url }, signal)
 }
 return { read, requests: () => requests, reads: () => reads }
}
const json = (response, status, operations) => {
 response.writeHead(status, { 'content-type': 'application/json' })
 response.end(JSON.stringify({ id: 'principal:creator', operations, token: 'must-not-appear-in-errors' }))
}
test('awaits actual HTTP responses until the operation exists, not an async false value', async t => {
 const f = await fixture(t, (r, n) => json(r, 200, n < 3 ? [] : ['request:create']))
 assert.equal(await waitForOperation(f.read, 'request:create', { timeoutMs: 2000, pollMs: 5 }), true)
 assert.equal(f.requests(), 3)
 await pause(20); assert.equal(f.requests(), 3, 'polling stops at the first permitted observation')
})
test('rejects an absent operation at a bounded deadline and stops polling', async t => {
 const f = await fixture(t, r => json(r, 200, []))
 await assert.rejects(waitForOperation(f.read, 'request:create', { timeoutMs: 500, pollMs: 5 }), /operation readiness timed out/)
 const stopped = f.reads(); assert(f.requests() > 0)
 // A request started before cancellation can reach the server after the caller
 // rejects. The contract is no NEW reads, not retroactive packet withdrawal.
 await pause(25); assert.equal(f.reads(), stopped)
 assert(f.requests() <= stopped)
})
test('HTTP failures are errors, never retryable missing permissions', async t => {
 for (const status of [202, 401, 429, 503]) await t.test(String(status), async t => {
  const f = await fixture(t, r => json(r, status, []))
  await assert.rejects(waitForOperation(f.read, 'request:create', { timeoutMs: 2000, pollMs: 5 }), error => {
   assert.match(error.message, /unexpected identity status/)
   assert(!error.message.includes('must-not-appear-in-errors')); return true
  })
  assert.equal(f.requests(), 1)
 })
})
test('malformed successful identity data is rejected without payload disclosure', async t => {
 const f = await fixture(t, r => json(r, 200, null))
 await assert.rejects(waitForOperation(f.read, 'request:create'), /invalid identity operations/)
 assert.equal(f.requests(), 1)
})
test('deadline aborts an actual pending HTTP observation', { timeout: 2000 }, async t => {
 const f = await fixture(t, () => {})
 await assert.rejects(waitForOperation(f.read, 'request:create', { timeoutMs: 500, pollMs: 5 }), /operation readiness timed out/)
 assert.equal(f.requests(), 1)
})
test('invalid JSON never exposes response bytes in error messages or stacks', async t => {
 const f = await fixture(t, r => { r.writeHead(200); r.end('secret-token-example invalid-json') })
 await assert.rejects(waitForOperation(f.read, 'request:create'), error => {
  assert.match(error.message, /^invalid identity JSON$/)
  assert(!error.stack.includes('secret-token')); assert.equal(error.cause, undefined); return true
 })
 assert.equal(f.requests(), 1)
})
test('HTTP failure does not parse a malformed response body', async t => {
 const f = await fixture(t, r => { r.writeHead(503); r.end('secret-token-example invalid-json') })
 await assert.rejects(waitForOperation(f.read, 'request:create'), error => {
  assert.match(error.message, /^unexpected identity status 503$/)
  assert(!error.stack.includes('secret-token')); return true
 })
 assert.equal(f.requests(), 1)
})
