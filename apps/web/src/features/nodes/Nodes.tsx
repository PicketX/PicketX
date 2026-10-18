// SPDX-License-Identifier: Apache-2.0
import { useEffect, useEffectEvent, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { APIError, request } from '@/lib/console'
import { observationAge, reportIsFresh } from './freshness'

type NodeObservation = {
  agent_ref: string; desired_revision: string; desired_sha256: string
  connection: 'connected' | 'disconnected'; freshness: 'unknown' | 'stale' | 'fresh'
  fresh_for_millis: number; matches_desired: boolean
  report: null | { revision: string; sha256: string; phase: string; observed_at: string; received_at: string }
}
type Observations = { authority: string; observed_at: string; nodes: NodeObservation[] }
type Reading = { data: Observations; wall: number; monotonic: number; received: number }

export function Nodes({ onUnauthorized }: { onUnauthorized: () => void }) {
  const { t, i18n } = useTranslation()
  const [reading, setReading] = useState<Reading | null>(null)
  const [busy, setBusy] = useState(true)
  const [failed, setFailed] = useState(false)
  const [clock, setClock] = useState(() => ({ wall: Date.now(), monotonic: performance.now() }))
  const refresh = useRef<() => void>(() => {})
  const unauthorized = useEffectEvent(onUnauthorized)
  useEffect(() => {
    let disposed = false
    let active: AbortController | null = null
    async function load() {
      if (disposed || active || document.hidden) return
      const controller = new AbortController(); active = controller
      const wall = Date.now(), monotonic = performance.now()
      setBusy(true)
      try {
        const { data } = await request<Observations>('/agent-observations', { signal: controller.signal })
        if (disposed) return
        setReading({ data, wall, monotonic, received: Date.now() }); setFailed(false)
        setClock({ wall: Date.now(), monotonic: performance.now() })
      } catch (error) {
        if (disposed) return
        if (error instanceof APIError && error.status === 401) unauthorized()
        else setFailed(true)
      } finally { active = null; if (!disposed) setBusy(false) }
    }
    refresh.current = () => { void load() }
    queueMicrotask(() => { void load() })
    const poll = window.setInterval(() => { void load() }, 15000)
    const tick = window.setInterval(() => setClock({ wall: Date.now(), monotonic: performance.now() }), 1000)
    const visible = () => { if (!document.hidden) void load() }
    document.addEventListener('visibilitychange', visible)
    return () => {
      disposed = true; active?.abort(); refresh.current = () => {}
      window.clearInterval(poll); window.clearInterval(tick)
      document.removeEventListener('visibilitychange', visible)
    }
  }, [])
  // Include request latency, use monotonic elapsed time, and conservatively
  // include wall-clock elapsed time across browser/device suspension.
  const elapsed = reading ? observationAge(reading, clock) : 0
  const historical = failed || elapsed >= 30000
  const date = (value: string) => new Date(value).toLocaleString(i18n.language)
  return <section className="nodes-section" aria-busy={busy}>
    <div className="section-heading"><div><h2>{t('nodes.observations')}</h2><p>{t('nodes.help')}</p></div>
      <Button variant="outline" disabled={busy} onClick={() => refresh.current()}><RefreshCw aria-hidden="true" />{t('nodes.refresh')}</Button></div>
    <p className="boundary">{t('nodes.boundary')}</p>
    {busy && <p role="status">{t('loading')}</p>}
    {failed && <p className="feedback error" role="alert">{t(reading ? 'nodes.failedRetained' : 'nodes.failed')}</p>}
    {!failed && historical && <p className="stale" role="status">{t('nodes.historical')}</p>}
    {reading && <>
      <p className="hint">{t('nodes.lastSuccess', { at: new Date(reading.received).toLocaleTimeString(i18n.language) })} · {t('nodes.polling')}</p>
      {reading.data.nodes.length === 0 && <p className="empty">{t('nodes.empty')}</p>}
      <div className="node-list">{reading.data.nodes.map(node => {
        const fresh = reportIsFresh(node, elapsed, historical)
        return <article key={node.agent_ref}>
          <h3>{node.agent_ref}</h3>
          <p>{t(historical ? 'nodes.connectionHistorical' : 'nodes.connection')}: <strong>{t(`nodes.${node.connection}`)}</strong></p>
          <div className="node-columns"><section><h4>{t('nodes.desired')}</h4><p>{t('revisionNumber', { revision: node.desired_revision })}</p><details><summary>{t('digest')}</summary><code>{node.desired_sha256}</code></details></section>
            <section><h4>{t('nodes.report')}</h4>{node.report ? <>
              <p><strong>{t(`nodes.phase.${node.report.phase}`, { defaultValue: node.report.phase })}</strong></p>
              <p className={fresh ? '' : 'stale'}>{t(fresh ? 'nodes.fresh' : 'nodes.stale')}</p>
              <p>{t(node.matches_desired ? 'nodes.matches' : 'nodes.mismatch')}</p>
              <p>{t('revisionNumber', { revision: node.report.revision })}</p>
              <dl><dt>{t('nodes.received')}</dt><dd>{date(node.report.received_at)}</dd><dt>{t('nodes.agentTime')}</dt><dd>{node.report.observed_at || '—'}</dd></dl>
              <details><summary>{t('digest')}</summary><code>{node.report.sha256}</code></details>
            </> : <p>{t('nodes.unreported')}</p>}</section></div>
        </article>
      })}</div>
    </>}
  </section>
}
