// SPDX-License-Identifier: Apache-2.0
// Disposable stopped-state verification only; caller must first stop its Controller.
import assert from 'node:assert/strict'
import { lstat, readFile, writeFile, rename, realpath } from 'node:fs/promises'
import { dirname, join, resolve } from 'node:path'
import { createHash } from 'node:crypto'
import { assertPrivateStateEqual, copyPrivateState, relocateInstanceConfig, relocateOIDCProvider, snapshotPrivateTree } from './controller-state-copy.mjs'

export async function restoreRealIdpInstance(instance, backup, target, retired) {
  try {
    const paths = [instance, backup, target, retired], parent = dirname(instance)
    assert.equal(new Set(paths).size,4)
    for (const path of paths) { assert.equal(path,resolve(path)); assert.equal(dirname(path),parent) }
    assert.equal(await realpath(parent),parent)
    assert.equal((await lstat(parent)).mode & 0o777,0o700)
    // Do not start a partial copy when any caller-owned destination already exists.
    for (const path of [backup,target,retired]) await assert.rejects(lstat(path),{code:'ENOENT'})
    const snapshot = await copyPrivateState(instance,backup)
    assertPrivateStateEqual(await copyPrivateState(backup,target),snapshot)
    const oldConfig = JSON.parse(await readFile(join(instance,'controller.yaml'),'utf8'))
    const configuration = relocateInstanceConfig(oldConfig,instance,target)
    const oldProvider = JSON.parse(await readFile(oldConfig.browser.oidc_config,'utf8'))
    const provider = relocateOIDCProvider(oldProvider,instance,target)
    await writeFile(join(target,'controller.yaml'),JSON.stringify(configuration),{mode:0o600})
    await writeFile(configuration.browser.oidc_config,JSON.stringify(provider),{mode:0o600})
    const after = await snapshotPrivateTree(target)
    assertPrivateStateEqual(Object.keys(after),Object.keys(snapshot))
    for (const path of Object.keys(snapshot).filter(x=>!['controller.yaml','identity/provider.json'].includes(x))) assert.equal(after[path],snapshot[path])
    assertPrivateStateEqual(provider,{...oldProvider,bindings:join(target,'identity/bindings.json')})
    assertPrivateStateEqual(await readFile(provider.bindings),await readFile(join(backup,'identity/bindings.json')))
    assert.equal(configuration.agent.authority,oldConfig.agent.authority)
    // Retain the stopped original for inspection, but make every old reference fail.
    await rename(instance,retired); await assert.rejects(lstat(instance),{code:'ENOENT'})
    return {instance:target,configuration,provider,proof:{completeFiles:Object.keys(snapshot).length,snapshotSHA256:createHash('sha256').update(JSON.stringify(snapshot)).digest('hex'),changedOnlyConfigPaths:true,originalUnavailable:true,bindingBytesPreserved:true,authorityRetained:true}}
  } catch { throw new Error('real IdP stopped restore failed') }
}

const only = (v, keys) => { assert(v&&typeof v==='object'&&!Array.isArray(v)); assert.deepEqual(Object.keys(v).sort(),[...keys].sort()) }
export function validateRealIdpRestoreEvidence(r,source) {
  only(r,['result','source','provider','runtimePATH','principal','revision','restore','proofs'])
  assert.equal(r.result,'pass');only(r.source,['commit','tree','platform']);assert.deepEqual(r.source,source)
  for(const field of ['commit','tree'])assert.match(r.source[field],/^[a-f0-9]{40}$/)
  assert.equal(r.source.platform,'linux-amd64');assert.equal(r.runtimePATH,'empty')
  assert.deepEqual(r.provider,{kind:'real-keycloak',version:'26.8.1',archiveSHA256:'4e2740b181faa2ea623a1c2ecf00855935311e546046e2f857615a47ed24cbe8',disposable:true})
  assert.match(r.principal,/^principal:account-[a-f0-9]{32}$/);assert.equal(r.revision,3)
  only(r.restore,['completeFiles','snapshotSHA256','changedOnlyConfigPaths','originalUnavailable','bindingBytesPreserved','authorityRetained'])
  assert(Number.isSafeInteger(r.restore.completeFiles)&&r.restore.completeFiles>=10);assert.match(r.restore.snapshotSHA256,/^[a-f0-9]{64}$/)
  for(const k of ['changedOnlyConfigPaths','originalUnavailable','bindingBytesPreserved','authorityRetained'])assert.equal(r.restore[k],true)
  const fields=['normalVerifiedTLS','providerRunningAcrossRestore','cleanStoppedBackup','durableViewsEqual','oldSessionRejected','freshSamePrincipalRoles','restoredPublicationAudit','withdrawalDenied','revokedCookieRejected','localRecoveryKeyRetained','ownedStateRemoved','ownedTrustRemoved']
  only(r.proofs,fields);for(const k of fields)assert.equal(r.proofs[k],true)
}
