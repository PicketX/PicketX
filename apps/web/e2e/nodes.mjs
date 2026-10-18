// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict'
import './node-freshness.mjs'
import { writeFile } from 'node:fs/promises'
import { join } from 'node:path'
import { createHash, randomBytes } from 'node:crypto'

export async function nodeObservations({ operator, view, temp, evidence }) {
  await operator.getByRole('button', { name: 'Nodes', exact: true }).click()
  await operator.getByText('No credentialed Agent scopes.', { exact: true }).waitFor()
  const token = randomBytes(32).toString('hex')
  await writeFile(join(temp, 'agent.json'), JSON.stringify({ credentials: [{ id: 'node-observer', agent_ref: 'agent-a', sha256: createHash('sha256').update(token).digest('hex') }] }), { mode: 0o600 })
  // Real credential reload, not a fabricated status response.
  await operator.getByRole('button', { name: 'Refresh nodes', exact: true }).click()
  await operator.getByRole('heading', { name: 'agent-a', exact: true }).waitFor({ timeout: 30000 })
  await operator.getByText('No report received', { exact: true }).waitFor()
  await operator.getByText('Disconnected', { exact: true }).waitFor()
  assert.equal(await operator.getByText('Agent reported applied', { exact: true }).count(), 0)
  await operator.screenshot({ path: join(evidence, 'nodes-desktop.png'), fullPage: true })
  await operator.route('**/console/api/v1/agent-observations', route => route.abort())
  await operator.getByRole('button', { name: 'Refresh nodes', exact: true }).click()
  await operator.getByText('Refresh failed. Retained observations are historical.', { exact: true }).waitFor()
  await operator.getByRole('heading', { name: 'agent-a', exact: true }).waitFor()
  await operator.setViewportSize({ width: 375, height: 812 })
  assert(await operator.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
  await operator.screenshot({ path: join(evidence, 'nodes-mobile-error.png'), fullPage: true })
  await operator.unroute('**/console/api/v1/agent-observations')
  await operator.getByRole('button', { name: 'Desired configuration', exact: true }).click()
  await operator.getByRole('button', { name: 'Nodes', exact: true }).click()
  await operator.getByRole('heading', { name: 'agent-a', exact: true }).waitFor()
  assert.equal(await operator.getByText('Refresh failed. Retained observations are historical.', { exact: true }).count(), 0)
  await view.getByRole('button', { name: 'Nodes', exact: true }).click()
  await view.getByRole('heading', { name: 'agent-a', exact: true }).waitFor()
  await view.getByText('No report received', { exact: true }).waitFor()
  await view.screenshot({ path: join(evidence, 'nodes-viewer-mobile.png'), fullPage: true })
  await operator.setViewportSize({ width: 1440, height: 1000 })
  console.log('PASS real Nodes API/credential reload → browser empty/unreported/disconnected, failed refresh/history, re-entry, viewer/mobile; no kernel assertion')
}
