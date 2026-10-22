// SPDX-License-Identifier: Apache-2.0
// Controlled valid display data; never authorization/forwarding acceptance.
import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { transformWithOxc } from 'vite'
import React from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider } from 'react-i18next'
import i18next from 'i18next'

async function compile(url) {
  const source = await readFile(url, 'utf8')
  let js = (await transformWithOxc(source, url.pathname)).code
  const button = '@/components/ui/button'
  if (js.includes(button)) js = js.replaceAll(button, await compile(new URL('../src/components/ui/button.tsx', import.meta.url)))
  js = js.replace(/from (["'])([^"']+)\1/g, (all, quote, name) => name.startsWith('data:') || name.startsWith('file:') ? all : `from ${quote}${name.startsWith('.') ? new URL(name + (name.endsWith('.ts') ? '' : '.ts'), url).href : import.meta.resolve(name)}${quote}`)
  return 'data:text/javascript;base64,' + Buffer.from(js).toString('base64')
}
const { DecisionLedger, ContributionList } = await import(await compile(new URL('../src/features/http-audit/DecisionLedger.tsx', import.meta.url)))
const language = i18next.createInstance()
await language.init({ lng: 'en', resources: { en: { translation: JSON.parse(await readFile(new URL('../src/locales/en.json', import.meta.url), 'utf8')) } } })
const render = element => renderToStaticMarkup(React.createElement(I18nextProvider, { i18n: language }, element))
const ids = Array.from({ length: 4096 }, (_, i) => `grant-${i}-${'x'.repeat(235)}`)
test('200 collapsed near-limit records render zero contribution list items', () => {
  const rows = Array.from({ length: 200 }, (_, i) => ({ id: i + 1, allowed: true, at: '2026-10-20T11:00:00Z', source: '2001:db8::1', route: 'route', caller: 'nginx', reason: 'allowed', revision: 1, grant_ids: ids, deny_ids: ids }))
  const html = render(React.createElement(DecisionLedger, { rows }))
  assert.equal((html.match(/<article /g) ?? []).length, 200)
  assert.equal((html.match(/<li[ >]/g) ?? []).length, 0)
  assert(html.length < 300000, 'collapsed output independent of contribution count')
})
test('an opened contribution list renders only50 IDs with a visible total and paging control', () => {
  const html = render(React.createElement(ContributionList, { ids, label: 'Grants' }))
  assert.equal((html.match(/<li[ >]/g) ?? []).length, 50)
  assert(html.includes('4096')); assert(html.includes('Next IDs'))
  assert(!html.includes(ids[50]))
})
