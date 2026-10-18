// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict'

// Fault injection only: suppress one outgoing write and supply an ambiguous
// receipt. Everything around it (built React, reads, later writes) remains real.
export async function rejectedMutationReceipt({ p, path, section, submit, labels, status = 202, abort = false }) {
  const snapshot = () => p.evaluate(async () => {
    const r = await fetch('/console/api/v1/publication')
    return { etag: r.headers.get('etag'), data: await r.json() }
  })
  const before = await snapshot()
  const values = await Promise.all(labels.map(label => p.getByLabel(label, { exact: true }).inputValue()))
  const body = path === '/publication' ? { ...before.data, revision: before.data.revision + 1 } : { uid: path.split('/').at(-2) || 'uncommitted', revision: before.data.revision + 1 }
  const pattern = `**/console/api/v1${path}`
  await p.route(pattern, r => {
    if (!['POST', 'PUT'].includes(r.request().method())) return r.continue()
    return abort ? r.abort() : r.fulfill({ status, contentType: 'application/json', headers: { ETag: `"${before.data.revision + 1}"` }, body: JSON.stringify(body) })
  })
  try {
    await p.getByRole('button', { name: submit, exact: true }).click()
    await p.locator(`${section} [role="alert"]`).waitFor({ timeout: 5000 })
    await p.waitForFunction(selector => document.querySelector(selector)?.getAttribute('aria-busy') === 'false', section)
    for (const [i, label] of labels.entries()) assert.equal(await p.getByLabel(label, { exact: true }).inputValue(), values[i], `${path}: ambiguous receipt must preserve ${label}`)
    assert(await p.getByRole('button', { name: submit, exact: true }).isDisabled(), `${path}: reconcile before another write`)
    if (path === '/publication') await p.locator('.editor-section .stale').waitFor()
    assert.equal(await p.locator(`${section} .feedback.success:visible`).count(), 0, `${path}: no completion claim`)
    assert.deepEqual(await snapshot(), before, `${path}: intercepted write did not alter real Controller state`)
    console.log(`PASS ${path}: ${abort ? 'transport failure' : status} retains draft, no completion, retry blocked, real Controller unchanged`)
  } finally { await p.unroute(pattern) }
}
