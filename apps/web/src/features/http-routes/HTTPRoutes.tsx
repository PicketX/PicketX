// SPDX-License-Identifier: Apache-2.0
import { useEffect, useEffectEvent, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { APIError, request } from '@/lib/console'

type Resource = { name: string; uid: string; endpoints: { name: string; protocol: string; port: number }[] }
type Route = { name: string; uid: string; resource_name: string; resource_uid: string; endpoint_name: string; host: string; methods: string[]; paths: string[] }
type Inventory = { revision: number; observed_at: string; resources: Resource[]; http_routes: Route[] }
type Props = { csrf: string; canWrite: boolean; onUnauthorized: () => void; onDirty: (value: boolean) => void; onBusy: (value: boolean) => void }
const methods = ['GET', 'HEAD', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS']
const bytes = (value: string) => new TextEncoder().encode(value).length

export function HTTPRoutes({ csrf, canWrite, onUnauthorized, onDirty, onBusy }: Props) {
  const { t, i18n } = useTranslation()
  const [inventory, setInventory] = useState<Inventory | null>(null)
  const [catalog, setCatalog] = useState<Resource[]>([])
  const [routeCatalog, setRouteCatalog] = useState<Route[]>([])
  const [removalUID, setRemovalUID] = useState('')
  const [acknowledged, setAcknowledged] = useState(false)
  const [base, setBase] = useState('')
  const [name, setName] = useState('')
  const [resourceUID, setResourceUID] = useState('')
  const [endpoint, setEndpoint] = useState('')
  const [host, setHost] = useState('')
  const [selectedMethods, setSelectedMethods] = useState<string[]>([])
  const [paths, setPaths] = useState('')
  const [reason, setReason] = useState('')
  const [busy, setBusy] = useState(true)
  const [error, setError] = useState('')
  const [fields, setFields] = useState<Record<string, string>>({})
  const [notice, setNotice] = useState('')
  const [conflict, setConflict] = useState(false)
  const [readFailed, setReadFailed] = useState(false)
  const [now, setNow] = useState(() => Date.now())
  const [page, setPage] = useState(0)
  const [controllers] = useState(() => new Set<AbortController>())
  const errors = useRef<HTMLDivElement>(null)
  const removalHeading = useRef<HTMLHeadingElement>(null)
  const dirty = !!(removalUID || name || resourceUID || endpoint || host || selectedMethods.length || paths || reason)
  const removalTarget = routeCatalog.find(r => r.uid === removalUID)
  const resource = catalog.find(r => r.uid === resourceUID)
  useEffect(() => { onDirty(dirty) }, [dirty, onDirty])
  useEffect(() => { onBusy(busy) }, [busy, onBusy])
  useEffect(() => { if (error) errors.current?.focus() }, [error, fields])
  useEffect(() => { if (removalUID) removalHeading.current?.focus() }, [removalUID])

  function fail(e: unknown, reading = false) {
    if (e instanceof DOMException && e.name === 'AbortError') return
    if (e instanceof APIError && e.status === 401) { onUnauthorized(); return }
    if (e instanceof APIError && e.status === 412) { setConflict(true); setError(t('conflict')); return }
    setError(e instanceof APIError && ['invalid_fields', 'invalid_publication'].includes(e.message)
      ? t('routes.invalid') : e instanceof APIError ? t(`errors.${e.message}`, { defaultValue: t('requestFailed') }) : t('requestFailed'))
    if (reading) setReadFailed(true)
  }
  async function read(signal: AbortSignal, rebase = false) {
    const result = await request<Inventory>('/inventory', { signal })
    if (signal.aborted) return
    if (!/^"\d+"$/.test(result.etag)) throw new Error('Missing strong revision')
    setInventory(result.data); setReadFailed(false); setPage(0)
    // Observation and mutation base remain independent while a draft is retained.
    if (rebase || !base) { setBase(result.etag); setCatalog(result.data.resources); setRouteCatalog(result.data.http_routes); setAcknowledged(false); setConflict(false) }
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
  async function refresh(rebase = false) {
    if (busy || (rebase && dirty && !window.confirm(t('routes.rebaseConfirm')))) return
    const controller = new AbortController(); controllers.add(controller)
    setBusy(true); setError(''); setFields({}); setNotice('')
    try { await read(controller.signal, rebase) } catch (e) { if (!controller.signal.aborted) fail(e, true) }
    finally { controllers.delete(controller); if (!controller.signal.aborted) setBusy(false) }
  }
  function clear() { setRemovalUID(''); setAcknowledged(false); setName(''); setResourceUID(''); setEndpoint(''); setHost(''); setSelectedMethods([]); setPaths(''); setReason(''); setFields({}) }
  function beginRemoval(uid: string) {
    if (busy || dirty || conflict || !canWrite || !base || !routeCatalog.some(r => r.uid === uid)) return
    setRemovalUID(uid); setAcknowledged(false); setError(''); setFields({}); setNotice('')
  }
  async function remove(e: React.FormEvent) {
    e.preventDefault()
    if (busy || !canWrite || !base || conflict || !removalTarget || !acknowledged) return
    if (!reason.trim() || bytes(reason) > 512) { setFields({ removalReason: t('routes.reasonLimit') }); setError(t('grants.fixFields')); return }
    const controller = new AbortController(); controllers.add(controller); setBusy(true); setError(''); setFields({}); setNotice('')
    try {
      const result = await request<{ uid: string; revision: number }>(`/http-routes/${encodeURIComponent(removalUID)}/remove`, { method: 'POST', signal: controller.signal, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf, 'If-Match': base }, body: JSON.stringify({ reason, acknowledge_removal: true }) })
      if (controller.signal.aborted) return
      // An ambiguous 2xx is not proof of the requested operation. Retain intent,
      // block retries, and require an explicit same-UID rebase before another write.
      if (result.status !== 200 || !result.data || result.data.uid !== removalUID || !Number.isSafeInteger(result.data.revision) || result.data.revision <= Number(base.slice(1, -1)) || result.etag !== `"${result.data.revision}"`) throw new Error('Invalid removal receipt')
      clear(); setNotice(t('routes.removed', { uid: removalUID })); setConflict(true)
      try { await read(controller.signal, true) } catch (e) { if (!controller.signal.aborted) fail(e, true) }
    } catch (e) {
      if (!controller.signal.aborted) { if (!(e instanceof APIError) || e.status >= 500 || e.status === 404) setConflict(true); fail(e) }
    } finally { controllers.delete(controller); if (!controller.signal.aborted) setBusy(false) }
  }
  async function submit(e: React.FormEvent) {
    e.preventDefault(); if (busy || !canWrite || !base || conflict) return
    const invalid: Record<string, string> = {}
    const pathList = paths.split('\n')
    if (!name.trim() || bytes(name) > 128) invalid.name = t('routes.nameLimit')
    if (!resource) invalid.resource = t('grants.selectResource')
    if (!resource?.endpoints.some(e => e.name === endpoint && e.protocol === 'TCP')) invalid.endpoint = t('routes.selectEndpoint')
    if (!host || host.length > 253 || /[\s/?#@]/.test(host)) invalid.host = t('routes.hostHelp')
    if (!selectedMethods.length) invalid.methods = t('routes.selectMethods')
    if (!paths || pathList.length > 128 || new Set(pathList).size !== pathList.length || pathList.some(p => !p.startsWith('/') || p.length > 8192 || /[^\x21-\x7e]|[%\\?#]/.test(p) || p.includes('//') || p.split('/').some(s => s === '.' || s === '..'))) invalid.paths = t('routes.pathHelp')
    if (!reason.trim() || bytes(reason) > 512) invalid.reason = t('routes.reasonLimit')
    if (Object.keys(invalid).length) { setFields(invalid); setError(t('grants.fixFields')); return }
    const body = JSON.stringify({ name, resource_name: resource!.name, resource_uid: resource!.uid, endpoint_name: endpoint, host, methods: selectedMethods, paths: pathList, reason })
    if (bytes(body) > 16384) { setFields({ paths: t('routes.bodyLimit') }); setError(t('grants.fixFields')); return }
    const controller = new AbortController(); controllers.add(controller); setBusy(true); setError(''); setFields({}); setNotice('')
    try {
      const result = await request<{ uid: string }>('/http-routes', { method: 'POST', signal: controller.signal, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf, 'If-Match': base }, body })
      if (controller.signal.aborted) return
      clear(); setNotice(t('routes.created', { uid: result.data.uid })); setConflict(true)
      // Never repeat a successful mutation merely because its follow-up read fails.
      try { await read(controller.signal, true) } catch (e) { if (!controller.signal.aborted) fail(e, true) }
    } catch (e) {
      if (!controller.signal.aborted) { if (!(e instanceof APIError) || e.status >= 500) setConflict(true); fail(e) }
    } finally { controllers.delete(controller); if (!controller.signal.aborted) setBusy(false) }
  }
  const fieldProps = (key: string) => ({ id: `route-${key}`, 'aria-invalid': !!fields[key], 'aria-describedby': fields[key] ? `route-${key}-error` : undefined })
  const fieldError = (key: string) => fields[key] && <p id={`route-${key}-error`} className="field-error">{fields[key]}</p>
  const pages = Math.ceil((inventory?.http_routes.length ?? 0) / 50)
  return <section className="http-routes-section" aria-busy={busy}>
    <div className="section-heading"><div><h2>{t('routes.management')}</h2><p>{t('routes.boundary')}</p></div><div className="actions"><Button variant="outline" disabled={busy} onClick={() => void refresh()}>{t('routes.refresh')}</Button><Button variant="outline" disabled={busy} onClick={() => void refresh(true)}>{t('routes.rebase')}</Button></div></div>
    <p className="hint">{t('routes.base', { base: base.slice(1, -1) || '—', observed: inventory?.revision ?? '—' })} · {t('lastRead')}: {inventory ? new Date(inventory.observed_at).toLocaleString(i18n.language) : t('notRead')}</p>
    {error && <div className="feedback error" role="alert" tabIndex={-1} ref={errors}><p>{error}</p><ul>{Object.entries(fields).map(([key, text]) => <li key={key}><a href={`#route-${key}`} onClick={() => document.getElementById(`route-${key}`)?.focus()}>{t(`routes.${key}`)}: {text}</a></li>)}</ul></div>}
    {notice && <p className="feedback success route-result" role="status">{notice}</p>}
    {conflict || readFailed || (inventory && now - Date.parse(inventory.observed_at) > 60000) ? <p className="stale" role="status">{conflict ? t('routes.conflictHelp') : readFailed ? t('staleFailed') : t('staleAge')}</p> : null}
    {busy && <p role="status">{t('loading')}</p>}
    {!canWrite ? <p className="boundary">{t('readOnly')}</p> : removalUID ? <form className="grant-form resource-form" onSubmit={e => void remove(e)} noValidate><fieldset className="grant-fields-owner" disabled={busy}>
      <h3 tabIndex={-1} ref={removalHeading}>{t('routes.confirmRemoval')}</h3><p className="boundary">{t('routes.removeWarning')}</p>
      <p><strong>{t('routes.uid')}</strong> <code>{removalUID}</code></p>
      {!removalTarget ? <p className="feedback error" role="status">{t('routes.missing')}</p> : <div><h4>{removalTarget.name}</h4><p>{removalTarget.resource_name} · {removalTarget.resource_uid} · {removalTarget.endpoint_name}</p><p><strong>{removalTarget.host}</strong> · {removalTarget.methods.join(', ')}</p><details><summary>{t('routes.inspectPaths', { n: removalTarget.paths.length })}</summary><ul>{removalTarget.paths.map(p => <li key={p}><code>{p}</code></li>)}</ul></details></div>}
      <div><label htmlFor="route-removalReason">{t('routes.removalReason')}</label><input {...fieldProps('removalReason')} value={reason} onChange={e => setReason(e.target.value)} maxLength={512} required />{fieldError('removalReason')}</div>
      <label className="endpoint-options"><input type="checkbox" checked={acknowledged} disabled={!removalTarget} onChange={e => setAcknowledged(e.target.checked)} />{t('routes.acknowledge')}</label>
      <div className="actions"><Button type="submit" disabled={busy || conflict || !base || !removalTarget || !acknowledged || !reason.trim()}>{t('routes.confirmRemoval')}</Button><Button type="button" variant="outline" onClick={() => { clear(); setError('') }}>{t('routes.cancelRemoval')}</Button></div>
    </fieldset></form> : <form className="grant-form resource-form" onSubmit={e => void submit(e)} noValidate><fieldset className="grant-fields-owner" disabled={busy}>
      <h3>{t('routes.create')}</h3><p className="hint">{t('routes.required')}</p>
      <div><label htmlFor="route-name">{t('routes.name')}</label><input {...fieldProps('name')} value={name} onChange={e => setName(e.target.value)} maxLength={128} required />{fieldError('name')}</div>
      <div className="grant-fields">
        <div><label htmlFor="route-resource">{t('routes.resource')}</label><select {...fieldProps('resource')} value={resourceUID} onChange={e => { setResourceUID(e.target.value); setEndpoint('') }} required><option value="">{t('grants.selectResource')}</option>{catalog.filter(r => r.endpoints.some(e => e.protocol === 'TCP')).map(r => <option key={r.uid} value={r.uid}>{r.name} · {r.uid}</option>)}</select>{fieldError('resource')}</div>
        <div><label htmlFor="route-endpoint">{t('routes.endpoint')}</label><select {...fieldProps('endpoint')} value={endpoint} onChange={e => setEndpoint(e.target.value)} required><option value="">{t('routes.selectEndpoint')}</option>{resource?.endpoints.filter(e => e.protocol === 'TCP').map(e => <option key={e.name} value={e.name}>{e.name} · TCP/{e.port}</option>)}</select>{fieldError('endpoint')}</div>
      </div>
      <div><label htmlFor="route-host">{t('routes.host')}</label><input {...fieldProps('host')} value={host} onChange={e => setHost(e.target.value)} maxLength={253} spellCheck={false} required /><p className="hint">{t('routes.hostHelp')}</p>{fieldError('host')}</div>
      <fieldset {...fieldProps('methods')} tabIndex={-1}><legend>{t('routes.methods')}</legend><div className="endpoint-options">{methods.map(method => <label key={method}><input type="checkbox" checked={selectedMethods.includes(method)} onChange={e => setSelectedMethods(old => e.target.checked ? [...old, method] : old.filter(m => m !== method))} />{method}</label>)}</div>{fieldError('methods')}</fieldset>
      <div><label htmlFor="route-paths">{t('routes.paths')}</label><textarea {...fieldProps('paths')} value={paths} onChange={e => setPaths(e.target.value)} maxLength={16384} rows={4} spellCheck={false} required /><p className="hint">{t('routes.pathHelp')}</p>{fieldError('paths')}</div>
      <div><label htmlFor="route-reason">{t('routes.reason')}</label><input {...fieldProps('reason')} value={reason} onChange={e => setReason(e.target.value)} maxLength={512} required />{fieldError('reason')}</div>
      <div className="actions"><Button type="submit" disabled={busy || conflict || !base}>{t('routes.create')}</Button><Button type="button" variant="outline" disabled={!dirty} onClick={() => { if (window.confirm(t('routes.discard'))) clear() }}>{t('routes.clear')}</Button></div>
    </fieldset></form>}
    <div className="inventory-list"><h2>{t('routes.list')}</h2><p className="hint">{t('routes.setup')}</p>
      {inventory?.http_routes.length === 0 && !busy && <p className="empty">{t('routes.empty')}</p>}
      {inventory?.http_routes.slice(page * 50, (page + 1) * 50).map(r => <article key={r.uid} data-route-id={r.uid}><h3>{r.name}</h3><p><strong>{t('routes.uid')}</strong> <code>{r.uid}</code></p><p>{r.resource_name} · {r.resource_uid} · {r.endpoint_name}</p><p><strong>{r.host}</strong> · {r.methods.join(', ')}</p><details><summary>{t('routes.inspectPaths', { n: r.paths.length })}</summary><ul>{r.paths.map(p => <li key={p}><code>{p}</code></li>)}</ul></details>{canWrite && <Button variant="outline" disabled={busy || dirty || conflict || !base || !routeCatalog.some(target => target.uid === r.uid)} onClick={() => beginRemoval(r.uid)}>{t('routes.remove')}</Button>}</article>)}
    </div>
    {pages > 1 && <div className="actions"><Button variant="outline" disabled={page === 0 || busy} onClick={() => setPage(n => n - 1)}>{t('resources.previous')}</Button><span>{t('resources.page', { n: page + 1, pages })}</span><Button variant="outline" disabled={page >= pages - 1 || busy} onClick={() => setPage(n => n + 1)}>{t('resources.next')}</Button></div>}
  </section>
}
