// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict'

// Feature fixtures use the existing API; token sign-in has no browser UI.
export async function tokenSession(page, token) {
  const api = page.context().request
  const base = new URL('/console/api/v1', page.url()).href
  const session = await api.get(`${base}/session`)
  assert.equal(session.status(), 200)
  const { csrf } = await session.json()
  const response = await api.post(`${base}/login`, {
    headers: { 'X-CSRF-Token': csrf, Origin: new URL(page.url()).origin },
    data: { token },
  })
  assert.equal(response.status(), 200)
  await page.reload()
}
