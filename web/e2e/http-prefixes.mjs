// SPDX-License-Identifier: Apache-2.0
// Catches missing explicit prefix UI, raw byte-prefix widening and stale allows.
import assert from 'node:assert/strict'
import { writeFile } from 'node:fs/promises'
import { createHash } from 'node:crypto'
import { join } from 'node:path'
import { freePort, startIngress } from './ingress.mjs'

export async function httpPrefixManagement({ operator, api, root, temp, authzPort, ingressToken, callersPath, evidence }) {
  const mutate = async (path, body) => {
    const base = await api('/publication')
    return api(path, { method: 'POST', headers: { 'If-Match': base.response.headers.get('etag') }, body: JSON.stringify(body) })
  }
  const resource = await mutate('/resources', { name:'prefix-app', reason:'Prefix application', endpoints:[{name:'web',protocol:'TCP',port:443}] })
  assert.equal(resource.response.status,201,JSON.stringify(resource.data))
  await operator.getByRole('button',{name:'Use latest route revision',exact:true}).click()
  const port = await freePort()
  await operator.getByLabel('Route name',{exact:true}).fill('prefix-only-route')
  await operator.getByLabel('Route Resource',{exact:true}).selectOption(resource.data.uid)
  await operator.getByLabel('TCP endpoint',{exact:true}).selectOption('web')
  await operator.getByLabel('Exact request Host',{exact:true}).fill(`127.0.0.1:${port}`)
  await operator.getByRole('checkbox',{name:'GET',exact:true}).check()
  const prefixes = operator.getByLabel('Path-segment prefixes, one per line',{exact:true})
  await prefixes.fill('/repo/')
  await operator.getByLabel('Route creation reason',{exact:true}).fill('Browser prefix-only route')
  await operator.getByRole('button',{name:'Create HTTP route',exact:true}).click()
  assert.equal(await prefixes.getAttribute('aria-invalid'),'true')
  assert.equal(await prefixes.inputValue(),'/repo/')
  await prefixes.fill('/')
  await operator.getByText('Root prefix / includes every canonical path. It still requires matching source Grants, Host and methods.',{exact:true}).waitFor()
  await operator.screenshot({path:join(evidence,'http-prefix-root-desktop.png'),fullPage:true})
  await prefixes.fill('/repo')
  const pending = operator.waitForResponse(r=>r.url().endsWith('/http-routes') && r.request().method()==='POST')
  await operator.getByRole('button',{name:'Create HTTP route',exact:true}).click()
  const response = await pending; assert.equal(response.status(),201); const route = await response.json()
  const article = operator.locator(`[data-route-id="${route.uid}"]`); await article.waitFor()
  const inventory = await api('/inventory'), saved = inventory.data.http_routes.find(r=>r.uid===route.uid)
  assert.deepEqual(saved.path_prefixes,['/repo']); assert.equal(saved.paths?.length ?? 0,0)
  await article.getByText('Inspect 1 segment prefixes',{exact:true}).click()
  assert.equal(await article.locator('code').last().textContent(),'/repo')
  await operator.screenshot({path:join(evidence,'http-prefix-desktop.png'),fullPage:true})
  await writeFile(callersPath,JSON.stringify({callers:[{id:'nginx',sha256:createHash('sha256').update(ingressToken).digest('hex'),routes:[route.uid]}]}),{mode:0o600})
  const ingress = await startIngress({root,temp:join(temp,'prefix-ingress'),authzPort,token:ingressToken,routeID:route.uid,listenPort:port})
  try {
    await ingress.verifyPrefixRoute(false)
    const grant = await mutate('/grants',{name:'prefix-source',resource_name:'prefix-app',resource_uid:resource.data.uid,endpoint_names:['web'],source:'127.0.0.1',duration_seconds:900,reason:'Prefix source access'})
    assert.equal(grant.response.status,201,JSON.stringify(grant.data)); await ingress.verifyPrefixRoute(true)
    const application = await operator.context().newPage(); await application.goto(`${ingress.url}/repo/owner/project`)
    assert.equal(await application.locator('body').textContent(),'protected real HTTP application'); await application.close()
    const revoke = await mutate(`/grants/${grant.data.uid}/revoke`,{reason:'Prefix source revoked'})
    assert.equal(revoke.response.status,200); await ingress.verifyPrefixRoute(false)
  } finally { await ingress.close() }
  await operator.setViewportSize({width:375,height:812})
  assert(await operator.evaluate(()=>document.documentElement.scrollWidth<=innerWidth))
  await operator.screenshot({path:join(evidence,'http-prefix-mobile.png'),fullPage:true})
  await prefixes.fill('/')
  await operator.route('**/console/api/v1/inventory',r=>r.fulfill({status:503,contentType:'application/json',body:'{"code":"management_unavailable"}'}))
  await operator.getByRole('button',{name:'Refresh routes',exact:true}).click()
  await operator.locator('.http-routes-section .stale').waitFor()
  assert.equal(await prefixes.inputValue(),'/','Failed observation must retain the prefix draft')
  await operator.unroute('**/console/api/v1/inventory')
  await operator.getByRole('button',{name:'Language',exact:true}).click()
  await operator.getByLabel('路径段前缀，每行一个',{exact:true}).waitFor()
  await operator.getByText('根前缀 / 包括所有规范路径，仍须满足来源 Grant、Host 和方法条件。',{exact:true}).waitFor()
  assert(await operator.evaluate(()=>document.documentElement.scrollWidth<=innerWidth))
  await operator.screenshot({path:join(evidence,'http-prefix-zh-mobile.png'),fullPage:true})
  await operator.setViewportSize({width:1440,height:1000})
  await operator.screenshot({path:join(evidence,'http-prefix-zh-desktop.png'),fullPage:true})
  await operator.getByText('English',{exact:true}).click()
  operator.once('dialog',d=>d.accept())
  await operator.getByRole('button',{name:'Clear route draft',exact:true}).click()
  console.log('PASS real prefix-only React/API/SQLite → Nginx → application: segment boundary, canonical path, source/Host/method isolation, Grant revoke on reused connection')
}
