// SPDX-License-Identifier: Apache-2.0
import { useEffect, useEffectEvent, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { APIError, request } from '@/lib/console'

type Endpoint = { name: string; protocol: string; port: number; addresses?: string[] }
type Resource = { name: string; uid: string; endpoints: Endpoint[] }
type Catalog = { revision: number; resources: Resource[]; observed_at: string }
type AccessRequest = {
  id: number; uid: string; requester: string; source: string; resource: Resource; endpoint_names: string[]
  expires_at: string; reason: string; created_at: string; state: 'pending' | 'approved' | 'rejected'; version: number
  approver?: string; decision_reason?: string; approved_endpoints?: string[]; approved_expires_at?: string; grant_uid?: string; revision?: number; grant_state?: string
}
type Page = { requests: AccessRequest[]; next_after: number; has_more: boolean; observed_at: string }
type Form = { source: string; resource: string; endpoints: string[]; duration: string; reason: string }
const empty: Form = { source: '', resource: '', endpoints: [], duration: '3600', reason: '' }
type Props = { active: boolean; csrf: string; identity: string; canCreate: boolean; canList: boolean; canReview: boolean; canApprove: boolean; onUnauthorized: () => void; onDirty: (v: boolean) => void; onBusy: (v: boolean) => void }

export function Requests({ active, csrf, identity, canCreate, canList, canReview, canApprove, onUnauthorized, onDirty, onBusy }: Props) {
  const { t, i18n } = useTranslation()
  const [page, setPage] = useState<Page | null>(null)
  const [catalog, setCatalog] = useState<Catalog | null>(null)
  const [base, setBase] = useState<Catalog | null>(null)
  const [form, setForm] = useState<Form>(empty)
  const [review, setReview] = useState<AccessRequest | null>(null)
  const [reviewRevision, setReviewRevision] = useState(0)
  const [approvedNames, setApprovedNames] = useState<string[]>([])
  const [approvedExpiry, setApprovedExpiry] = useState('')
  const [decisionReason, setDecisionReason] = useState('')
  const [acknowledged, setAcknowledged] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [fields, setFields] = useState<Record<string, string>>({})
  const [notice, setNotice] = useState('')
  const [locked, setLocked] = useState(false)
  const [readFailed, setReadFailed] = useState(false)
  const [now, setNow] = useState(() => Date.now())
  const errors = useRef<HTMLDivElement>(null)
  const [controllers] = useState(() => new Set<AbortController>())
  const [writers] = useState(() => new Set<AbortController>())
  function finish(c: AbortController) { controllers.delete(c); writers.delete(c); setBusy(controllers.size > 0) }
  const dirty = form.source !== '' || form.resource !== '' || form.reason !== '' || form.duration !== empty.duration || review !== null
  useEffect(() => { onDirty(dirty) }, [dirty, onDirty])
  useEffect(() => { onBusy(busy) }, [busy, onBusy])
  useEffect(() => { if (error) errors.current?.focus() }, [error, fields])
  const date = (value: string) => new Date(value).toLocaleString(i18n.language)

  function fail(e: unknown, writing = false) {
    if (e instanceof DOMException && e.name === 'AbortError') return
    if (e instanceof APIError && e.status === 401) { onUnauthorized(); return }
    if (writing && (!(e instanceof APIError) || e.status >= 500 || e.status === 412 || e.status === 409)) setLocked(true)
    setError(e instanceof APIError ? t(`requests.errors.${e.message}`, { defaultValue: t(`errors.${e.message}`, { defaultValue: t('requestFailed') }) }) : t('requestFailed'))
    if (!writing) setReadFailed(true)
  }
  async function read(signal: AbortSignal, rebase = false, next = false) {
    const [rows, latest] = await Promise.all([
      canList ? request<Page>(`/access-requests?limit=50${next && page ? `&after=${page.next_after}` : ''}`, { signal }) : Promise.resolve({ data: { requests: [], next_after: 0, has_more: false, observed_at: new Date().toISOString() } }),
      canCreate || canApprove ? request<Catalog>('/access-request-catalog', { signal }) : Promise.resolve(null),
    ])
    if (signal.aborted) return
    if (latest && (!/^"\d+"$/.test(latest.etag) || latest.etag !== `"${latest.data.revision}"`)) throw new Error('Missing request catalog revision')
    setPage(rows.data); setReadFailed(false)
    if (latest) { setCatalog(latest.data); if (rebase || !base) setBase(latest.data) }
    if (rebase) setLocked(false)
  }
  const start = useEffectEvent(async (controller: AbortController) => {
    setBusy(true)
    try { await read(controller.signal) } catch (e) { if (!controller.signal.aborted) fail(e) }
    finally { finish(controller) }
  })
  useEffect(() => {
    if (!active) return
    const c = new AbortController(); controllers.add(c)
    queueMicrotask(() => { if (!c.signal.aborted) void start(c) })
    const timer = window.setInterval(() => setNow(Date.now()), 15000)
    return () => { for (const read of controllers) { if (!writers.has(read)) { read.abort(); controllers.delete(read) } } setBusy(controllers.size > 0); window.clearInterval(timer) }
  }, [active, controllers, writers])
  useEffect(() => () => { for (const c of controllers) c.abort(); controllers.clear(); writers.clear() }, [controllers, writers])

  async function refresh(rebase = false, next = false) {
    if (busy || (rebase && dirty && !window.confirm(t('requests.discard')))) return
    const c = new AbortController(); controllers.add(c); setBusy(true); setError(''); setFields({}); setNotice('')
    try { await read(c.signal, rebase, next); if (rebase) { setForm(empty); setReview(null); setDecisionReason(''); setAcknowledged(false) } }
    catch (e) { if (!c.signal.aborted) fail(e) }
    finally { finish(c) }
  }
  async function submit(e: React.FormEvent) {
    e.preventDefault(); if (busy || locked || !canCreate || !base) return
    const invalid: Record<string, string> = {}
    if (!form.source.trim()) invalid['request-source'] = t('requests.required')
    if (!form.reason.trim()) invalid['request-reason'] = t('requests.required')
    const resource = base.resources.find(r => r.uid === form.resource)
    if (!resource) invalid['request-resource'] = t('requests.pickResource')
    if (!form.endpoints.length) invalid['request-endpoints'] = t('requests.pickEndpoints')
    if (!/^\d+$/.test(form.duration) || Number(form.duration) < 60 || Number(form.duration) > 86400) invalid['request-duration'] = t('requests.durationHelp')
    if (Object.keys(invalid).length) { setFields(invalid); setError(t('requests.fixFields')); return }
    const c = new AbortController(); controllers.add(c); writers.add(c); setBusy(true); setError(''); setNotice(''); setFields({})
    try {
      await request('/access-requests', { method: 'POST', signal: c.signal, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf, 'If-Match': `"${base.revision}"` }, body: JSON.stringify({ source: form.source, resource_name: resource!.name, resource_uid: resource!.uid, endpoint_names: form.endpoints, duration_seconds: Number(form.duration), reason: form.reason }) }, 201)
      if (c.signal.aborted) return
      setForm(empty); setLocked(true); setNotice(t('requests.sent')); await read(c.signal, true)
    } catch (e) { if (!c.signal.aborted) fail(e, true) }
    finally { finish(c) }
  }
  function openReview(r: AccessRequest) {
    if (busy || locked || !canReview) return
    if (dirty && !window.confirm(t('requests.discard'))) return
    setForm(empty)
    setReview(r); setReviewRevision(catalog?.revision ?? 0); setApprovedNames([...r.endpoint_names]); setApprovedExpiry(r.expires_at); setDecisionReason(''); setAcknowledged(false); setError(''); setFields({}); setNotice('')
  }
  async function decide(action: 'approve' | 'reject') {
    if (!review || busy || locked || !canReview) return
    const invalid: Record<string, string> = {}
    if (!decisionReason.trim()) invalid['decision-reason'] = t('requests.required')
    if (action === 'approve') {
      if (!acknowledged) invalid['decision-ack'] = t('requests.ackHelp')
      if (!approvedNames.length) invalid['decision-endpoints'] = t('requests.pickEndpoints')
      const expiry = Date.parse(approvedExpiry)
      if (!Number.isFinite(expiry) || expiry <= Date.now() || expiry > Date.parse(review.expires_at)) invalid['decision-expiry'] = t('requests.expiryHelp')
      if (!canApprove || review.requester === identity || !reviewRevision) return
    }
    if (Object.keys(invalid).length) { setFields(invalid); setError(t('requests.fixFields')); return }
    const c = new AbortController(); controllers.add(c); writers.add(c); setBusy(true); setError(''); setNotice(''); setFields({})
    try {
      await request(`/access-requests/${encodeURIComponent(review.uid)}/decision`, { method: 'POST', signal: c.signal, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf, ...(action === 'approve' ? { 'If-Match': `"${reviewRevision}"` } : {}) }, body: JSON.stringify({ version: review.version, action, reason: decisionReason, ...(action === 'approve' ? { endpoint_names: approvedNames, expires_at: approvedExpiry } : {}) }) }, 200)
      if (c.signal.aborted) return
      setReview(null); setLocked(true); setNotice(t('requests.decided')); await read(c.signal, true)
    } catch (e) { if (!c.signal.aborted) fail(e, true) }
    finally { finish(c) }
  }
  const field = (id: string) => ({ 'aria-invalid': !!fields[id], 'aria-describedby': fields[id] ? `${id}-error` : undefined })
  const fieldError = (id: string) => fields[id] && <p id={`${id}-error`} className="field-error">{fields[id]}</p>
  const resource = base?.resources.find(r => r.uid === form.resource)
  return <section className="requests-section" aria-busy={busy}>
    <div className="section-heading"><div><h2>{t('requests.workflow')}</h2><p>{t('requests.boundary')}</p></div><div className="actions"><Button variant="outline" disabled={busy} onClick={() => void refresh()}>{t('requests.refresh')}</Button><Button variant="outline" disabled={busy} onClick={() => void refresh(true)}>{t('requests.rebase')}</Button></div></div>
    <p className="hint">{t('lastRead')}: {page ? date(page.observed_at) : t('notRead')} · {t('requests.catalogRevision', { revision: base?.revision ?? '—' })}</p>
    {error && <div className="feedback error" role="alert" tabIndex={-1} ref={errors}><p>{error}</p>{Object.entries(fields).map(([id, text]) => <p key={id}><a href={`#${id}`} onClick={() => document.getElementById(id)?.focus()}>{text}</a></p>)}</div>}
    {notice && <p role="status" className="feedback success">{notice}</p>}
    {(locked || readFailed || (page && now - Date.parse(page.observed_at) > 60000)) && <p className="stale" role="status">{locked ? t('requests.locked') : readFailed ? t('staleFailed') : t('staleAge')}</p>}
    {busy && <p role="status">{t('loading')}</p>}
    {canCreate && <form className="grant-form" onSubmit={e => void submit(e)} noValidate><fieldset disabled={busy || !!review}><legend>{t('requests.new')}</legend><div className="grant-fields">
      <div><label htmlFor="request-source">{t('requests.source')}</label><input id="request-source" {...field('request-source')} value={form.source} onChange={e => setForm({ ...form, source: e.target.value })} maxLength={128} autoCapitalize="none" spellCheck={false} required /><p className="hint">{t('requests.sourceHelp')}</p>{fieldError('request-source')}</div>
      <div><label htmlFor="request-resource">{t('requests.resource')}</label><select id="request-resource" {...field('request-resource')} value={form.resource} onChange={e => setForm({ ...form, resource: e.target.value, endpoints: [] })} required><option value="">{t('requests.pickResource')}</option>{base?.resources.map(r => <option key={r.uid} value={r.uid}>{r.name}</option>)}</select>{fieldError('request-resource')}</div>
      <fieldset id="request-endpoints" tabIndex={-1} {...field('request-endpoints')}><legend>{t('requests.endpoints')}</legend>{resource?.endpoints.map(ep => <label className="endpoint-option" key={ep.name}><input type="checkbox" checked={form.endpoints.includes(ep.name)} onChange={e => setForm({ ...form, endpoints: e.target.checked ? [...form.endpoints, ep.name] : form.endpoints.filter(n => n !== ep.name) })} />{ep.name} {ep.protocol}/{ep.port} {ep.addresses?.join(', ')}</label>)}{fieldError('request-endpoints')}</fieldset>
      <div><label htmlFor="request-duration">{t('requests.duration')}</label><input id="request-duration" {...field('request-duration')} type="number" min={60} max={86400} value={form.duration} onChange={e => setForm({ ...form, duration: e.target.value })} required /><p className="hint">{t('requests.durationHelp')}</p>{fieldError('request-duration')}</div>
      <div><label htmlFor="request-reason">{t('requests.reason')}</label><input id="request-reason" {...field('request-reason')} value={form.reason} onChange={e => setForm({ ...form, reason: e.target.value })} maxLength={512} required />{fieldError('request-reason')}</div>
    </div><Button type="submit" disabled={busy || locked || !base}>{t('requests.send')}</Button></fieldset></form>}
    {review && <form className="grant-form" onSubmit={e => { e.preventDefault(); void decide('approve') }} noValidate><h3>{t('requests.reviewing', { uid: review.uid })}</h3><p>{review.requester} · {review.source} · {review.resource.name}</p><p>{t('requests.requestedUntil', { date: date(review.expires_at) })}</p>
      <fieldset id="decision-endpoints" tabIndex={-1} {...field('decision-endpoints')} disabled={busy}><legend>{t('requests.approvedEntries')}</legend>{review.endpoint_names.map(name => <label className="endpoint-option" key={name}><input type="checkbox" aria-label={t('requests.approveEntry', { name })} checked={approvedNames.includes(name)} onChange={e => setApprovedNames(e.target.checked ? [...approvedNames, name] : approvedNames.filter(n => n !== name))} />{name}</label>)}{fieldError('decision-endpoints')}</fieldset>
      <label htmlFor="decision-expiry">{t('requests.expiry')}</label><input id="decision-expiry" {...field('decision-expiry')} value={approvedExpiry} onChange={e => setApprovedExpiry(e.target.value)} disabled={busy} /><p className="hint">{t('requests.expiryHelp')}</p>{fieldError('decision-expiry')}
      <label htmlFor="decision-reason">{t('requests.decisionReason')}</label><input id="decision-reason" {...field('decision-reason')} value={decisionReason} onChange={e => setDecisionReason(e.target.value)} maxLength={400} disabled={busy} />{fieldError('decision-reason')}
      <label className="endpoint-option"><input id="decision-ack" {...field('decision-ack')} type="checkbox" aria-label={t('requests.ack')} checked={acknowledged} onChange={e => setAcknowledged(e.target.checked)} disabled={busy} />{t('requests.ackHelp')}</label>{fieldError('decision-ack')}
      {review.requester === identity && <p className="boundary">{t('requests.noSelf')}</p>}
      <div className="actions"><Button type="submit" disabled={busy || locked || !canApprove || review.requester === identity || !reviewRevision}>{t('requests.approve')}</Button><Button type="button" variant="outline" disabled={busy || locked} onClick={() => void decide('reject')}>{t('requests.reject')}</Button><Button type="button" variant="ghost" disabled={busy} onClick={() => setReview(null)}>{t('requests.close')}</Button></div>
    </form>}
    <div className="audit-list">{page?.requests.map(r => <article className="request-row" key={r.uid}>
      <div className="audit-meta"><strong>{t(`requests.states.${r.state}`)}</strong><time dateTime={r.created_at}>{date(r.created_at)}</time></div><h3>{r.reason}</h3><p>{r.requester} · {r.source}</p><p>{r.resource.name} · {r.endpoint_names.join(', ')}</p><p>{t('requests.requestedUntil', { date: date(r.expires_at) })}</p><details><summary>{t('requests.details')}</summary><code>{r.uid} · {r.resource.uid}</code>{r.resource.endpoints.map(ep => <p key={ep.name}>{ep.name} {ep.protocol}/{ep.port} {ep.addresses?.join(', ')}</p>)}</details>
      {r.state !== 'pending' && <p>{r.approver}: {r.decision_reason}</p>}
      {r.state === 'approved' && <><p>{t('requests.approvedScope', { names: r.approved_endpoints?.join(', '), date: r.approved_expires_at ? date(r.approved_expires_at) : '—' })}</p><p>{t('requests.grantState', { state: t(`requests.grantStates.${r.grant_state ?? 'unknown'}`) })}</p><p><code>{r.grant_uid}</code> · {t('revisionNumber', { revision: r.revision })}</p><p className="boundary">{t('requests.notApplied')}</p></>}
      {canReview && r.state === 'pending' && <Button variant="outline" disabled={busy || locked} onClick={() => openReview(r)}>{t('requests.review')}</Button>}
    </article>)}</div>
    {page?.requests.length === 0 && !busy && <p className="empty">{t('requests.empty')}</p>}
    {page?.has_more && <Button variant="outline" disabled={busy} onClick={() => void refresh(false, true)}>{t('requests.next')}</Button>}
  </section>
}
