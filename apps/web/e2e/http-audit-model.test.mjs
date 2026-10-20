// SPDX-License-Identifier: Apache-2.0
import test from 'node:test'
import assert from 'node:assert/strict'
import { parseHTTPAuditPage, appendHTTPAudit } from '../src/features/http-audit/model.ts'

// Unit data only; browser acceptance reads actual persisted decisions separately.
const event = id => ({ id, caller: 'nginx', route: 'route-v1', source: '2001:db8::1', allowed: false,
  reason: 'explicit-deny', revision: 4, grant_ids: [], deny_ids: ['deny-v1'],
  next_change: '0001-01-01T00:00:00Z', cache_ttl: 0, at: '2026-10-20T11:00:00Z' })
test('parses actual-shaped IPv6/deny records without deriving current permissions', () => {
  const input = { events: [event(11)], next_after: 11 }
  assert.deepEqual(parseHTTPAuditPage(input, 10), input)
})
test('empty tail keeps its exclusive cursor and earlier records', () => {
  const previous = [event(11)], page = parseHTTPAuditPage({ events: [], next_after: 11 }, 11)
  assert.deepEqual(appendHTTPAudit(previous, page), previous)
})
test('bounded window drops oldest loaded records while retaining newest actual IDs', () => {
  const previous = Array.from({ length: 190 }, (_, i) => event(i + 1))
  const page = parseHTTPAuditPage({ events: Array.from({ length: 50 }, (_, i) => event(i + 191)), next_after: 240 }, 190)
  const result = appendHTTPAudit(previous, page)
  assert.equal(result.length, 200); assert.equal(result[0].id, 41); assert.equal(result.at(-1).id, 240)
  assert.equal(previous[0].id, 1); assert.equal(previous.length, 190)
})
for (const [name, mutate, after = 10] of [
  ['null page', () => null], ['missing records', p => { delete p.events }],
  ['more than one API page', p => { p.events = Array.from({ length: 51 }, (_, i) => event(i + 11)); p.next_after = 61 }],
  ['duplicate record', p => { p.events.push(event(11)) }], ['old cursor record', p => { p.events[0].id = 10 }],
  ['backwards records', p => { p.events = [event(12), event(11)] }], ['wrong next cursor', p => { p.next_after = 12 }],
  ['empty advancing cursor', p => { p.events = []; p.next_after = 12 }],
  ['unsafe cursor', p => { p.next_after = Number.MAX_SAFE_INTEGER + 1 }], ['negative request cursor', p => p, -1],
  ['unsafe event ID', p => { p.events[0].id = Number.MAX_SAFE_INTEGER + 1 }],
  ['fake allowed string', p => { p.events[0].allowed = 'false' }], ['missing boolean', p => { delete p.events[0].allowed }],
  ['invalid time', p => { p.events[0].at = 'not-time' }], ['zero revision', p => { p.events[0].revision = 0 }],
  ['unsafe revision', p => { p.events[0].revision = Number.MAX_SAFE_INTEGER + 1 }],
  ['non-string caller', p => { p.events[0].caller = {} }], ['oversize source', p => { p.events[0].source = 'x'.repeat(257) }],
  ['missing IDs', p => { delete p.events[0].deny_ids }], ['bad ID list', p => { p.events[0].grant_ids = [''] }],
  ['oversized contributing list', p => { p.events[0].deny_ids = Array.from({ length: 4097 }, (_, i) => `deny-${i}`) }],
  ['duplicate contributing IDs', p => { p.events[0].grant_ids = ['grant-1', 'grant-1'] }],
  ['secret extra page field', p => { p.credentials = 'private-unit-marker' }],
  ['private extra event field', p => { p.events[0].uri = '/private-unit-marker' }],
]) test(`rejects ${name} with fixed error and no input disclosure`, () => {
  let input = { events: [event(11)], next_after: 11 }; input = mutate(input) ?? input
  // Deliberately distinguish a null mutation from in-place edits.
  if (name === 'null page') input = null
  assert.throws(() => parseHTTPAuditPage(input, after), e => e.message === 'Invalid HTTP decision page')
})
