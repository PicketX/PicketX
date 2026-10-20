// SPDX-License-Identifier: Apache-2.0
import { useEffect, useEffectEvent, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { APIError, request } from '@/lib/console'
import { observationAge } from '@/features/nodes/freshness'
import { appendHTTPAudit, parseHTTPAuditPage } from './model'
import type { HTTPAuditEvent } from './model'

type Reading = { rows: HTTPAuditEvent[]; cursor: number; emptyTail: boolean; wall: number; monotonic: number; received: number }
export function HTTPAudit({ onUnauthorized }: { onUnauthorized: () => void }) {
  const { t, i18n } = useTranslation()
  const [reading, setReading] = useState<Reading | null>(null)
  const [busy, setBusy] = useState(true)
  const [failed, setFailed] = useState(false)
  const [forbidden, setForbidden] = useState(false)
  const [clock, setClock] = useState(() => ({ wall: Date.now(), monotonic: performance.now() }))
  const actions = useRef({ next: () => {}, reset: () => {} })
  const unauthorized = useEffectEvent(onUnauthorized)
  useEffect(() => {
    let disposed = false, initialPending = true, current: Reading | null = null
    let active: AbortController | null = null
    async function load(reset: boolean) {
      if (disposed || active || document.hidden) return
      initialPending = false
      const controller = new AbortController(); active = controller
      const after = reset ? 0 : current?.cursor ?? 0
      const started = { wall: Date.now(), monotonic: performance.now() }
      setBusy(true)
      try {
        const { data } = await request<unknown>(`/http-decisions?after=${after}&limit=50`, { signal: controller.signal }, 200)
        if (disposed || controller.signal.aborted) return
        const page = parseHTTPAuditPage(data, after)
        current = { rows: appendHTTPAudit(reset ? [] : current?.rows ?? [], page), cursor: page.next_after,
          emptyTail: page.events.length === 0, ...started, received: Date.now() }
        setReading(current); setFailed(false); setForbidden(false)
        setClock({ wall: Date.now(), monotonic: performance.now() })
      } catch (error) {
        if (disposed || controller.signal.aborted) return
        if (error instanceof APIError && [401, 403].includes(error.status)) {
          current = null; setReading(null); setFailed(false)
          if (error.status === 401) unauthorized()
          else setForbidden(true)
        } else setFailed(true)
      } finally {
        active = null
        if (!disposed) setBusy(false)
      }
    }
    actions.current = { next: () => { void load(false) }, reset: () => { void load(true) } }
    queueMicrotask(() => { void load(true) })
    const tick = window.setInterval(() => setClock({ wall: Date.now(), monotonic: performance.now() }), 1000)
    const visible = () => {
      if (!document.hidden) {
        setClock({ wall: Date.now(), monotonic: performance.now() })
        if (initialPending) void load(true)
      }
    }
    document.addEventListener('visibilitychange', visible)
    return () => {
      disposed = true; active?.abort(); actions.current = { next: () => {}, reset: () => {} }
      window.clearInterval(tick); document.removeEventListener('visibilitychange', visible)
    }
  }, [])
  const outdated = reading !== null && observationAge(reading, clock) >= 60000
  return <section className="http-audit-section" aria-busy={busy} data-after={reading?.cursor ?? 0}>
    <div className="section-heading"><div><h2>{t('httpAudit.title')}</h2><p>{t('httpAudit.help')}</p></div>
      <div className="http-audit-actions"><Button disabled={busy} onClick={() => actions.current.next()}>{t('httpAudit.next')}</Button>
        <Button variant="outline" disabled={busy} onClick={() => actions.current.reset()}>{t('httpAudit.reset')}</Button></div></div>
    <p className="boundary">{t('httpAudit.boundary')}</p>
    {busy && <p role="status">{t('httpAudit.loading')}</p>}
    {forbidden && <p className="feedback error" role="alert">{t('httpAudit.forbidden')}</p>}
    {failed && <p className="feedback error" role="alert">{t('httpAudit.failed')}</p>}
    {!failed && outdated && <p className="stale" role="status">{t('httpAudit.outdated')}</p>}
    {reading && <>
      <p className="hint">{t('httpAudit.lastRead', { at: new Date(reading.received).toLocaleString(i18n.language) })}</p>
      <p className="hint">{t('httpAudit.window', { count: reading.rows.length, cursor: reading.cursor })}</p>
      {reading.rows.length === 0 ? <p className="empty">{t('httpAudit.empty')}</p> : <>
        {reading.emptyTail && <p role="status">{t('httpAudit.emptyTail')}</p>}
        <div className="http-decision-list">{reading.rows.map(event => <article key={event.id} data-http-decision-id={event.id}>
          <div className="http-decision-heading"><h3>{t(event.allowed ? 'httpAudit.allowed' : 'httpAudit.denied')}</h3><time dateTime={event.at}>{new Date(event.at).toLocaleString(i18n.language)}</time></div>
          <dl><div><dt>{t('httpAudit.source')}</dt><dd>{event.source}</dd></div><div><dt>{t('httpAudit.reason')}</dt><dd>{event.reason}</dd></div>
            <div><dt>{t('httpAudit.route')}</dt><dd>{event.route}</dd></div><div><dt>{t('httpAudit.caller')}</dt><dd>{event.caller}</dd></div>
            <div><dt>{t('httpAudit.event')}</dt><dd>{event.id}</dd></div><div><dt>{t('httpAudit.revision')}</dt><dd>{event.revision}</dd></div></dl>
          <details><summary>{t('httpAudit.details')}</summary><h4>{t('httpAudit.grants')}</h4>{event.grant_ids.length ? <ul>{event.grant_ids.map(id => <li key={id}>{id}</li>)}</ul> : <p>{t('httpAudit.none')}</p>}
            <h4>{t('httpAudit.denies')}</h4>{event.deny_ids.length ? <ul>{event.deny_ids.map(id => <li key={id}>{id}</li>)}</ul> : <p>{t('httpAudit.none')}</p>}</details>
        </article>)}</div>
      </>}
    </>}
  </section>
}
