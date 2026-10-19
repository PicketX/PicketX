// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict'
import { test } from 'node:test'
import { mkdtemp, readFile, readlink, realpath, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { spawn, spawnSync } from 'node:child_process'
import { once } from 'node:events'
import { renderControllerUnit, waitForControllerIdentity } from './controller-service-fixture.mjs'
import { durableServiceView } from './controller-service-evidence.mjs'

test('shipped unit with space-containing relocated paths passes actual systemd parser', async t => {
  const temp = await mkdtemp(join(tmpdir(), 'picketx-unit-test-'))
  t.after(() => rm(temp, { recursive: true, force: true }))
  const template = await readFile(new URL('../deploy/controller/picketx-controller.service', import.meta.url), 'utf8')
  const unit = renderControllerUnit(template, { user: 'nobody', binary: '/bin/true', instance: join(temp, 'with space') })
  const path = join(temp, 'picketx-test.service'); await writeFile(path, unit)
  const result = spawnSync('systemd-analyze', ['verify', '--man=no', '--generators=no', path], { encoding: 'utf8' })
  assert.equal(result.status, 0, result.stderr)
})
const accounts = observedAt => ({ mode: 'database', revision: 1, accounts: [{ id: 'principal:account-fixed', username: 'operator', roles: ['role:admin'], version: 1, disabled: false }], events: [{ revision: 1, actor: 'principal:bootstrap', accountID: 'principal:account-fixed', operation: 'create', reason: 'Created operator', at: '2026-10-19T17:00:00Z' }], observedAt })
test('accounts restore ignores only changing read-time observation, not durable events', () => {
  const before = durableServiceView('accounts', accounts('2026-10-19T17:00:01Z'))
  const after = durableServiceView('accounts', accounts('2026-10-19T17:01:01Z'))
  assert.deepEqual(after, before)
  for (const mutate of [c => c.revision++, c => c.accounts[0].disabled = true, c => c.events[0].at = '2026-10-19T17:01:00Z']) {
    const changed = accounts('2026-10-19T17:02:00Z'); mutate(changed)
    assert.notDeepEqual(durableServiceView('accounts', changed), before)
  }
})
test('accounts restore rejects absent or invalid observation metadata', () => {
  for (const at of [undefined, '', 'not-a-time']) assert.throws(() => durableServiceView('accounts', accounts(at)))
})
async function transitionProcess(t) {
  // Shell builtins open their own procfs status, so this works when the procfs
  // mount exposes an outer PID namespace and /proc/<tid>/children is unavailable.
  const child = spawn('/bin/sh', ['-c', 'while read -r key value; do case "$key" in Tgid:) printf "%s\\n" "$value"; break;; esac; done < /proc/self/status; read -r value; exec /bin/sleep 30'], { stdio: ['pipe', 'pipe', 'ignore'] })
  await once(child, 'spawn')
  t.after(async () => { const exit = once(child, 'exit'); child.kill('SIGKILL'); await exit })
  const [bytes] = await once(child.stdout, 'data'), pid = bytes.toString().trim()
  assert.match(pid, /^[1-9]\d*$/)
  return { child, proc: `/proc/${pid}` }
}
test('identity barrier waits for actual process exec rather than active parent identity', async t => {
  const { child, proc } = await transitionProcess(t), expected = await realpath('/bin/sleep')
  let attempts = 0
  const pid = await waitForControllerIdentity(async () => {
    const executable = await readlink(proc + '/exe')
    if (++attempts === 1) child.stdin.end('continue\n')
    assert.equal(executable, expected, 'real child has not completed exec')
    return child.pid
  })
  assert.equal(pid, child.pid); assert(attempts >= 2)
})
test('identity barrier fails closed within bounded time for a persistently wrong real process', async t => {
  const { proc } = await transitionProcess(t), expected = await realpath('/bin/sleep')
  let attempts = 0
  await assert.rejects(waitForControllerIdentity(async () => {
    attempts++; assert.equal(await readlink(proc + '/exe'), expected)
  }, 200))
  assert(attempts >= 2)
})
