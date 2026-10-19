// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict'
export function durableServiceView(path, value) {
  const output = structuredClone(value)
  if (path === 'accounts') {
    assert(typeof output.observedAt === 'string' && /^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d{1,9})?Z$/.test(output.observedAt) && Number.isFinite(Date.parse(output.observedAt)), 'valid read-time observation required')
    delete output.observedAt // Every other durable field remains exact, including event timestamps.
  }
  return output
}
