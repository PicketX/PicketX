// SPDX-License-Identifier: Apache-2.0
export type HTTPAuditEvent = {
  id: number; caller: string; route: string; source: string; allowed: boolean
  reason: string; revision: number; grant_ids: string[]; deny_ids: string[]
  next_change: string; cache_ttl: number; at: string
}
export type HTTPAuditPage = { events: HTTPAuditEvent[]; next_after: number }

function object(value: unknown, names: string[]): value is Record<string, unknown> {
  return value !== null && typeof value === 'object' && Object.getPrototypeOf(value) === Object.prototype &&
    Object.keys(value).sort().join(',') === names.sort().join(',')
}
const integer = (value: unknown, minimum: number): value is number => typeof value === 'number' && Number.isSafeInteger(value) && value >= minimum
const text = (value: unknown): value is string => typeof value === 'string' && value.length > 0 && value.length <= 256
const ids = (value: unknown): value is string[] => Array.isArray(value) && value.length <= 4096 && value.every(text) && new Set(value).size === value.length
const time = (value: unknown): value is string => typeof value === 'string' && value.length <= 64 && /^\d{4}-\d\d-\d\dT/.test(value) && Number.isFinite(Date.parse(value))

export function parseHTTPAuditPage(input: unknown, after: number): HTTPAuditPage {
  const invalid = () => { throw new Error('Invalid HTTP decision page') }
  if (!integer(after, 0) || !object(input, ['events', 'next_after']) || !integer(input.next_after, 0) || !Array.isArray(input.events) || input.events.length > 50) return invalid()
  let cursor = after
  for (const event of input.events) {
    if (!object(event, ['id', 'caller', 'route', 'source', 'allowed', 'reason', 'revision', 'grant_ids', 'deny_ids', 'next_change', 'cache_ttl', 'at']) ||
      !integer(event.id, 1) || event.id <= cursor || !integer(event.revision, 1) || typeof event.allowed !== 'boolean' ||
      ![event.caller, event.route, event.source, event.reason].every(text) || !ids(event.grant_ids) || !ids(event.deny_ids) ||
      !time(event.at) || !time(event.next_change) || !integer(event.cache_ttl, 0)) return invalid()
    cursor = event.id
  }
  if (input.next_after !== cursor) return invalid()
  return input as HTTPAuditPage
}

// This is the last loaded window, never a total count or a complete-history claim.
export function appendHTTPAudit(previous: HTTPAuditEvent[], page: HTTPAuditPage): HTTPAuditEvent[] {
  return [...previous, ...page.events].slice(-200)
}
