// SPDX-License-Identifier: Apache-2.0
// Real filesystem and synthetic receipt tests; not real IdP/browser proof.
import assert from 'node:assert/strict'
import test from 'node:test'
import { mkdtemp, mkdir, writeFile, readFile, lstat, rm, symlink } from 'node:fs/promises'
import { join } from 'node:path'
import { tmpdir } from 'node:os'
let helper = {}
try { helper = await import('./real-idp-restore.mjs') } catch (e) { if (e.code !== 'ERR_MODULE_NOT_FOUND') throw e }
const source = { commit: 'a'.repeat(40), tree: 'b'.repeat(40), platform: 'linux-amd64' }
const proofs = Object.fromEntries(['normalVerifiedTLS','providerRunningAcrossRestore','cleanStoppedBackup','durableViewsEqual','oldSessionRejected','freshSamePrincipalRoles','restoredPublicationAudit','withdrawalDenied','revokedCookieRejected','localRecoveryKeyRetained','ownedStateRemoved','ownedTrustRemoved'].map(k => [k,true]))
function receipt() {
  return { result:'pass', source:{...source}, provider:{kind:'real-keycloak',version:'26.8.1',archiveSHA256:'4e2740b181faa2ea623a1c2ecf00855935311e546046e2f857615a47ed24cbe8',disposable:true},runtimePATH:'empty',principal:'principal:account-'+'1'.repeat(32),revision:3,
    restore:{completeFiles:10,snapshotSHA256:'c'.repeat(64),changedOnlyConfigPaths:true,originalUnavailable:true,bindingBytesPreserved:true,authorityRetained:true},proofs:{...proofs} }
}
test('public gate accepts complete correlated restored-instance evidence',()=>{
  assert.equal(typeof helper.validateRealIdpRestoreEvidence,'function'); helper.validateRealIdpRestoreEvidence(receipt(),source)
})
for(const field of Object.keys(proofs)) test('public gate rejects false '+field,()=>{
  assert.equal(typeof helper.validateRealIdpRestoreEvidence,'function');const r=receipt();r.proofs[field]=false;assert.throws(()=>helper.validateRealIdpRestoreEvidence(r,source))
})
for(const [name,change] of [
 ['fixture provider',r=>{r.provider.kind='signed-protocol-fixture'}],['wrong source',r=>{r.source.commit='d'.repeat(40)}],
 ['private field',r=>{r.secret='PRIVATE-MARKER'}],['private nested field',r=>{r.restore.path='PRIVATE-MARKER'}],
 ['missing proof',r=>{delete r.proofs.oldSessionRejected}],['old revision',r=>{r.revision=2}],['invalid principal',r=>{r.principal='principal:bootstrap'}],
 ['original fallback',r=>{r.restore.originalUnavailable=false}],['changed authority',r=>{r.restore.authorityRetained=false}],
 ['invalid snapshot',r=>{r.restore.snapshotSHA256='not-a-hash'}],['empty inventory',r=>{r.restore.completeFiles=0}],['inherited PATH',r=>{r.runtimePATH='inherited'}],
]) test('public gate rejects '+name,()=>{assert.equal(typeof helper.validateRealIdpRestoreEvidence,'function');const r=receipt();change(r);assert.throws(()=>helper.validateRealIdpRestoreEvidence(r,source))})
async function fixture(t){
 const dir=await mkdtemp(join(tmpdir(),'picketx-real-idp-restore-unit-'));t.after(()=>rm(dir,{recursive:true,force:true}))
 const instance=join(dir,'instance'),backup=join(dir,'stopped backup'),target=join(dir,'restored instance'),retired=join(dir,'original unavailable')
 await mkdir(instance,{mode:0o700});await mkdir(join(instance,'state'),{mode:0o700});await mkdir(join(instance,'identity'),{mode:0o700})
 const configuration={database:join(instance,'state/controller.sqlite'),credentials:join(instance,'agents.json'),management:{credentials:join(instance,'principals.json')},authz:{credentials:join(instance,'callers.json')},browser:{origin:'https://127.0.0.1:12345',oidc_config:join(instance,'identity/provider.json')},agent:{authority:'same-authority'}}
 const provider={issuer:'https://127.0.0.2:12346/realms/test',client_id:'test',client_secret:'PRIVATE-CLIENT-SECRET',bindings:join(instance,'identity/bindings.json')}
 for(const [name,value]of Object.entries({'controller.yaml':JSON.stringify(configuration),'identity/provider.json':JSON.stringify(provider),'identity/bindings.json':'{"bindings":[{"subject":"PRIVATE-SUBJECT","principal_id":"same-principal","generation":"same-generation"}]}','state/controller.sqlite':'PRIVATE-DB','agents.json':'PRIVATE-AGENTS','principals.json':'PRIVATE-PRINCIPALS','callers.json':'PRIVATE-CALLERS','bootstrap-key':'PRIVATE-BOOTSTRAP','agent-token':'PRIVATE-AGENT','ingress-token':'PRIVATE-INGRESS'}))await writeFile(join(instance,name),value,{mode:0o600})
 return {dir,instance,backup,target,retired,configuration,provider}
}
test('complete private restore relocates both references and retires original without reinit',async t=>{
 assert.equal(typeof helper.restoreRealIdpInstance,'function');const f=await fixture(t)
 const r=await helper.restoreRealIdpInstance(f.instance,f.backup,f.target,f.retired)
 assert.equal(r.instance,f.target);assert.equal(r.configuration.browser.oidc_config,join(f.target,'identity/provider.json'))
 assert.equal(r.configuration.database,join(f.target,'state/controller.sqlite'));assert.equal(r.provider.bindings,join(f.target,'identity/bindings.json'))
 assert.equal(r.provider.client_secret,f.provider.client_secret);assert.equal(r.configuration.agent.authority,'same-authority')
 assert.equal(await readFile(r.provider.bindings,'utf8'),await readFile(join(f.backup,'identity/bindings.json'),'utf8'))
 assert.equal(await readFile(join(f.retired,'state/controller.sqlite'),'utf8'),'PRIVATE-DB')
 await assert.rejects(lstat(f.instance),{code:'ENOENT'});assert.equal(r.proof.completeFiles,10);assert(r.proof.originalUnavailable&&r.proof.changedOnlyConfigPaths)
 for(const path of [f.target,f.backup,f.retired])assert.equal((await lstat(path)).mode&0o777,0o700)
 for(const path of ['controller.yaml','identity/provider.json','identity/bindings.json','state/controller.sqlite'])assert.equal((await lstat(join(f.target,path))).mode&0o777,0o600)
})
for(const name of ['backup','target','retired']) test('existing '+name+' is preserved before any restore writes',async t=>{
 assert.equal(typeof helper.restoreRealIdpInstance,'function');const f=await fixture(t);await writeFile(f[name],'OPERATOR-RETAINED',{mode:0o600})
 await assert.rejects(helper.restoreRealIdpInstance(f.instance,f.backup,f.target,f.retired),{message:'real IdP stopped restore failed'})
 assert.equal(await readFile(f[name],'utf8'),'OPERATOR-RETAINED');assert.equal(await readFile(join(f.instance,'state/controller.sqlite'),'utf8'),'PRIVATE-DB')
})
test('symlinked provider is rejected without revealing private paths or credentials',async t=>{
 assert.equal(typeof helper.restoreRealIdpInstance,'function');const f=await fixture(t);await rm(join(f.instance,'identity/provider.json'));await symlink('../controller.yaml',join(f.instance,'identity/provider.json'))
 let error;try{await helper.restoreRealIdpInstance(f.instance,f.backup,f.target,f.retired)}catch(e){error=e}
 assert(error);assert.equal(error.message,'real IdP stopped restore failed');for(const x of [f.dir,'PRIVATE-CLIENT-SECRET','PRIVATE-SUBJECT'])assert(!error.stack.includes(x))
})
