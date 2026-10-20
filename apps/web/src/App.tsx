// SPDX-License-Identifier: Apache-2.0
import { useEffect, useEffectEvent, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { FileText, History, KeyRound, Boxes, Route, ShieldCheck, RefreshCw, LogOut } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { APIError, request } from '@/lib/console'
import type { AuditEvent, AuditPage, Identity, Publication, Session } from '@/lib/console'
import { Grants } from '@/features/grants/Grants'
import { Resources } from '@/features/resources/Resources'
import { Nodes } from '@/features/nodes/Nodes'
import { HTTPRoutes } from '@/features/http-routes/HTTPRoutes'
import { Requests } from '@/features/requests/Requests'
import { Accounts } from '@/features/accounts/Accounts'
import { HTTPAudit } from '@/features/http-audit/HTTPAudit'

type Draft = Publication & { etag: string }
export default function App() {
  const { t, i18n } = useTranslation()
  const [session, setSession] = useState<Session | null>(null)
  const [identity, setIdentity] = useState<Identity | null>(null)
  const [accepted, setAccepted] = useState<Publication | null>(null)
  const [draft, setDraft] = useState<Draft | null>(null)
  const [reason, setReason] = useState('')
  const [key, setKey] = useState('')
  const [signInMethod, setSignInMethod] = useState<'key' | 'password'>('key')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [tab, setTab] = useState<'publication' | 'audit' | 'grants' | 'resources' | 'routes' | 'nodes' | 'requests' | 'accounts'>('publication')
  const [accountDirty, setAccountDirty] = useState(false)
  const [accountBusy, setAccountBusy] = useState(false)
  const [requestDirty, setRequestDirty] = useState(false)
  const [requestBusy, setRequestBusy] = useState(false)
  const [routeDirty, setRouteDirty] = useState(false)
  const [routeBusy, setRouteBusy] = useState(false)
  const [grantDirty, setGrantDirty] = useState(false)
  const [grantBusy, setGrantBusy] = useState(false)
  const [resourceDirty, setResourceDirty] = useState(false)
  const [resourceBusy, setResourceBusy] = useState(false)
  const [audit, setAudit] = useState<AuditEvent[]>([])
  const [auditView, setAuditView] = useState<'publication' | 'http'>('publication')
  const [cursor, setCursor] = useState(0)
  const [hasMore, setHasMore] = useState(false)
  const [busy, setBusy] = useState(true)
  const [error, setError] = useState('')
  const oidcFailure = useRef(new URLSearchParams(window.location.search).get('login_error') === 'oidc')
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
  const canRead = identity?.operations.includes('publication:read') ?? false
  const canRequests = identity?.operations.some(op => op.startsWith('request:')) ?? false
  const canAccounts = identity?.operations.includes('account:read') ?? false
  const canAudit = identity?.operations.includes('audit:read') ?? false
  const passwordMode = session?.password_login && signInMethod === 'password'

  function clear() {
    setAccountDirty(false); setAccountBusy(false)
    setSignInMethod('key'); setUsername(''); setPassword(''); setShowPassword(false)
    setRouteDirty(false); setRouteBusy(false)
    setAuditView('publication')
    setRequestDirty(false); setRequestBusy(false)
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
    const me = await request<Identity>('/me', { signal })
    if (generation !== epoch.current) return
    // Observed capabilities are authoritative even if a dependent view fails.
    setIdentity(me.data)
    const operations = me.data.operations
    const publicationRead = operations.includes('publication:read')
    const requestRead = operations.some(op => op.startsWith('request:'))
    const accountRead = operations.includes('account:read')
    const auditRead = operations.includes('audit:read')
    setTab(previous => (previous === 'audit' ? auditRead : previous === 'requests' ? requestRead : previous === 'accounts' ? accountRead : publicationRead)
      ? previous : publicationRead ? 'publication' : requestRead ? 'requests' : accountRead ? 'accounts' : auditRead ? 'audit' : 'publication')
    if (!auditRead) { setAudit([]); setCursor(0); setHasMore(false) }
    const [doc, events] = await Promise.all([
      me.data.operations.includes('publication:read') ? request<Publication>('/publication', { signal }) : Promise.resolve(null),
      me.data.operations.includes('audit:read') ? request<AuditPage>('/audit?limit=50', { signal }) : Promise.resolve(null),
    ])
    if (generation !== epoch.current) return
    if (!doc) {
      setAccepted(null); setDraft(null)
      setAudit(events?.data.events ?? []); setCursor(events?.data.next_after ?? 0); setHasMore((events?.data.events?.length ?? 0) === 50)
      setLastRead(Date.now()); setReadFailed(false); return
    }
    if (!/^"\d+"$/.test(doc.etag)) throw new Error('Missing strong revision')
    setAccepted(doc.data)
    // A read must not advance the editor's CAS base behind a preserved draft.
    setDraft(previous => replaceDraft || previous === null ? { ...doc.data, etag: doc.etag } : previous)
    if (replaceDraft) { setReason(''); setConflict(false) }
    setAudit(events?.data.events ?? []); setCursor(events?.data.next_after ?? 0); setHasMore((events?.data.events?.length ?? 0) === 50)
    setLastRead(Date.now()); setReadFailed(false)
  }
  const bootstrap = useEffectEvent(async (controller: AbortController) => {
      try {
        const { data } = await request<Session>('/session', { signal: controller.signal }); setSession(data)
        if (data.authenticated) await load(controller.signal)
        if (oidcFailure.current) { oidcFailure.current = false; window.history.replaceState(null, '', '/console/'); setError(t('oidc.rejected')) }
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
    if (!dirty && !grantDirty && !resourceDirty && !routeDirty && !requestDirty && !accountDirty) return
    const prevent = (e: BeforeUnloadEvent) => e.preventDefault()
    window.addEventListener('beforeunload', prevent)
    return () => window.removeEventListener('beforeunload', prevent)
  }, [dirty, grantDirty, resourceDirty, routeDirty, requestDirty, accountDirty])

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
    if (passwordMode && (new TextEncoder().encode(password).length < 12 || new TextEncoder().encode(password).length > 72)) { setError(t('humanLogin.passwordLimit')); return }
    setBusy(true); setError(''); setNotice('')
    try {
      const { data } = await request<Session>('/session'); setSession(data)
      if (signInMethod === 'password' && !data.password_login) { setSignInMethod('key'); setError(t('humanLogin.unavailable')); return }
      await request(signInMethod === 'password' ? '/login/password' : '/login', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': data.csrf }, body: JSON.stringify(signInMethod === 'password' ? { username, password } : { token: key }) }, 200)
      setKey(''); setPassword(''); setUsername(''); setShowPassword(false); setSession({ ...data, authenticated: true }); await load(undefined, true)
    } catch (e) { if (e instanceof APIError && e.status === 401) setError(t(passwordMode ? 'humanLogin.rejected' : 'errors.unauthorized')); else fail(e) } finally { setKey(''); setPassword(''); setShowPassword(false); setBusy(false) }
  }
  async function logout() {
    if (busy || grantBusy || resourceBusy || routeBusy || requestBusy || accountBusy || ((dirty || grantDirty || resourceDirty || routeDirty || requestDirty || accountDirty) && !window.confirm(t('routes.discardAll')))) return
    setBusy(true); setError('')
    try { await request('/logout', { method: 'POST', headers: { 'X-CSRF-Token': session?.csrf ?? '' } }); clear(); setSession(s => s ? { ...s, authenticated: false } : null) }
    catch (e) { fail(e) } finally { setBusy(false) }
  }
  async function loginOIDC() {
    if (busy) return
    setBusy(true); setError(''); setKey(''); setPassword(''); setUsername(''); setShowPassword(false)
    try {
      const { data } = await request<Session>('/session'); setSession(data)
      if (!data.oidc_login) { setError(t('oidc.unavailable')); return }
      const result = await request<{ url: string }>('/oidc/start', { method: 'POST', headers: { 'X-CSRF-Token': data.csrf } }, 200)
      // Only the trusted Controller supplies this destination, never a URL query.
      const target = new URL(result.data.url)
      if (target.protocol !== 'https:' && target.protocol !== 'http:') throw new Error('Invalid sign-in destination')
      window.location.assign(target.href)
    } catch (e) { if (e instanceof APIError && e.message === 'oidc_unavailable') setError(t('oidc.unavailable')); else fail(e) }
    finally { setBusy(false) }
  }
  async function save(e: React.FormEvent) {
    e.preventDefault(); if (busy || !draft || !canWrite || conflict) return
    setBusy(true); setError(''); setNotice('')
    try {
      const result = await request<Publication>('/publication', { method: 'PUT', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': session?.csrf ?? '', 'If-Match': draft.etag }, body: JSON.stringify({ policy: draft.policy, facts: draft.facts, reason }) }, 200)
      setAccepted(result.data); setDraft({ ...result.data, etag: result.etag }); setReason(''); setNotice(t('saved')); setLastRead(Date.now()); setReadFailed(false)
      try { const events = await request<AuditPage>('/audit?limit=50'); setAudit(events.data.events ?? []); setCursor(events.data.next_after); setHasMore((events.data.events?.length ?? 0) === 50) } catch (e) { fail(e, true) }
    } catch (e) {
      // An unknown outcome is not proof of rollback. Inspect and explicitly
      // reload the editor before attempting another publication.
      if (!(e instanceof APIError) || e.status >= 500) setConflict(true)
      fail(e)
    } finally { setBusy(false) }
  }
  async function moreAudit() {
    if (busy) return
    setBusy(true); setError('')
    try { const { data } = await request<AuditPage>(`/audit?after=${cursor}&limit=50`); setAudit(old => [...old, ...(data.events ?? [])]); setCursor(data.next_after); setHasMore((data.events?.length ?? 0) === 50) } catch (e) { fail(e, true) } finally { setBusy(false) }
  }
  const language = <Button variant="ghost" onClick={() => void i18n.changeLanguage(i18n.language === 'en' ? 'zh-CN' : 'en')} aria-label={t('language')}>{i18n.language === 'en' ? '简体中文' : 'English'}</Button>
  const errorPanel = error && <div id="console-error" className="feedback error" role="alert" tabIndex={-1} ref={errorBox}>{error}</div>
  if (!session?.authenticated) return <div className="login-shell">
    <header className="login-header"><span className="brand"><ShieldCheck aria-hidden="true" />PicketX</span>{language}</header>
    <main className="login-main"><section className="login-intro"><h1>{t('loginTitle')}</h1><p>{t('loginDescription')}</p><p className="boundary">{t('desiredBoundary')}</p></section>
      <form className="login-panel" onSubmit={e => void login(e)} aria-busy={busy}><h2>{t('signIn')}</h2>{errorPanel}
        {session?.oidc_login && <div className="mb-6"><Button type="button" variant="outline" className="w-full" disabled={busy} onClick={() => void loginOIDC()} aria-describedby="oidc-help">{busy ? t('loading') : t('oidc.continue')}</Button><p id="oidc-help" className="hint">{t('oidc.help')}</p></div>}
        {session?.password_login && <><label htmlFor="sign-in-method">{t('humanLogin.method')}</label><select id="sign-in-method" value={signInMethod} disabled={busy} onChange={e => { setSignInMethod(e.target.value as 'key' | 'password'); setKey(''); setPassword(''); setUsername(''); setShowPassword(false); setError('') }}><option value="key">{t('accessKey')}</option><option value="password">{t('humanLogin.methodPassword')}</option></select></>}
        {passwordMode ? <>
          <label htmlFor="username">{t('humanLogin.username')}</label><input id="username" name="username" autoComplete="username" autoCapitalize="none" spellCheck={false} value={username} onChange={e => setUsername(e.target.value)} minLength={3} maxLength={64} required disabled={busy} aria-describedby="password-help" />
          <label htmlFor="human-password">{t('humanLogin.password')}</label><input id="human-password" name="password" type={showPassword ? 'text' : 'password'} autoComplete="current-password" value={password} onChange={e => setPassword(e.target.value)} maxLength={72} required disabled={busy} aria-describedby={error ? 'password-help console-error' : 'password-help'} />
          <Button type="button" variant="ghost" disabled={busy} aria-pressed={showPassword} onClick={() => setShowPassword(v => !v)}>{t(showPassword ? 'humanLogin.hide' : 'humanLogin.show')}</Button>
          <p id="password-help" className="hint">{t('humanLogin.help')}</p>
        </> : <><label htmlFor="access-key">{t('accessKey')}</label><input id="access-key" type="password" autoComplete="current-password" value={key} onChange={e => setKey(e.target.value)} minLength={32} maxLength={256} required disabled={busy} aria-describedby="key-help" /><p id="key-help" className="hint">{t('keyHelp')}</p></>}
        <Button type="submit" disabled={busy}>{busy ? t('loading') : t('signIn')}</Button>
        {!session && !busy && <Button type="button" variant="outline" onClick={() => window.location.reload()}>{t('retry')}</Button>}
      </form>
    </main><footer>{t('sessionBoundary')}</footer>
  </div>
  return <div className="console-shell">
    <aside className="sidebar"><span className="brand"><ShieldCheck aria-hidden="true" />PicketX</span><p className="sidebar-subtitle">{t('operatorConsole')}</p>
      <nav aria-label={t('navigation')}>{canRead && <><Button variant={tab === 'publication' ? 'secondary' : 'ghost'} aria-current={tab === 'publication' ? 'page' : undefined} onClick={() => setTab('publication')}><FileText aria-hidden="true" />{t('configuration')}</Button><Button variant={tab === 'resources' ? 'secondary' : 'ghost'} aria-current={tab === 'resources' ? 'page' : undefined} onClick={() => setTab('resources')}><Boxes aria-hidden="true" />{t('resources.title')}</Button><Button variant={tab === 'grants' ? 'secondary' : 'ghost'} aria-current={tab === 'grants' ? 'page' : undefined} onClick={() => setTab('grants')}><KeyRound aria-hidden="true" />{t('grants.title')}</Button><Button variant={tab === 'routes' ? 'secondary' : 'ghost'} aria-current={tab === 'routes' ? 'page' : undefined} onClick={() => setTab('routes')}><Route aria-hidden="true" />{t('routes.title')}</Button>{identity?.operations.includes('audit:read') && <Button variant={tab === 'audit' ? 'secondary' : 'ghost'} aria-current={tab === 'audit' ? 'page' : undefined} onClick={() => { setTab('audit'); void refresh() }}><History aria-hidden="true" />{t('audit')}</Button>}<Button variant={tab === 'nodes' ? 'secondary' : 'ghost'} aria-current={tab === 'nodes' ? 'page' : undefined} onClick={() => setTab('nodes')}><ShieldCheck aria-hidden="true" />{t('nodes.title')}</Button></>}{canRequests && <Button variant={tab === 'requests' ? 'secondary' : 'ghost'} aria-current={tab === 'requests' ? 'page' : undefined} onClick={() => setTab('requests')}><FileText aria-hidden="true" />{t('requests.title')}</Button>}</nav>
      {!canRead && canAudit && <nav aria-label={t('audit')}><Button variant={tab === 'audit' ? 'secondary' : 'ghost'} aria-current={tab === 'audit' ? 'page' : undefined} onClick={() => { setTab('audit'); void refresh() }}><History aria-hidden="true" />{t('audit')}</Button></nav>}
      {canAccounts && <nav aria-label={t('accounts.title')}><Button variant={tab === 'accounts' ? 'secondary' : 'ghost'} aria-current={tab === 'accounts' ? 'page' : undefined} onClick={() => setTab('accounts')}><KeyRound aria-hidden="true" />{t('accounts.title')}</Button></nav>}
      <div className="identity"><span>{identity?.id ?? t('loading')}</span><small>{identity?.roles.join(', ')}</small>{language}<Button variant="outline" disabled={busy || grantBusy || resourceBusy || routeBusy || requestBusy || accountBusy} onClick={() => void logout()}><LogOut aria-hidden="true" />{t('signOut')}</Button></div>
    </aside>
    <main className="workspace" aria-busy={busy}>
      <header className="workspace-header"><div><h1>{tab === 'accounts' ? t('accounts.title') : tab === 'requests' ? t('requests.title') : tab === 'publication' ? t('configuration') : tab === 'grants' ? t('grants.title') : tab === 'resources' ? t('resources.title') : tab === 'routes' ? t('routes.title') : tab === 'nodes' ? t('nodes.title') : t('audit')}</h1><p>{tab === 'accounts' ? t('accounts.boundary') : t('desiredBoundary')}</p></div>{tab !== 'accounts' && tab !== 'requests' && tab !== 'grants' && tab !== 'resources' && tab !== 'routes' && tab !== 'nodes' && <Button variant="outline" disabled={busy} onClick={() => void refresh()}><RefreshCw aria-hidden="true" />{t('refresh')}</Button>}</header>
      {tab !== 'accounts' && <section className="revision-strip" aria-label={t('status')}><div><span>{t('authority')}</span><strong>{accepted?.authority ?? '—'}</strong></div><div><span>{t('editorRevision')}</span><strong>{draft?.etag.slice(1, -1) ?? '—'}</strong></div><div><span>{t('lastRead')}</span><strong>{lastRead ? new Date(lastRead).toLocaleTimeString(i18n.language) : t('notRead')}</strong></div><div><span>{t('enforcement')}</span><strong>{t('notConfirmed')}</strong></div></section>}
      {errorPanel}{notice && <div className="feedback success" role="status">{notice}</div>}
      {readFailed ? <p className="stale" role="status">{t('staleFailed')}</p> : lastRead > 0 && now - lastRead > 60000 ? <p className="stale" role="status">{t('staleAge')}</p> : null}
      {busy && <p role="status">{t('loading')}</p>}
      {canRead && <><div hidden={tab !== 'grants'}><Grants csrf={session.csrf} canWrite={canWrite} onUnauthorized={() => fail(new APIError(401, 'unauthorized'))} onDirty={setGrantDirty} onBusy={setGrantBusy} /></div>
      <div hidden={tab !== 'resources'}><Resources csrf={session.csrf} canWrite={canWrite} onUnauthorized={() => fail(new APIError(401, 'unauthorized'))} onDirty={setResourceDirty} onBusy={setResourceBusy} /></div>
      <div hidden={tab !== 'routes'}><HTTPRoutes csrf={session.csrf} canWrite={canWrite} onUnauthorized={() => fail(new APIError(401, 'unauthorized'))} onDirty={setRouteDirty} onBusy={setRouteBusy} /></div>
      {tab === 'nodes' && <Nodes onUnauthorized={() => fail(new APIError(401, 'unauthorized'))} />}</>}
      {canRequests && identity && <div hidden={tab !== 'requests'}><Requests key={`${identity.id}-${identity.operations.join(',')}`} active={tab === 'requests'} csrf={session.csrf} identity={identity.id} canCreate={identity.operations.includes('request:create')} canList={identity.operations.includes('request:read-own') || identity.operations.includes('request:review')} canReview={identity.operations.includes('request:review')} canApprove={canWrite && identity.operations.includes('request:review')} onUnauthorized={() => fail(new APIError(401, 'unauthorized'))} onDirty={setRequestDirty} onBusy={setRequestBusy} /></div>}
      {canAccounts && identity && <div hidden={tab !== 'accounts'}><Accounts key={`${identity.id}-${identity.operations.join(',')}`} active={tab === 'accounts'} csrf={session.csrf} canWrite={identity.operations.includes('account:write')} onUnauthorized={() => fail(new APIError(401, 'unauthorized'))} onDirty={setAccountDirty} onBusy={setAccountBusy} /></div>}
      {tab === 'publication' ? <section className="editor-section"><div className="section-heading"><div><h2>{t('policyAndFacts')}</h2><p>{canWrite ? t('editorHelp') : t('readOnly')}</p></div><span className="draft-state">{dirty ? t('unsaved') : t('inSync')}</span></div>
        {conflict && <p className="stale" role="status">{t('publicationConflictHelp')}</p>}
        {draft ? <form onSubmit={e => void save(e)}><div className="editors"><div><label htmlFor="policy">{t('policyYaml')}</label><textarea id="policy" spellCheck={false} value={draft.policy} onChange={e => setDraft({ ...draft, policy: e.target.value })} readOnly={!canWrite} disabled={busy} required /></div><div><label htmlFor="facts">{t('factsJson')}</label><textarea id="facts" spellCheck={false} value={draft.facts} onChange={e => setDraft({ ...draft, facts: e.target.value })} readOnly={!canWrite} disabled={busy} required /></div></div>
          <div className="save-area"><div><label htmlFor="reason">{t('changeReason')}</label><input id="reason" value={reason} onChange={e => setReason(e.target.value)} maxLength={512} disabled={!canWrite || busy} required /><p className="hint">{t('reasonHelp')}</p></div><div className="actions"><Button type="button" variant="outline" disabled={busy} onClick={() => void refresh(true)}>{t('reloadLatest')}</Button><Button type="submit" disabled={busy || !canWrite || !reason.trim() || conflict}>{t('save')}</Button></div></div>
        </form> : !busy && <p>{t('noConfiguration')}</p>}
      </section> : tab === 'audit' && canAudit ? <>
        <div className="audit-switch" role="group" aria-label={t('httpAudit.view')}>
          <Button variant={auditView === 'publication' ? 'secondary' : 'outline'} aria-pressed={auditView === 'publication'} onClick={() => setAuditView('publication')}>{t('publicationHistory')}</Button>
          <Button variant={auditView === 'http' ? 'secondary' : 'outline'} aria-pressed={auditView === 'http'} onClick={() => setAuditView('http')}>{t('httpAudit.navigation')}</Button>
        </div>
        {auditView === 'http' && identity ? <HTTPAudit key={`${identity.id}-${identity.operations.join(',')}`} onUnauthorized={() => fail(new APIError(401, 'unauthorized'))} /> : <section className="audit-section"><h2>{t('publicationHistory')}</h2><p>{t('auditHelp')}</p>{audit.length === 0 && !busy ? <p className="empty">{t('emptyAudit')}</p> : <div className="audit-list">{audit.map(event => <article key={event.id}><div className="audit-meta"><strong>{t('revisionNumber', { revision: event.revision })}</strong><time dateTime={event.at}>{new Date(event.at).toLocaleString(i18n.language)}</time></div><h3>{event.reason}</h3><p>{event.actor}</p><details><summary>{t('digest')}</summary><code>{event.digest}</code></details></article>)}</div>}{hasMore && <Button variant="outline" disabled={busy} onClick={() => void moreAudit()}>{t('moreAudit')}</Button>}</section>}
      </> : null}
    </main>
  </div>
}
