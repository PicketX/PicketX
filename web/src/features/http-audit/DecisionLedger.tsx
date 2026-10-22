// SPDX-License-Identifier: Apache-2.0
import { memo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import type { HTTPAuditEvent } from './model'

export function ContributionList({ ids, label }: { ids: string[]; label: string }) {
  const { t } = useTranslation()
  const [offset, setOffset] = useState(0)
  return <div className="http-contributions"><h4>{label}</h4>{ids.length === 0 ? <p>{t('httpAudit.none')}</p> : <>
    <p className="hint">{t('httpAudit.idWindow', { first: offset + 1, last: Math.min(offset + 50, ids.length), total: ids.length })}</p>
    <ul>{ids.slice(offset, offset + 50).map(id => <li key={id}>{id}</li>)}</ul>
    {ids.length > 50 && <div className="http-audit-actions"><Button variant="outline" disabled={offset === 0} onClick={() => setOffset(n => Math.max(0, n - 50))}>{t('httpAudit.previousIDs')}</Button>
      <Button variant="outline" disabled={offset + 50 >= ids.length} onClick={() => setOffset(n => n + 50)}>{t('httpAudit.nextIDs')}</Button></div>}
  </>}</div>
}

// Only records/locale/disclosure changes reconcile the ledger, never clock ticks.
export const DecisionLedger = memo(function DecisionLedger({ rows }: { rows: HTTPAuditEvent[] }) {
  const { t, i18n } = useTranslation()
  const [selected, setSelected] = useState<number | null>(null)
  return <div className="http-decision-list">{rows.map(event => <article key={event.id} data-http-decision-id={event.id}>
    <div className="http-decision-heading"><h3>{t(event.allowed ? 'httpAudit.allowed' : 'httpAudit.denied')}</h3><time dateTime={event.at}>{new Date(event.at).toLocaleString(i18n.language)}</time></div>
    <dl><div><dt>{t('httpAudit.source')}</dt><dd>{event.source}</dd></div><div><dt>{t('httpAudit.reason')}</dt><dd>{event.reason}</dd></div>
      <div><dt>{t('httpAudit.route')}</dt><dd>{event.route}</dd></div><div><dt>{t('httpAudit.caller')}</dt><dd>{event.caller}</dd></div>
      <div><dt>{t('httpAudit.event')}</dt><dd>{event.id}</dd></div><div><dt>{t('httpAudit.revision')}</dt><dd>{event.revision}</dd></div></dl>
    <details open={selected === event.id} onToggle={e => { const open = e.currentTarget.open; setSelected(previous => open ? event.id : previous === event.id ? null : previous) }}>
      <summary>{t('httpAudit.details')}</summary>{selected === event.id && <>
        <ContributionList key={`grants-${event.id}`} ids={event.grant_ids} label={t('httpAudit.grants')} />
        <ContributionList key={`denies-${event.id}`} ids={event.deny_ids} label={t('httpAudit.denies')} />
      </>}
    </details>
  </article>)}</div>
})
