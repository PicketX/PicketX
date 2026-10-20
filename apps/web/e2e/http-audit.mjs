// SPDX-License-Identifier: Apache-2.0
// Success assertions read the actual Controller/SQLite, not response fixtures.
import assert from 'node:assert/strict'
import { join } from 'node:path'

export async function initialHTTPAudit({ page, evidence }) {
  await page.getByRole('button', { name: 'HTTP decisions', exact: true }).click({ timeout: 5000 })
  const section = page.locator('.http-audit-section')
  await section.getByRole('heading', { name: 'HTTP authorization decisions', exact: true }).waitFor()
  await section.getByText('No HTTP decisions have been recorded.', { exact: true }).waitFor()
  assert.equal(await section.locator('[data-http-decision-id]').count(), 0)
  for (const width of [1440, 375]) {
    await page.setViewportSize({ width, height: 1000 })
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
    await page.screenshot({ path: join(evidence, `http-audit-empty-${width}.png`), fullPage: true })
  }
  await page.getByRole('button', { name: 'Publication history', exact: true }).click()
}
