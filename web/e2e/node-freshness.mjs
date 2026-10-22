// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict'
import { observationAge, reportIsFresh } from '../src/features/nodes/freshness.ts'

assert.equal(observationAge({ wall: 100, monotonic: 200 }, { wall: 50, monotonic: 300 }), 100)
assert.equal(observationAge({ wall: 100, monotonic: 200 }, { wall: 70100, monotonic: 201 }), 70000)
const current = { freshness: 'fresh', fresh_for_millis: 1000 }
assert.equal(reportIsFresh(current, 999, false), true)
assert.equal(reportIsFresh(current, 1000, false), false)
assert.equal(reportIsFresh(current, 0, true), false)
assert.equal(reportIsFresh({ ...current, freshness: 'stale' }, 0, false), false)
assert.equal(reportIsFresh({ ...current, fresh_for_millis: 0 }, 0, false), false)
console.log('PASS browser freshness boundary, request latency, failed-read history and clock suspension/skew')
