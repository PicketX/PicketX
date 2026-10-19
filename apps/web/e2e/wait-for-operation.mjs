// SPDX-License-Identifier: Apache-2.0
import { setTimeout as pause } from 'node:timers/promises'

// Test synchronization, never authorization: only the real /me result establishes
// readiness. Async predicates inside the pinned waitForFunction do not repoll.
export async function waitForOperation(readIdentity, operation, { timeoutMs = 45000, pollMs = 100 } = {}) {
 if (!Number.isFinite(timeoutMs) || timeoutMs <= 0 || !Number.isFinite(pollMs) || pollMs <= 0) throw new Error('invalid readiness deadline')
 const controller = new AbortController()
 let timer
 const expired = new Promise((_, reject) => {
  timer = setTimeout(() => {
   reject(new Error('operation readiness timed out'))
   controller.abort()
  }, timeoutMs)
 })
 try {
  for (;;) {
   const identity = await Promise.race([readIdentity(controller.signal), expired])
   if (identity?.status !== 200) throw new Error(`unexpected identity status ${Number.isInteger(identity?.status) ? identity.status : 'unknown'}`)
   if (!Array.isArray(identity.operations) || !identity.operations.every(x => typeof x === 'string')) throw new Error('invalid identity operations')
   if (identity.operations.includes(operation)) return true
   // No transport/auth/429 retries. A valid200 still lacking a propagated
   // operation is the ONLY polling case; both I/O and delay share one deadline.
   await Promise.race([pause(pollMs, undefined, { signal: controller.signal }), expired])
  }
 } finally {
  clearTimeout(timer)
  controller.abort()
 }
}
