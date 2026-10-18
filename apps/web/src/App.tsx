// SPDX-License-Identifier: Apache-2.0
import { useEffect, useEffectEvent, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { FileText, History, KeyRound, Boxes, Route, ShieldCheck, RefreshCw, LogOut } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { APIError, request } from '@/lib/console'
import type { AuditEvent, AuditPage, Identity, Publication, Session } from '@/lib/console'
import { Grants } from '@/features/grants/Grants'
import { Resources } from '@/features/resources/Resources'
import { HTTPRoutes } from '@/features/http-routes/HTTPRoutes'

type Draft = Publication & { etag: string }
export default function App() {
  const { t, i18n } = useTranslation()
  const [session, setSession] = useState<Session | null>(null)
  const [identity, setIdentity] = useState<Identity | null>(null)
  const [accepted, setAccepted] = useState<Publication | null>(null)
  const [draft, setDraft] = useState<Draft | null>(null)
  const [reason, setReason] = useState('')
  const [key, setKey] = useState('')
  const [tab, setTab] = useState<'publication' | 'audit' | 'grants' | 'resources' | 'routes'>('publication')
  const [routeDirty, setRouteDirty] = useState(false)
  const [routeBusy, setRouteBusy] = useState(false)
  const [grantDirty, setGrantDirty] = useState(false)
  const [grantBusy, setGrantBusy] = useState(false)
  const [resourceDirty, setResourceDirty] = useState(false)
  const [resourceBusy, setResourceBusy] = useState(false)
  const [audit, setAudit] = useState<AuditEvent[]>([])
  const [cursor, setCursor] = useState(0)
  const [hasMore, setHasMore] = useState(false)
  const [busy, setBusy] = useState(true)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [lastRead, setLastRead] = useState(0)
  const [now, setNow] = useState(() => Date.now())
  const [readFailed, setReadFailed] = useState(false)
  const [conflict, setConflict] = useState(false)
  const epoch = useRef(0)
  const errorBox = useRef<HTMLDivElement>(null)
  const [controllers] = useState(() => new Set<AbortController>())
  const dirty = !!draft && !!accepted && (draft.policy !== accepted.policy || draft.facts !== accepted.facts)
  const canWrite = identity?.operations.includes('publication:write') ?? false

  function clear() {
    setRouteDirty(false); setRouteBusy(false)
    epoch.current++
    setIdentity(null); setAccepted(null); setDraft(null); setReason(''); setKey(''); setAudit([]); setCursor(0); setHasMore(false); setLastRead(0); setConflict(false); setGrantDirty(false); setGrantBusy(false); setResourceDirty(false); setResourceBusy(false)
  }
  function fail(e: unknown, reading = false) {
    if (e instanceof DOMException && e.name === 'AbortError') return
    if (e instanceof APIError && e.status === 401) {
      clear(); setSession(s => s ? { ...s, authenticated: false } : null); setError(t('expired')); return
    }
    if (e instanceof APIError && e.status === 412) { setConflict(true); setError(t('conflict')); return }
    const code = e instanceof APIError ? e.message : 'request_failed'
    setError(t(`errors.${code}`, { defaultValue: t('requestFailed') }))
    if (reading) setReadFailed(true)
  }
  async function load(signal?: AbortSignal, replaceDraft = false) {
    const generation = epoch.current
    const [me, doc, events] = await Promise.all([
      request<Identity>('/me', { signal }), request<Publication>('/publication', { signal }), request<AuditPage>('/audit?limit=50', { signal }),
    ])
    if (generation !== epoch.current) return
    if (!/^"\d+"$/.test(doc.etag)) throw new Error('Missing strong revision')
    setIdentity(me.data); setAccepted(doc.data)
    // A read must not advance the editor's CAS base behind a preserved draft.
    setDraft(previous => replaceDraft || previous === null ? { ...doc.data, etag: doc.etag } : previous)
    if (replaceDraft) { setReason(''); setConflict(false) }
    setAudit(events.data.events ?? []); setCursor(events.data.next_after); setHasMore((events.data.events?.length ?? 0) === 50)
    setLastRead(Date.now()); setReadFailed(false)
  }
  const bootstrap = useEffectEvent(async (controller: AbortController) => {
      try {
        const { data } = await request<Session>('/session', { signal: controller.signal }); setSession(data)
        if (data.authenticated) await load(controller.signal)
      } catch (e) { fail(e, true) } finally { if (!controller.signal.aborted) setBusy(false) }
  })
  const invalidate = useEffectEvent(() => { epoch.current++ })
  useEffect(() => {
    const controller = new AbortController(); controllers.add(controller)
    // Bootstrap crosses the network before updating state; schedule it outside
    // synchronous effect setup so StrictMode cleanup can cancel the first run.
    queueMicrotask(() => { if (!controller.signal.aborted) void bootstrap(controller) })
    const timer = window.setInterval(() => setNow(Date.now()), 15000)
    return () => { for (const request of controllers) request.abort(); controllers.clear(); invalidate(); window.clearInterval(timer) }
    // Mount owns bootstrap. Language changes do not create another session.
  }, [controllers])
  useEffect(() => { if (error) errorBox.current?.focus() }, [error])
  useEffect(() => {
    if (!dirty && !grantDirty && !resourceDirty && !routeDirty) return
    const prevent = (e: BeforeUnloadEvent) => e.preventDefault()
    window.addEventListener('beforeunload', prevent)
    return () => window.removeEventListener('beforeunload', prevent)
  }, [dirty, grantDirty, resourceDirty, routeDirty])

  async function refresh(replaceDraft = false) {
    if (busy) return
    if (replaceDraft && dirty && !window.confirm(t('discard'))) return
    setBusy(true); setError(''); setNotice('')
    const controller = new AbortController(); controllers.add(controller)
    try {
      const { data } = await request<Session>('/session', { signal: controller.signal }); setSession(data)
      if (data.authenticated) await load(controller.signal, replaceDraft)
      else { clear(); setError(t('expired')) }
    } catch (e) { fail(e, true) } finally { controllers.delete(controller); setBusy(false) }
  }
  async function login(e: React.FormEvent) {
    e.preventDefault(); if (busy) return
    setBusy(true); setError(''); setNotice('')
    try {
      const { data } = await request<Session>('/session'); setSession(data)
      await request('/login', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': data.csrf }, body: JSON.stringify({ token: key }) })
      setKey(''); setSession({ ...data, authenticated: true }); await load(undefined, true)
    } catch (e) { setKey(''); if (e instanceof APIError && e.status === 401) setError(t('errors.unauthorized')); else fail(e) } finally { setBusy(false) }
  }
  async function logout() {
    if (busy || grantBusy || resourceBusy || routeBusy || ((dirty || grantDirty || resourceDirty || routeDirty) && !window.confirm(t('routes.discardAll')))) return
    setBusy(true); setError('')
    try { await request('/logout', { method: 'POST', headers: { 'X-CSRF-Token': session?.csrf ?? '' } }); clear(); setSession(s => s ? { ...s, authenticated: false } : null) }
    catch (e) { fail(e) } finally { setBusy(false) }
  }
  async function save(e: React.FormEvent) {
    e.preventDefault(); if (busy || !draft || !canWrite || conflict) return
    setBusy(true); setError(''); setNotice('')
    try {
      const result = await request<Publication>('/publication', { method: 'PUT', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': session?.csrf ?? '', 'If-Match': draft.etag }, body: JSON.stringify({ policy: draft.policy, facts: draft.facts, reason }) })
      setAccepted(result.data); setDraft({ ...result.data, etag: result.etag }); setReason(''); setNotice(t('saved')); setLastRead(Date.now()); setReadFailed(false)
      try { const events = await request<AuditPage>('/audit?limit=50'); setAudit(events.data.events ?? []); setCursor(events.data.next_after); setHasMore((events.data.events?.length ?? 0) === 50) } catch (e) { fail(e, true) }
    } catch (e) { fail(e) } finally { setBusy(false) }
  }
  async function moreAudit() {
    if (busy) return
    setBusy(true); setError('')
    try { const { data } = await request<AuditPage>(`/audit?after=${cursor}&limit=50`); setAudit(old => [...old, ...(data.events ?? [])]); setCursor(data.next_after); setHasMore((data.events?.length ?? 0) === 50) } catch (e) { fail(e, true) } finally { setBusy(false) }
  }
  const language = <Button variant="ghost" onClick={() => void i18n.changeLanguage(i18n.language === 'en' ? 'zh-CN' : 'en')} aria-label={t('language')}>{i18n.language === 'en' ? '简体中文' : 'English'}</Button>
  const errorPanel = error && <div className="feedback error" role="alert" tabIndex={-1} ref={errorBox}>{error}</div>
  if (!session?.authenticated) return <div className="login-shell">
    <header className="login-header"><span className="brand"><ShieldCheck aria-hidden="true" />PicketX</span>{language}</header>
    <main className="login-main"><section className="login-intro"><h1>{t('loginTitle')}</h1><p>{t('loginDescription')}</p><p className="boundary">{t('desiredBoundary')}</p></section>
      <form className="login-panel" onSubmit={e => void login(e)} aria-busy={busy}><h2>{t('signIn')}</h2>{errorPanel}
        <label htmlFor="access-key">{t('accessKey')}</label><input id="access-key" type="password" autoComplete="current-password" value={key} onChange={e => setKey(e.target.value)} minLength={32} maxLength={256} required disabled={busy} aria-describedby="key-help" />
        <p id="key-help" className="hint">{t('keyHelp')}</p><Button type="submit" disabled={busy}>{busy ? t('loading') : t('signIn')}</Button>
        {!session && !busy && <Button type="button" variant="outline" onClick={() => window.location.reload()}>{t('retry')}</Button>}
      </form>
    </main><footer>{t('sessionBoundary')}</footer>
  </div>
  return <div className="console-shell">
    <aside className="sidebar"><span className="brand"><ShieldCheck aria-hidden="true" />PicketX</span><p className="sidebar-subtitle">{t('operatorConsole')}</p>
      <nav aria-label={t('navigation')}><Button variant={tab === 'publication' ? 'secondary' : 'ghost'} aria-current={tab === 'publication' ? 'page' : undefined} onClick={() => setTab('publication')}><FileText aria-hidden="true" />{t('configuration')}</Button><Button variant={tab === 'resources' ? 'secondary' : 'ghost'} aria-current={tab === 'resources' ? 'page' : undefined} onClick={() => setTab('resources')}><Boxes aria-hidden="true" />{t('resources.title')}</Button><Button variant={tab === 'grants' ? 'secondary' : 'ghost'} aria-current={tab === 'grants' ? 'page' : undefined} onClick={() => setTab('grants')}><KeyRound aria-hidden="true" />{t('grants.title')}</Button><Button variant={tab === 'routes' ? 'secondary' : 'ghost'} aria-current={tab === 'routes' ? 'page' : undefined} onClick={() => setTab('routes')}><Route aria-hidden="true" />{t('routes.title')}</Button><Button variant={tab === 'audit' ? 'secondary' : 'ghost'} aria-current={tab === 'audit' ? 'page' : undefined} onClick={() => { setTab('audit'); void refresh() }}><History aria-hidden="true" />{t('audit')}</Button></nav>
      <div className="identity"><span>{identity?.id ?? t('loading')}</span><small>{identity?.roles.join(', ')}</small>{language}<Button variant="outline" disabled={busy || grantBusy || resourceBusy || routeBusy} onClick={() => void logout()}><LogOut aria-hidden="true" />{t('signOut')}</Button></div>
    </aside>
    <main className="workspace" aria-busy={busy}>
      <header className="workspace-header"><div><h1>{tab === 'publication' ? t('configuration') : tab === 'grants' ? t('grants.title') : tab === 'resources' ? t('resources.title') : tab === 'routes' ? t('routes.title') : t('audit')}</h1><p>{t('desiredBoundary')}</p></div>{tab !== 'grants' && tab !== 'resources' && tab !== 'routes' && <Button variant="outline" disabled={busy} onClick={() => void refresh()}><RefreshCw aria-hidden="true" />{t('refresh')}</Button>}</header>
      <section className="revision-strip" aria-label={t('status')}><div><span>{t('authority')}</span><strong>{accepted?.authority ?? '—'}</strong></div><div><span>{t('editorRevision')}</span><strong>{draft?.etag.slice(1, -1) ?? '—'}</strong></div><div><span>{t('lastRead')}</span><strong>{lastRead ? new Date(lastRead).toLocaleTimeString(i18n.language) : t('notRead')}</strong></div><div><span>{t('enforcement')}</span><strong>{t('notConfirmed')}</strong></div></section>
      {errorPanel}{notice && <div className="feedback success" role="status">{notice}</div>}
      {readFailed ? <p className="stale" role="status">{t('staleFailed')}</p> : lastRead > 0 && now - lastRead > 60000 ? <p className="stale" role="status">{t('staleAge')}</p> : null}
      {busy && <p role="status">{t('loading')}</p>}
      <div hidden={tab !== 'grants'}><Grants csrf={session.csrf} canWrite={canWrite} onUnauthorized={() => fail(new APIError(401, 'unauthorized'))} onDirty={setGrantDirty} onBusy={setGrantBusy} /></div>
      <div hidden={tab !== 'resources'}><Resources csrf={session.csrf} canWrite={canWrite} onUnauthorized={() => fail(new APIError(401, 'unauthorized'))} onDirty={setResourceDirty} onBusy={setResourceBusy} /></div>
      <div hidden={tab !== 'routes'}><HTTPRoutes csrf={session.csrf} canWrite={canWrite} onUnauthorized={() => fail(new APIError(401, 'unauthorized'))} onDirty={setRouteDirty} onBusy={setRouteBusy} /></div>
      {tab === 'publication' ? <section className="editor-section"><div className="section-heading"><div><h2>{t('policyAndFacts')}</h2><p>{canWrite ? t('editorHelp') : t('readOnly')}</p></div><span className="draft-state">{dirty ? t('unsaved') : t('inSync')}</span></div>
        {draft ? <form onSubmit={e => void save(e)}><div className="editors"><div><label htmlFor="policy">{t('policyYaml')}</label><textarea id="policy" spellCheck={false} value={draft.policy} onChange={e => setDraft({ ...draft, policy: e.target.value })} readOnly={!canWrite} disabled={busy} required /></div><div><label htmlFor="facts">{t('factsJson')}</label><textarea id="facts" spellCheck={false} value={draft.facts} onChange={e => setDraft({ ...draft, facts: e.target.value })} readOnly={!canWrite} disabled={busy} required /></div></div>
          <div className="save-area"><div><label htmlFor="reason">{t('changeReason')}</label><input id="reason" value={reason} onChange={e => setReason(e.target.value)} maxLength={512} disabled={!canWrite || busy} required /><p className="hint">{t('reasonHelp')}</p></div><div className="actions"><Button type="button" variant="outline" disabled={busy} onClick={() => void refresh(true)}>{t('reloadLatest')}</Button><Button type="submit" disabled={busy || !canWrite || !reason.trim() || conflict}>{t('save')}</Button></div></div>
        </form> : !busy && <p>{t('noConfiguration')}</p>}
      </section> : tab === 'audit' ? <section className="audit-section"><h2>{t('publicationHistory')}</h2><p>{t('auditHelp')}</p>{audit.length === 0 && !busy ? <p className="empty">{t('emptyAudit')}</p> : <div className="audit-list">{audit.map(event => <article key={event.id}><div className="audit-meta"><strong>{t('revisionNumber', { revision: event.revision })}</strong><time dateTime={event.at}>{new Date(event.at).toLocaleString(i18n.language)}</time></div><h3>{event.reason}</h3><p>{event.actor}</p><details><summary>{t('digest')}</summary><code>{event.digest}</code></details></article>)}</div>}{hasMore && <Button variant="outline" disabled={busy} onClick={() => void moreAudit()}>{t('moreAudit')}</Button>}</section> : null}
    </main>
  </div>
}
