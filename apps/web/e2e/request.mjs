// SPDX-License-Identifier: Apache-2.0
// Exercise the real transport helper; only its external fetch boundary is faked.
import assert from 'node:assert/strict'
import { request, APIError } from '../src/lib/console.ts'

const original = globalThis.fetch
try {
  for (const expected of [200, 201]) {
    for (const actual of [200, 201, 202, 203, 204, 205, 206]) {
      if (actual === expected) continue
      globalThis.fetch = async () => new Response([204, 205].includes(actual) ? null : '{"uid":"target","revision":2}', { status: actual, headers: { ETag: '"2"' } })
      await assert.rejects(request('/resources', { method: 'POST' }, expected), e => e instanceof Error && !(e instanceof APIError), `expected ${expected} must reject ${actual} as an ambiguous result`)
    }
    globalThis.fetch = async () => new Response('{"uid":"target","revision":2}', { status: expected, headers: { ETag: '"2"' } })
    assert.deepEqual(await request('/resources', { method: 'POST' }, expected), { data: { uid: 'target', revision: 2 }, etag: '"2"', status: expected })
  }
  globalThis.fetch = async () => new Response(null, { status: 204 })
  assert.deepEqual(await request('/logout', { method: 'POST' }, 204), { data: undefined, etag: '', status: 204 })
  for (const status of [401, 403, 412, 500]) {
    globalThis.fetch = async () => new Response('{"code":"expected_error"}', { status })
    await assert.rejects(request('/publication', { method: 'PUT' }, 200), e => e instanceof APIError && e.status === status && e.message === 'expected_error')
  }
  globalThis.fetch = async () => new Response('{"revision":1}', { status: 200, headers: { ETag: '"1"' } })
  assert.deepEqual(await request('/publication'), { data: { revision: 1 }, etag: '"1"', status: 200 })
  console.log('PASS real request helper exact completion status, ambiguous 2xx, APIError, GET and explicit 204')
} finally { globalThis.fetch = original }
