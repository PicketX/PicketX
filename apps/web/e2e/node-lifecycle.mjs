// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict'

const endpoint = '**/console/api/v1/agent-observations'
const isObservation = request => new URL(request.url()).pathname === '/console/api/v1/agent-observations'

export async function nodeLifecycle({ operator }) {
  // Copy the real authenticated session into an isolated cookie jar: the deliberate
  // 401 must not clear cookies used by the later credential-revocation acceptance.
  const context = await operator.context().browser().newContext({ storageState: await operator.context().storageState() })
  const page = await context.newPage()
  const errors = []
  page.on('pageerror', error => errors.push(error.message))
  let reads = 0
  page.on('request', request => { if (isObservation(request)) reads++ })
  const refresh = () => page.getByRole('button', { name: 'Refresh nodes', exact: true }).click()
  const settled = () => page.waitForFunction(() => [...document.querySelectorAll('button')]
    .some(button => button.textContent === 'Refresh nodes' && !button.disabled), { }, { polling: 50 })
  const nextRead = () => page.waitForRequest(isObservation, { timeout: 3000 })
  // Headless deterministic visibility fault injection, not an OS-backgrounding claim.
  const hidden = value => page.evaluate(value => {
    Object.defineProperty(document, 'hidden', { configurable: true, value })
    document.dispatchEvent(new Event('visibilitychange'))
  }, value)
  const resume = async () => { await hidden(true); await hidden(false) }
  async function holdNext({ unauthenticated = false } = {}) {
    const ready = Promise.withResolvers(), release = Promise.withResolvers()
    const done = Promise.withResolvers()
    await page.route(endpoint, async route => {
      try {
        const headers = { ...route.request().headers() }
        if (unauthenticated) headers.cookie = ''
        const response = await route.fetch({ headers })
        assert.equal(response.status(), unauthenticated ? 401 : 200)
        ready.resolve()
        const action = await release.promise
        if (action === 'abort') await route.abort()
        else await route.fulfill({ response })
        done.resolve()
      } catch (error) { ready.reject(error); done.reject(error) }
    }, { times: 1 })
    return { ready: ready.promise, release: async action => { release.resolve(action); await done.promise } }
  }
  try {
    await page.clock.install()
    await page.goto(operator.url())
    await page.getByRole('heading', { name: 'Desired configuration', exact: true }).waitFor()
    await page.getByRole('button', { name: 'Nodes', exact: true }).click()
    await page.getByRole('heading', { name: 'agent-a', exact: true }).waitFor()
    await settled()
    await page.clock.pauseAt(await page.evaluate(() => Date.now() + 100))

    // Multiple resumes while an actual API response is delayed must coalesce.
    let start = reads
    let held = await holdNext()
    await refresh(); await held.ready
    await resume(); await resume(); await resume()
    assert.equal(reads, start + 1, 'no overlap while an observation is in flight')
    let followup = nextRead()
    await held.release()
    await followup // RED on PR21: no request until the paused 15-second poll.
    await settled()
    await page.clock.runFor(1000)
    assert.equal(reads, start + 2, 'resume events coalesce into exactly one follow-up')

    // Re-hiding before settlement must not drain the queued work in the background.
    start = reads; held = await holdNext()
    await refresh(); await held.ready; await resume(); await hidden(true)
    await held.release(); await settled()
    await page.clock.runFor(15000)
    assert.equal(reads, start + 1, 'hidden polling and queued refresh stay paused')
    followup = nextRead(); await hidden(false); await followup; await settled()
    assert.equal(reads, start + 2)

    // A transport failure still drains one queued resume and recovers from history.
    start = reads; held = await holdNext()
    await refresh(); await held.ready; await resume()
    followup = nextRead(); await held.release('abort'); await followup; await settled()
    assert.equal(reads, start + 2)
    assert.equal(await page.getByText('Refresh failed. Retained observations are historical.', { exact: true }).count(), 0)

    // Unmount aborts the old request and its queued work, even after a fresh mount.
    start = reads; held = await holdNext()
    await refresh(); await held.ready; await resume()
    const aborted = page.waitForEvent('requestfailed', { predicate: isObservation, timeout: 3000 })
    await page.getByRole('button', { name: 'Desired configuration', exact: true }).click()
    await aborted
    await page.getByRole('button', { name: 'Nodes', exact: true }).click()
    await page.getByRole('heading', { name: 'agent-a', exact: true }).waitFor(); await settled()
    await held.release(); await page.clock.runFor(1000)
    assert.equal(reads, start + 2, 'disposed effect cannot launch its queued request')
    assert.equal(await page.getByText('Refresh failed. Retained observations are historical.', { exact: true }).count(), 0)

    // Actual Controller 401 (cookie removed from the forwarded read), not a fake status.
    start = reads; held = await holdNext({ unauthenticated: true })
    await refresh(); await held.ready; await resume(); await held.release()
    await page.getByRole('button', { name: 'Sign in', exact: true }).waitFor()
    await page.clock.runFor(15000)
    assert.equal(reads, start + 1, '401 discards queued work before logout cleanup')
    assert.deepEqual(errors, [])
    console.log('PASS real Nodes responses with controlled delay/visibility/clock: coalesced resume, hidden pause, retry, unmount abort, actual 401; no OS/background or kernel assertion')
  } finally { await context.close() }
}
