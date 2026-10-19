// SPDX-License-Identifier: Apache-2.0
// Opt-in disposable-host acceptance only. Never installs a production unit.
import assert from 'node:assert/strict'
import { chmod, chown, lstat, readFile, readlink, unlink, writeFile } from 'node:fs/promises'
import { join } from 'node:path'
import { randomBytes, createHash } from 'node:crypto'
import { spawnSync } from 'node:child_process'

const pause = () => new Promise(r => setTimeout(r, 100))
function command(binary, args, check = true) {
  const result = spawnSync(binary, args, { encoding: 'utf8', timeout: 30000, maxBuffer: 4 << 20 })
  if (check) assert.equal(result.status, 0, `${binary} failed: ${result.stderr}`)
  return result
}
const quoted = value => {
  assert(!/[\r\n\0]/.test(value))
  return '"' + value.replaceAll('\\', '\\\\').replaceAll('"', '\\"').replaceAll('%', '%%') + '"'
}

export async function createControllerService({ temp, bundle, stateParent }) {
  assert(process.getuid() === 0 && process.env.PICKETX_SERVICE_TESTS === '1', 'real service opt-in/root required')
  assert(temp.startsWith('/run/picketx-service-'), 'owned /run fixture required')
  command('systemctl', ['show', '--property=Version']) // Fail, never skip, without actual manager.
  const user = 'px-test-' + randomBytes(6).toString('hex'), unit = user + '.service'
  const unitPath = '/run/systemd/system/' + unit
  let userCreated = false, unitCreated = false
  const ctl = (...args) => command('systemctl', args)
  const value = key => ctl('show', unit, '--property=' + key, '--value').stdout.trim()
  const proof = { status: 'pending', manager: command('systemd-analyze', ['--version']).stdout.split('\n')[0], processes: [], crashRestarted: false }
  const expectedBinary = join(bundle, 'bin/picketx-controller')
  const binarySHA256 = createHash('sha256').update(await readFile(expectedBinary)).digest('hex')
  let uid, gid
  async function inspect() {
    const pid = Number(value('MainPID')); assert(pid > 0)
    const proc = await readFile(`/proc/${pid}/status`, 'utf8')
    const actualUID = proc.match(/^Uid:\s+(\d+)/m)?.[1]
    const capabilities = proc.match(/^CapEff:\s+(\w+)/m)?.[1]
    const noNewPrivileges = Number(proc.match(/^NoNewPrivs:\s+(\d+)/m)?.[1])
    assert.equal(Number(actualUID), uid); assert.notEqual(uid, 0)
    assert.equal(BigInt('0x' + capabilities), 0n); assert.equal(noNewPrivileges, 1)
    assert.equal(await readlink(`/proc/${pid}/exe`), expectedBinary)
    assert.equal(createHash('sha256').update(await readFile(`/proc/${pid}/exe`)).digest('hex'), binarySHA256)
    assert.equal(value('ProtectSystem'), 'strict')
    assert.equal(value('NoNewPrivileges'), 'yes')
    assert.equal(value('Restart'), 'on-failure')
    const result = { pid, uid, capabilities, noNewPrivileges, binarySHA256 }
    proof.processes.push(result)
    return pid
  }
  async function stop() {
    if (!unitCreated) return
    ctl('stop', unit)
    assert.equal(value('MainPID'), '0'); assert.equal(value('ActiveState'), 'inactive')
    assert.equal(value('Result'), 'success'); assert.equal(value('ExecMainStatus'), '0')
  }
  async function close() {
    // Never delete state/user while a service might still be running.
    await stop()
    if (unitCreated) {
      await unlink(unitPath); unitCreated = false
      ctl('daemon-reload')
      command('systemctl', ['reset-failed', unit], false)
    }
    if (userCreated) {
      command('userdel', [user]); userCreated = false
      if (command('getent', ['group', user], false).status === 0) command('groupdel', [user])
    }
  }
  try {
    command('useradd', ['--system', '--no-create-home', '--user-group', '--home-dir', '/nonexistent', '--shell', '/usr/sbin/nologin', user])
    userCreated = true
    uid = Number(command('id', ['-u', user]).stdout.trim())
    gid = Number(command('id', ['-g', user]).stdout.trim())
    await chmod(temp, 0o755)
    await chown(stateParent, uid, gid); await chmod(stateParent, 0o700)
    const template = await readFile(join(bundle, 'deploy/picketx-controller.service'), 'utf8')
    return {
      identity: { uid, gid }, proof,
      async start(instance) {
        if (unitCreated) assert.equal(value('MainPID'), '0', 'stop before changing service paths')
        let service = template
        for (const [pattern, replacement] of [
          [/^User=picketx$/m, 'User=' + user], [/^Group=picketx$/m, 'Group=' + user],
          [/^WorkingDirectory=.+$/m, 'WorkingDirectory=' + quoted(instance)],
          [/^ExecStart=.+$/m, 'ExecStart=' + quoted(expectedBinary) + ' -config ' + quoted(join(instance, 'controller.yaml'))],
          [/^ReadWritePaths=.+$/m, 'ReadWritePaths=' + quoted(instance)],
        ]) {
          assert(pattern.test(service), 'shipped unit contract changed')
          service = service.replace(pattern, replacement)
        }
        service = service.replace('[Service]\n', '[Service]\nEnvironment=PATH=\n')
        if (unitCreated) {
          const info = await lstat(unitPath); assert(info.isFile() && info.uid === 0)
        }
        await writeFile(unitPath, service, { flag: unitCreated ? 'w' : 'wx', mode: 0o600 })
        unitCreated = true
        command('systemd-analyze', ['verify', '--man=no', '--generators=no', unitPath])
        ctl('daemon-reload'); ctl('start', unit)
        assert.equal(value('ActiveState'), 'active'); assert.equal(value('Type'), 'simple')
        await inspect()
      },
      stop,
      async crashRestart() {
        const old = Number(value('MainPID')); assert(old > 0)
        ctl('kill', '--signal=SIGKILL', '--kill-whom=main', unit)
        for (let n = 0; n < 200; n++) {
          const pid = Number(value('MainPID'))
          if (pid > 0 && pid !== old && value('ActiveState') === 'active') {
            assert(Number(value('NRestarts')) >= 1); await inspect()
            proof.crashRestarted = true; return
          }
          await pause()
        }
        throw new Error('actual systemd automatic restart deadline')
      },
      logs() { return command('journalctl', ['--unit=' + unit, '--no-pager', '--output=cat']).stdout },
      close,
    }
  } catch (error) {
    await close()
    throw error
  }
}
