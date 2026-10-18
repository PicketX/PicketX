// SPDX-License-Identifier: Apache-2.0
export type Identity = { id: string; roles: string[]; operations: string[] }
export type Publication = { authority: string; revision: number; policy: string; facts: string }
export type AuditEvent = { id: number; actor: string; reason: string; revision: number; digest: string; at: string }
export type AuditPage = { events: AuditEvent[]; next_after: number }
export type Session = { authenticated: boolean; csrf: string }
export class APIError extends Error {
  status: number
  constructor(status: number, code: string) { super(code); this.status = status }
}
export async function request<T>(path: string, init: RequestInit = {}, expectedStatus?: number): Promise<{ data: T; etag: string; status: number }> {
  const signal = init.signal ? AbortSignal.any([init.signal, AbortSignal.timeout(15000)]) : AbortSignal.timeout(15000)
  const response = await fetch(`/console/api/v1${path}`, { ...init, credentials: 'same-origin', cache: 'no-store', signal })
  if (!response.ok) {
    let code = 'request_failed'
    try { const body = await response.json(); if (typeof body.code === 'string') code = body.code } catch { /* non-JSON transport/Origin rejection */ }
    throw new APIError(response.status, code)
  }
  // A 2xx alone (notably 202 Accepted) is not proof that a mutation completed.
  // Keep this an ambiguous transport result, not a server-declared APIError:
  // write callers must retain intent and reconcile before allowing a retry.
  if (expectedStatus !== undefined && response.status !== expectedStatus) throw new Error('Unexpected completion status')
  if (response.status === 204) return { data: undefined as T, etag: '', status: response.status }
  return { data: await response.json() as T, etag: response.headers.get('etag') ?? '', status: response.status }
}
