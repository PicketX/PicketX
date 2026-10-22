// SPDX-License-Identifier: Apache-2.0
import { useEffect, useEffectEvent, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { APIError, request } from '@/lib/console'

type Account = { id: string; username: string; roles: string[]; disabled: boolean; version: number }
type Event = { actor: string; accountId: string; operation: string; reason: string; revision: number; at: string }
type Page = { mode: string; revision: number; accounts: Account[]; events: Event[]; observedAt: string }
type Form = { id: string; username: string; password: string; roles: string[]; disabled: boolean; reason: string; acknowledged: boolean; version: number }
const empty: Form = { id: '', username: '', password: '', roles: ['role:requester'], disabled: false, reason: '', acknowledged: false, version: 0 }
const roles = ['role:admin', 'role:viewer', 'role:requester', 'role:blocked']
type Props = { active: boolean; csrf: string; canWrite: boolean; onUnauthorized: () => void; onDirty: (value: boolean) => void; onBusy: (value: boolean) => void }

export function Accounts({ active, csrf, canWrite, onUnauthorized, onDirty, onBusy }: Props) {
  const { t, i18n } = useTranslation()
  const [page, setPage] = useState<Page | null>(null)
  const [base, setBase] = useState<number | null>(null)
  const [form, setForm] = useState<Form>(empty)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [fields, setFields] = useState<Record<string, string>>({})
  const [notice, setNotice] = useState('')
  const [locked, setLocked] = useState(false)
  const [readFailed, setReadFailed] = useState(false)
  const [now, setNow] = useState(() => Date.now())
  const errors = useRef<HTMLDivElement>(null)
  const readEpoch = useRef(0)
  const [controllers] = useState(() => new Set<AbortController>())
  const [writers] = useState(() => new Set<AbortController>())
  const dirty = form.id !== '' || form.username !== '' || form.password !== '' || form.reason !== '' || form.roles.join(',') !== empty.roles.join(',') || form.disabled
  useEffect(() => { onDirty(dirty) }, [dirty, onDirty])
  useEffect(() => { onBusy(busy) }, [busy, onBusy])
  useEffect(() => { if (error) errors.current?.focus() }, [error, fields])
  const date = (at: string) => new Date(at).toLocaleString(i18n.language)
  function finish(c: AbortController) { controllers.delete(c); writers.delete(c); setBusy(controllers.size > 0) }
  function fail(e: unknown, writing = false) {
    if (e instanceof DOMException && e.name === 'AbortError') return
    if (e instanceof APIError && e.status === 401) { onUnauthorized(); return }
    if (writing && (!(e instanceof APIError) || e.status >= 500 || e.status === 412)) setLocked(true)
    setError(e instanceof APIError ? t(`accounts.errors.${e.message}`, { defaultValue: t(`errors.${e.message}`, { defaultValue: t('requestFailed') }) }) : t('requestFailed'))
    if (!writing) setReadFailed(true)
  }
  async function read(signal: AbortSignal, rebase = false) {
    const generation = ++readEpoch.current
    const result = await request<Page>('/accounts', { signal })
    // A pre-write read may be held across navigation while the writer owns a
    // newer reconciliation read. It must not roll the visible directory back.
    if (signal.aborted || generation !== readEpoch.current) return
    if (result.data.mode !== 'database' || !Number.isSafeInteger(result.data.revision) || result.data.revision < 0 || result.etag !== `"${result.data.revision}"`) throw new Error('Invalid directory revision')
    setPage(result.data); setReadFailed(false)
    // A refresh updates observations, not the CAS ownership of a retained draft.
    if (base === null || rebase) setBase(result.data.revision)
    if (rebase) setLocked(false)
  }
  const start = useEffectEvent(async (c: AbortController) => {
    setBusy(true)
    try { await read(c.signal) } catch (e) { if (!c.signal.aborted) fail(e) } finally { finish(c) }
  })
  useEffect(() => {
    if (!active) return
    const c = new AbortController(); controllers.add(c)
    queueMicrotask(() => { if (!c.signal.aborted) void start(c) })
    const timer = window.setInterval(() => setNow(Date.now()), 15000)
    return () => { for (const c of controllers) if (!writers.has(c)) { c.abort(); controllers.delete(c) } setBusy(controllers.size > 0); window.clearInterval(timer) }
  }, [active, controllers, writers])
  useEffect(() => () => { for (const c of controllers) c.abort(); controllers.clear(); writers.clear() }, [controllers, writers])
  async function refresh(clear = false) {
    if (busy || (clear && dirty && !window.confirm(t('accounts.discard')))) return
    const c = new AbortController(); controllers.add(c); setBusy(true); setError(''); setFields({}); setNotice('')
    try { await read(c.signal, clear); if (clear && !c.signal.aborted) setForm(empty) }
    catch (e) { if (!c.signal.aborted) fail(e) } finally { finish(c) }
  }
  function edit(account: Account) {
    if (busy || locked || !page || (dirty && !window.confirm(t('accounts.discard')))) return
    setForm({ ...empty, ...account, password: '', reason: '', acknowledged: false }); setBase(page.revision); setFields({}); setError(''); setNotice('')
    queueMicrotask(() => document.getElementById('account-reason')?.focus())
  }
  async function save(e: React.FormEvent) {
    e.preventDefault(); if (busy || locked || !canWrite || base === null) return
    const invalid: Record<string, string> = {}
    const bytes = new TextEncoder()
    if (!form.id && !/^[a-z0-9][a-z0-9._-]{2,63}$/.test(form.username)) invalid['account-username'] = t('accounts.usernameHelp')
    if ((!form.id || form.password !== '') && (bytes.encode(form.password).length < 12 || bytes.encode(form.password).length > 72)) invalid['account-password'] = t('humanLogin.passwordLimit')
    if (!form.roles.length) invalid['account-roles'] = t('accounts.pickRole')
    if (!form.reason.trim() || bytes.encode(form.reason).length > 512) invalid['account-reason'] = t('accounts.reasonHelp')
    if (form.id && !form.acknowledged) invalid['account-ack'] = t('accounts.confirmHelp')
    if (Object.keys(invalid).length) { setFields(invalid); setError(t('requests.fixFields')); return }
    const c = new AbortController(); controllers.add(c); writers.add(c); setBusy(true); setError(''); setNotice(''); setFields({})
    const body = form.id ? { roles: form.roles, disabled: form.disabled, password: form.password, reason: form.reason } : { username: form.username, password: form.password, roles: form.roles, reason: form.reason }
    // Password is never retained for retry, including failed/ambiguous receipts.
    setForm(previous => ({ ...previous, password: '' }))
    let committed = false
    try {
      const result = await request<Account>(form.id ? `/accounts/${encodeURIComponent(form.id)}` : '/accounts', { method: form.id ? 'PUT' : 'POST', signal: c.signal, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf, 'If-Match': `"${base}"` }, body: JSON.stringify(body) }, form.id ? 200 : 201)
      if (c.signal.aborted) return
      if (result.etag !== `"${base + 1}"` || !/^principal:account-[a-f0-9]{32}$/.test(result.data.id) || (form.id && result.data.id !== form.id) || result.data.version !== form.version + 1) throw new Error('Invalid account receipt')
      committed = true; setForm(empty); setLocked(true); setNotice(t('accounts.saved')); await read(c.signal, true)
    } catch (e) { if (!c.signal.aborted) fail(e, !committed) } finally { finish(c) }
  }
  const field = (id: string) => ({ 'aria-invalid': !!fields[id], 'aria-describedby': fields[id] ? `${id}-error` : `${id}-help` })
  const fieldError = (id: string) => fields[id] && <p id={`${id}-error`} className="field-error">{fields[id]}</p>
  return <section className="accounts-section" aria-busy={busy}>
    <div className="section-heading"><div><h2>{t('accounts.directory')}</h2><p>{t('accounts.boundary')}</p></div><div className="actions"><Button variant="outline" disabled={busy} onClick={() => void refresh()}>{t('accounts.refresh')}</Button><Button variant="outline" disabled={busy} onClick={() => void refresh(true)}>{t('accounts.clear')}</Button></div></div>
    <p className="hint">{t('accounts.revision', { revision: base ?? '—' })} · {t('lastRead')}: {page ? date(page.observedAt) : t('notRead')}</p>
    {error && <div className="feedback error" role="alert" tabIndex={-1} ref={errors}><p>{error}</p>{Object.entries(fields).map(([id, text]) => <p key={id}><a href={`#${id}`} onClick={() => document.getElementById(id)?.focus()}>{text}</a></p>)}</div>}
    {notice && <p role="status" className="feedback success">{notice}</p>}
    {locked && <p className="stale" role="status">{t('accounts.locked')}</p>}
    {(readFailed || (page && now - Date.parse(page.observedAt) > 60000)) && <p className="stale" role="status">{t(readFailed ? 'staleFailed' : 'staleAge')}</p>}
    {busy && <p role="status">{t('loading')}</p>}
    {canWrite && <form className="grant-form" onSubmit={e => void save(e)} noValidate><h3>{t(form.id ? 'accounts.editing' : 'accounts.new', { username: form.username })}</h3><fieldset className="grant-fields-owner" disabled={busy}>
      <div className="grant-fields"><div><label htmlFor="account-username">{t('accounts.username')}</label><input id="account-username" {...field('account-username')} value={form.username} readOnly={!!form.id} onChange={e => setForm({ ...form, username: e.target.value })} autoComplete="off" autoCapitalize="none" spellCheck={false} maxLength={64} /><p id="account-username-help" className="hint">{t('accounts.usernameHelp')}</p>{fieldError('account-username')}</div>
      <div><label htmlFor="account-password">{t('accounts.password')}</label><input id="account-password" {...field('account-password')} type="password" value={form.password} onChange={e => setForm({ ...form, password: e.target.value })} autoComplete="new-password" maxLength={72} /><p id="account-password-help" className="hint">{t(form.id ? 'accounts.resetHelp' : 'accounts.passwordHelp')}</p>{fieldError('account-password')}</div></div>
      <fieldset id="account-roles" tabIndex={-1} {...field('account-roles')}><legend>{t('accounts.roles')}</legend><div className="endpoint-options">{roles.map(role => <label key={role}><input type="checkbox" aria-label={t(`accounts.roleLabels.${role.slice(5)}`)} checked={form.roles.includes(role)} onChange={e => setForm({ ...form, roles: e.target.checked ? [...form.roles, role] : form.roles.filter(r => r !== role) })} />{t(`accounts.roleLabels.${role.slice(5)}`)}</label>)}</div><p id="account-roles-help" className="hint">{t('accounts.roleHelp')}</p>{fieldError('account-roles')}</fieldset>
      {form.id && <label className="endpoint-option"><input type="checkbox" aria-label={t('accounts.disabled')} checked={form.disabled} onChange={e => setForm({ ...form, disabled: e.target.checked })} />{t('accounts.disabled')}</label>}
      <div><label htmlFor="account-reason">{t('accounts.reason')}</label><input id="account-reason" {...field('account-reason')} value={form.reason} onChange={e => setForm({ ...form, reason: e.target.value })} maxLength={512} /><p id="account-reason-help" className="hint">{t('accounts.reasonHelp')}</p>{fieldError('account-reason')}</div>
      {form.id && <div><label className="endpoint-option"><input id="account-ack" {...field('account-ack')} type="checkbox" aria-label={t('accounts.confirm')} checked={form.acknowledged} onChange={e => setForm({ ...form, acknowledged: e.target.checked })} />{t('accounts.confirmHelp')}</label><p id="account-ack-help" className="hint">{t('accounts.sessionHelp')}</p>{fieldError('account-ack')}</div>}
      <Button type="submit" disabled={busy || locked || base === null}>{t(form.id ? 'accounts.save' : 'accounts.create')}</Button>
    </fieldset></form>}
    <div className="audit-list">{page?.accounts.map(a => <article className="account-row" key={a.id}><div className="grant-row-heading"><h3>{a.username}</h3><strong>{t(a.disabled ? 'accounts.stateDisabled' : 'accounts.stateEnabled')}</strong></div><p>{a.roles.map(r => t(`accounts.roleLabels.${r.slice(5)}`)).join(', ')}</p><p>{t('accounts.version', { version: a.version })}</p><details><summary>{t('accounts.identity')}</summary><code>{a.id}</code></details>{canWrite && <Button variant="outline" disabled={busy || locked} onClick={() => edit(a)}>{t('accounts.edit')}</Button>}</article>)}</div>
    {page?.accounts.length === 0 && !busy && <p className="empty">{t('accounts.empty')}</p>}
    <h2>{t('accounts.audit')}</h2><p>{t('accounts.auditHelp')}</p><div className="audit-list">{page?.events.map(event => <article key={event.revision}><div className="audit-meta"><strong>{t('revisionNumber', { revision: event.revision })} · {t(`accounts.operations.${event.operation}`)}</strong><time dateTime={event.at}>{date(event.at)}</time></div><h3>{event.reason}</h3><p>{event.actor}</p><code>{event.accountId}</code></article>)}</div>
    {page?.events.length === 0 && !busy && <p className="empty">{t('accounts.emptyAudit')}</p>}
  </section>
}
