// SPDX-License-Identifier: Apache-2.0
import { useEffect, useEffectEvent, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { APIError, request } from '@/lib/console'

type Resource = { name: string; uid: string; endpoints: { name: string; protocol: string; port: number }[] }
type Scope = { resource_name: string; resource_uid: string; selection: 'named' | 'selector' | 'all'; endpoint_names?: string[]; selector?: string }
type Grant = { name: string; uid: string; source: string; scopes: Scope[]; mode: string; not_before?: string; expires_at?: string; revoked: boolean }
type Page = { authority: string; revision: number; observed_at: string; resources: Resource[]; grants: Grant[]; next_after: string; has_more: boolean }
type Form = { name: string; source: string; resource: string; endpoints: string[]; duration: string; reason: string }
const empty: Form = { name: '', source: '', resource: '', endpoints: [], duration: '3600', reason: '' }
type Props = { csrf: string; canWrite: boolean; onUnauthorized: () => void; onDirty: (dirty: boolean) => void; onBusy: (busy: boolean) => void }

export function Grants({ csrf, canWrite, onUnauthorized, onDirty, onBusy }: Props) {
  const { t, i18n } = useTranslation()
  const [page, setPage] = useState<Page | null>(null)
  const [pageTag, setPageTag] = useState('')
  const [base, setBase] = useState('')
  const [catalog, setCatalog] = useState<Resource[]>([])
  const [form, setForm] = useState<Form>(empty)
  const [revoke, setRevoke] = useState<Grant | null>(null)
  const [revokeReason, setRevokeReason] = useState('')
  const [busy, setBusy] = useState(true)
  const [error, setError] = useState('')
  const [fields, setFields] = useState<Record<string, string>>({})
  const [notice, setNotice] = useState('')
  const [conflict, setConflict] = useState(false)
  const [readFailed, setReadFailed] = useState(false)
  const [now, setNow] = useState(() => Date.now())
  const errors = useRef<HTMLDivElement>(null)
  const [controllers] = useState(() => new Set<AbortController>())
  const issueDirty = form.name !== '' || form.source !== '' || form.resource !== '' || form.endpoints.length > 0 || form.reason !== '' || form.duration !== empty.duration
  useEffect(() => { onDirty(issueDirty || revoke !== null) }, [issueDirty, revoke, onDirty])
  useEffect(() => { onBusy(busy) }, [busy, onBusy])
  useEffect(() => { if (error) errors.current?.focus() }, [error, fields])

  function fail(e: unknown, reading = false) {
    if (e instanceof DOMException && e.name === 'AbortError') return
    if (e instanceof APIError && e.status === 401) { onUnauthorized(); return }
    if (e instanceof APIError && e.status === 412) { setConflict(true); setError(t('conflict')); return }
    setError(e instanceof APIError ? t(`errors.${e.message}`, { defaultValue: t('requestFailed') }) : t('requestFailed'))
    if (reading) setReadFailed(true)
  }
  function failWrite(e: unknown) {
    // A transport/5xx failure is not proof of rollback. Require inspection and
    // explicit rebase, never automatically resubmit a possibly committed Grant.
    if (!(e instanceof APIError) || e.status >= 500) setConflict(true)
    fail(e)
  }
  async function read(signal: AbortSignal, rebase = false, next = false) {
    const result = await request<Page>(`/grants?limit=50${next && page ? `&after=${encodeURIComponent(page.next_after)}` : ''}`, { signal, headers: next ? { 'If-Match': pageTag } : {} })
    if (signal.aborted) return
    if (!/^"\d+"$/.test(result.etag)) throw new Error('Missing strong revision')
    setPage(result.data); setPageTag(result.etag); setReadFailed(false)
    // Only explicit rebase, initial load or a completed operation starts a new draft.
    if (rebase || base === '') { setBase(result.etag); setCatalog(result.data.resources); setConflict(false) }
  }
  const initial = useEffectEvent(async (controller: AbortController) => {
    try { await read(controller.signal, true) } catch (e) { if (!controller.signal.aborted) fail(e, true) }
    finally { controllers.delete(controller); if (!controller.signal.aborted) setBusy(false) }
  })
  useEffect(() => {
    const controller = new AbortController(); controllers.add(controller)
    queueMicrotask(() => { if (!controller.signal.aborted) void initial(controller) })
    const timer = window.setInterval(() => setNow(Date.now()), 15000)
    return () => { for (const c of controllers) c.abort(); controllers.clear(); window.clearInterval(timer) }
  }, [controllers])

  async function refresh(rebase = false, next = false) {
    if (busy) return
    if (rebase && (issueDirty || revoke) && !window.confirm(t('grants.rebaseConfirm'))) return
    const controller = new AbortController(); controllers.add(controller); setBusy(true); setError(''); setFields({}); setNotice('')
    try { await read(controller.signal, rebase, next) } catch (e) { if (!controller.signal.aborted) fail(e, true) }
    finally { controllers.delete(controller); if (!controller.signal.aborted) setBusy(false) }
  }
  function edit<K extends keyof Form>(key: K, value: Form[K]) { setForm(old => ({ ...old, [key]: value })) }
  async function submit(e: React.FormEvent) {
    e.preventDefault(); if (busy || !canWrite || conflict || !base || revoke) return
    const invalid: Record<string, string> = {}
    for (const key of ['name', 'source', 'reason'] as const) if (!form[key].trim()) invalid[`grant-${key}`] = t('grants.required')
    const resource = catalog.find(r => r.uid === form.resource)
    if (!resource) invalid['grant-resource'] = t('grants.selectResource')
    if (!form.endpoints.length || form.endpoints.some(name => !resource?.endpoints.some(ep => ep.name === name))) invalid['grant-endpoints'] = t('grants.selectEndpoints')
    if (Object.keys(invalid).length) { setFields(invalid); setError(t('grants.fixFields')); return }
    const controller = new AbortController(); controllers.add(controller); setBusy(true); setError(''); setFields({}); setNotice('')
    try {
      await request('/grants', { signal: controller.signal, method: 'POST', headers: { 'Content-Type': 'application/json', 'If-Match': base, 'X-CSRF-Token': csrf }, body: JSON.stringify({ name: form.name, source: form.source, resource_name: resource!.name, resource_uid: resource!.uid, endpoint_names: form.endpoints, duration_seconds: Number(form.duration), reason: form.reason }) })
      if (controller.signal.aborted) return
      setForm(empty); setNotice(t('grants.issued'))
      // A failed follow-up read must not permit a duplicate POST with the old base.
      setConflict(true)
      await read(controller.signal, true)
    } catch (e) { if (!controller.signal.aborted) failWrite(e) }
    finally { controllers.delete(controller); if (!controller.signal.aborted) setBusy(false) }
  }
  async function confirmRevoke(e: React.FormEvent) {
    e.preventDefault(); if (busy || !revoke || !revokeReason.trim() || !canWrite || conflict || !base) return
    const controller = new AbortController(); controllers.add(controller); setBusy(true); setError(''); setNotice('')
    try {
      await request(`/grants/${encodeURIComponent(revoke.uid)}/revoke`, { signal: controller.signal, method: 'POST', headers: { 'Content-Type': 'application/json', 'If-Match': base, 'X-CSRF-Token': csrf }, body: JSON.stringify({ reason: revokeReason }) })
      if (controller.signal.aborted) return
      setRevoke(null); setRevokeReason(''); setNotice(t('grants.revoked')); setConflict(true)
      await read(controller.signal, true)
    } catch (e) { if (!controller.signal.aborted) failWrite(e) }
    finally { controllers.delete(controller); if (!controller.signal.aborted) setBusy(false) }
  }
  function validity(g: Grant) {
    const at = page ? Date.parse(page.observed_at) : 0
    return g.revoked ? 'revokedState' : g.not_before && Date.parse(g.not_before) > at ? 'future' : g.expires_at && Date.parse(g.expires_at) <= at ? 'expiredState' : 'valid'
  }
  const date = (value: string) => new Date(value).toLocaleString(i18n.language)
  const fieldError = (id: string) => fields[id] && <p id={`${id}-error`} className="field-error">{fields[id]}</p>
  const fieldProps = (id: string) => ({ 'aria-invalid': !!fields[id], 'aria-describedby': fields[id] ? `${id}-error` : undefined })
  return <section className="grants-section" aria-busy={busy}>
    <div className="section-heading"><div><h2>{t('grants.access')}</h2><p>{t('grants.boundary')}</p></div><div className="actions"><Button variant="outline" disabled={busy} onClick={() => void refresh()}>{t('grants.refresh')}</Button><Button variant="outline" disabled={busy} onClick={() => void refresh(true)}>{t('grants.rebase')}</Button></div></div>
    <p className="hint">{t('grants.base', { base: base.slice(1, -1) || '—', observed: page?.revision ?? '—' })} · {t('lastRead')}: {page ? date(page.observed_at) : t('notRead')}</p>
    {error && <div className="feedback error" role="alert" tabIndex={-1} ref={errors}>{error}{Object.keys(fields).length > 0 && <ul>{Object.entries(fields).map(([id, text]) => <li key={id}><a href={`#${id}`} onClick={() => document.getElementById(id)?.focus()}>{t(`grants.${id.slice(6)}`)}: {text}</a></li>)}</ul>}</div>}
    {notice && <p className="feedback success" role="status">{notice}</p>}
    {readFailed || conflict || (page && now - Date.parse(page.observed_at) > 60000) ? <p className="stale" role="status">{conflict ? t('grants.conflictHelp') : readFailed ? t('staleFailed') : t('staleAge')}</p> : null}
    {busy && <p role="status">{t('loading')}</p>}
    {!canWrite ? <p className="boundary">{t('readOnly')}</p> : <form className="grant-form" onSubmit={e => void submit(e)} noValidate><fieldset className="grant-fields-owner" disabled={busy || !!revoke}>
      <div className="grant-fields">
        <div><label htmlFor="grant-name">{t('grants.name')}</label><input id="grant-name" {...fieldProps('grant-name')} value={form.name} onChange={e => edit('name', e.target.value)} maxLength={128} required disabled={busy} />{fieldError('grant-name')}</div>
        <div><label htmlFor="grant-source">{t('grants.source')}</label><input id="grant-source" {...fieldProps('grant-source')} value={form.source} onChange={e => edit('source', e.target.value)} maxLength={128} required disabled={busy} autoCapitalize="none" spellCheck={false} /><p className="hint">{t('grants.sourceHelp')}</p>{fieldError('grant-source')}</div>
        <div><label htmlFor="grant-resource">{t('grants.resource')}</label><select id="grant-resource" {...fieldProps('grant-resource')} value={form.resource} onChange={e => setForm(old => ({ ...old, resource: e.target.value, endpoints: [] }))} required disabled={busy}><option value="">{t('grants.selectResource')}</option>{catalog.map(r => <option key={r.uid} value={r.uid}>{r.name} · {r.uid}</option>)}</select>{fieldError('grant-resource')}{page && catalog.length === 0 && <p className="hint">{t('grants.noResources')}</p>}</div>
        <div><label htmlFor="grant-duration">{t('grants.duration')}</label><select id="grant-duration" value={form.duration} onChange={e => edit('duration', e.target.value)} disabled={busy}>{['900', '3600', '28800', '86400'].map(seconds => <option key={seconds} value={seconds}>{t(`grants.duration${seconds}`)}</option>)}</select><p className="hint">{t('grants.expiryHelp')}</p></div>
      </div>
      <fieldset id="grant-endpoints" tabIndex={-1} {...fieldProps('grant-endpoints')} disabled={busy}><legend>{t('grants.endpoints')}</legend><div className="endpoint-options">{catalog.find(r => r.uid === form.resource)?.endpoints.map(ep => <label key={ep.name}><input type="checkbox" checked={form.endpoints.includes(ep.name)} onChange={e => edit('endpoints', e.target.checked ? [...form.endpoints, ep.name] : form.endpoints.filter(n => n !== ep.name))} />{ep.name} <span>{ep.protocol}/{ep.port}</span></label>)}</div>{fieldError('grant-endpoints')}</fieldset>
      <div><label htmlFor="grant-reason">{t('grants.reason')}</label><input id="grant-reason" {...fieldProps('grant-reason')} value={form.reason} onChange={e => edit('reason', e.target.value)} maxLength={512} required disabled={busy} />{fieldError('grant-reason')}</div>
      <div className="actions"><Button type="submit" disabled={busy || conflict || !base || !!revoke}>{t('grants.issue')}</Button><Button type="button" variant="outline" disabled={busy || !issueDirty} onClick={() => { if (window.confirm(t('grants.discard'))) { setForm(empty); setFields({}) } }}>{t('grants.clear')}</Button></div>
    </fieldset></form>}
    {revoke && <form className="revoke-panel" onSubmit={e => void confirmRevoke(e)}><h3>{t('grants.confirmTitle')}</h3><p>{revoke.name} · <code>{revoke.source}</code></p><p className="hint">{revoke.uid} · {t('grants.revokeHelp')}</p><label htmlFor="revoke-reason">{t('grants.revokeReason')}</label><input id="revoke-reason" value={revokeReason} onChange={e => setRevokeReason(e.target.value)} maxLength={512} required disabled={busy} /><div className="actions"><Button type="submit" variant="destructive" disabled={busy || conflict || !revokeReason.trim()}>{t('grants.confirm')}</Button><Button type="button" variant="outline" disabled={busy} onClick={() => { setRevoke(null); setRevokeReason('') }}>{t('grants.cancel')}</Button></div></form>}
    <div className="section-heading"><div><h2>{t('grants.list')}</h2><p>{t('grants.observed')}</p></div></div>
    {page?.grants.length === 0 && !busy && <p className="empty">{t('grants.empty')}</p>}
    <div className="grant-list">{page?.grants.map(g => <article key={g.uid} data-grant-id={g.uid}><div className="grant-row-heading"><h3>{g.name}</h3><span className="grant-status">{t(`grants.${validity(g)}`)}</span></div><p><code>{g.source}</code></p><p className="hint">{g.uid}</p><ul>{g.scopes.map((scope, index) => <li key={index}>{scope.resource_name} · <code>{scope.resource_uid}</code> / {scope.selection === 'named' ? scope.endpoint_names?.join(', ') : t(`grants.scope${scope.selection}`)}{scope.selector && <details><summary>{t('grants.selector')}</summary><code>{scope.selector}</code></details>}</li>)}</ul><p>{g.not_before && <>{t('grants.starts')}: <time dateTime={g.not_before}>{date(g.not_before)}</time> · </>}{t('grants.expires')}: {g.expires_at ? <time dateTime={g.expires_at}>{date(g.expires_at)}</time> : t('grants.permanent')}</p>{canWrite && !g.revoked && <Button variant="outline" disabled={busy || conflict || issueDirty || !!revoke} onClick={() => { setRevoke(g); setRevokeReason(''); setError(''); queueMicrotask(() => document.getElementById('revoke-reason')?.focus()) }}>{t('grants.revoke')}</Button>}</article>)}</div>
    {page?.has_more && <Button variant="outline" disabled={busy || conflict} onClick={() => void refresh(false, true)}>{t('grants.next')}</Button>}
    {issueDirty && <p className="hint">{t('grants.finishDraft')}</p>}
  </section>
}
