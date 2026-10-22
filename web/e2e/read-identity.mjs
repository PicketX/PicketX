// SPDX-License-Identifier: Apache-2.0
// Self-contained so Playwright serializes this exact reader into the browser.
export async function readIdentity({ url, timeoutMs = 15000 }, signal) {
 const signals = [AbortSignal.timeout(timeoutMs)]
 if (signal) signals.push(signal)
 const response = await fetch(url, { cache: 'no-store', signal: AbortSignal.any(signals) })
 if (response.status !== 200) return { status: response.status }
 let body
 try { body = await response.json() }
 catch { throw new Error('invalid identity JSON') }
 if (!body || typeof body.id !== 'string' || !Array.isArray(body.operations) ||
     !body.operations.every(operation => typeof operation === 'string')) {
  throw new Error('invalid identity operations')
 }
 return { status: response.status, id: body.id, operations: body.operations }
}
