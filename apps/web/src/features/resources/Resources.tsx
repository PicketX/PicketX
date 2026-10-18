// SPDX-License-Identifier: Apache-2.0
import { useEffect, useEffectEvent, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { APIError, request } from '@/lib/console'

type Endpoint = { name: string; protocol: string; port: number; addresses?: string[] }
type Resource = { name: string; uid: string; endpoints: Endpoint[] }
type Binding = { name: string; uid: string; resources: string; agents: string; enforcement: string }
type Context = { agent_name: string; id: string; profile: string; class: string; stage: string; destinations: string[]; handling: string; selectable: boolean }
type Inventory = { authority: string; revision: number; observed_at: string; resources: Resource[]; bindings: Binding[]; contexts: Context[] }
type EndpointDraft = { name: string; protocol: string; port: string; addresses: string }
type Props = { csrf: string; canWrite: boolean; onUnauthorized: () => void; onDirty: (dirty: boolean) => void; onBusy: (busy: boolean) => void }
const blankEndpoint = (): EndpointDraft => ({ name: '', protocol: 'TCP', port: '', addresses: '' })
const contextKey = (c: Context) => JSON.stringify([c.agent_name, c.id])
const addressList = (value: string) => value.trim() ? value.split(/[\n,]/).map(s => s.trim()).filter(Boolean) : undefined

export function Resources({ csrf, canWrite, onUnauthorized, onDirty, onBusy }: Props) {
  const { t, i18n } = useTranslation()
  const [inventory, setInventory] = useState<Inventory | null>(null)
  const [catalog, setCatalog] = useState<Inventory | null>(null)
  const [base, setBase] = useState('')
  const [mode, setMode] = useState<'resource' | 'binding' | 'remove'>('resource')
  const [bindingUID, setBindingUID] = useState('')
  const [acknowledged, setAcknowledged] = useState(false)
  const [name, setName] = useState('')
  const [reason, setReason] = useState('')
  const [endpoints, setEndpoints] = useState<EndpointDraft[]>([blankEndpoint()])
  const [resourceUID, setResourceUID] = useState('')
  const [selectedEndpoints, setSelectedEndpoints] = useState<string[]>([])
  const [contextID, setContextID] = useState('')
  const [page, setPage] = useState(0)
  const [busy, setBusy] = useState(true)
  const [error, setError] = useState('')
  const [fields, setFields] = useState<Record<string, string>>({})
  const [notice, setNotice] = useState('')
  const [conflict, setConflict] = useState(false)
  const [readFailed, setReadFailed] = useState(false)
  const [now, setNow] = useState(() => Date.now())
  const [controllers] = useState(() => new Set<AbortController>())
  const errors = useRef<HTMLDivElement>(null)
  const removalHeading = useRef<HTMLHeadingElement>(null)
  const dirty = !!(bindingUID || name || reason || resourceUID || contextID || selectedEndpoints.length || endpoints.length !== 1 || endpoints.some(e => e.name || e.port || e.addresses || e.protocol !== 'TCP'))
  const binding = catalog?.bindings.find(b => b.uid === bindingUID)
  const resource = catalog?.resources.find(r => r.uid === resourceUID)
  const context = catalog?.contexts.find(c => contextKey(c) === contextID && c.selectable)
  useEffect(() => { onDirty(dirty) }, [dirty, onDirty])
  useEffect(() => { onBusy(busy) }, [busy, onBusy])
  useEffect(() => { if (error) errors.current?.focus() }, [error, fields])
  useEffect(() => { if (mode === 'remove') removalHeading.current?.focus() }, [mode, bindingUID])

  function fail(e: unknown, reading = false) {
    if (e instanceof DOMException && e.name === 'AbortError') return
    if (e instanceof APIError && e.status === 401) { onUnauthorized(); return }
    if (e instanceof APIError && e.status === 412) { setConflict(true); setError(t('conflict')); return }
    setError(e instanceof APIError ? t(`errors.${e.message}`, { defaultValue: t('requestFailed') }) : t('requestFailed'))
    if (reading) setReadFailed(true)
  }
  async function read(signal: AbortSignal, rebase = false) {
    const result = await request<Inventory>('/inventory', { signal })
    if (signal.aborted) return
    if (!/^"\d+"$/.test(result.etag)) throw new Error('Missing strong revision')
    setInventory(result.data); setReadFailed(false); setPage(0)
    // An observation must not change the mutation base or choices behind a draft.
    if (rebase || !base) { setBase(result.etag); setCatalog(result.data); setConflict(false); setAcknowledged(false) }
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
    if (busy || (rebase && dirty && !window.confirm(t('resources.rebaseConfirm')))) return
    const controller = new AbortController(); controllers.add(controller)
    setBusy(true); setError(''); setFields({}); setNotice('')
    try { await read(controller.signal, rebase) } catch (e) { if (!controller.signal.aborted) fail(e, true) }
    finally { controllers.delete(controller); if (!controller.signal.aborted) setBusy(false) }
  }
  function clear() {
    setBindingUID(''); setAcknowledged(false)
    setName(''); setReason(''); setEndpoints([blankEndpoint()]); setResourceUID(''); setSelectedEndpoints([]); setContextID(''); setFields({})
  }
  function selectRemoval(uid: string) {
    if (busy || !canWrite || (dirty && !window.confirm(t('resources.discard')))) return
    clear(); setBindingUID(uid); setMode('remove'); setError(''); setNotice('')
  }
  function switchMode(value: typeof mode) {
    if (value === mode || busy || (dirty && !window.confirm(t('resources.discard')))) return
    clear(); setMode(value); setError(''); setNotice('')
  }
  function editEndpoint(index: number, field: keyof EndpointDraft, value: string) {
    setEndpoints(old => old.map((e, n) => n === index ? { ...e, [field]: value } : e))
  }
  async function submit(e: React.FormEvent) {
    e.preventDefault(); if (busy || !canWrite || !base || conflict) return
    const invalid: Record<string, string> = {}
    if (mode !== 'remove' && !name.trim()) invalid[`${mode}-name`] = t('grants.required')
    if (!reason.trim()) invalid[`${mode}-reason`] = t('grants.required')
    if (mode === 'resource') {
      for (const [index, endpoint] of endpoints.entries()) {
        if (!endpoint.name.trim() || endpoints.some((other, n) => n !== index && other.name === endpoint.name)) invalid[`endpoint-${index}-name`] = t('resources.uniqueEndpoint')
        if (!/^\d+$/.test(endpoint.port) || Number(endpoint.port) < 1 || Number(endpoint.port) > 65535) invalid[`endpoint-${index}-port`] = t('resources.portRange')
        if ((addressList(endpoint.addresses)?.length ?? 0) > 64) invalid[`endpoint-${index}-addresses`] = t('resources.addressLimit')
      }
    } else if (mode === 'binding') {
      if (!resource) invalid['binding-resource'] = t('grants.selectResource')
      if (!selectedEndpoints.length || selectedEndpoints.length > 64 || selectedEndpoints.some(n => !resource?.endpoints.some(e => e.name === n))) invalid['binding-endpoints'] = t('resources.selectEndpoints')
      if (!context) invalid['binding-context'] = t('resources.selectContext')
    } else {
      if (!binding || !acknowledged) return
    }
    if (Object.keys(invalid).length) { setFields(invalid); setError(t('grants.fixFields')); return }
    const body = mode === 'remove'
      ? { reason, acknowledge_unprotection: acknowledged }
      : mode === 'resource'
      ? { name, reason, endpoints: endpoints.map(e => ({ name: e.name, protocol: e.protocol, port: Number(e.port), addresses: addressList(e.addresses) })) }
      : { name, reason, resource_name: resource!.name, resource_uid: resource!.uid, endpoint_names: selectedEndpoints, agent_name: context!.agent_name, context_id: context!.id }
    const controller = new AbortController(); controllers.add(controller); setBusy(true); setError(''); setFields({}); setNotice('')
    try {
      await request(mode === 'remove' ? `/bindings/${encodeURIComponent(bindingUID)}/remove` : mode === 'resource' ? '/resources' : '/bindings', { signal: controller.signal, method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf, 'If-Match': base }, body: JSON.stringify(body) })
      if (controller.signal.aborted) return
      clear(); setNotice(t(mode === 'remove' ? 'resources.removed' : mode === 'resource' ? 'resources.created' : 'resources.bound')); setConflict(true)
      if (mode === 'remove') setMode('resource')
      // The successful POST is not repeated if its follow-up read fails.
      try { await read(controller.signal, true) } catch (e) { if (!controller.signal.aborted) fail(e, true) }
    } catch (e) {
      if (!controller.signal.aborted) {
        if (!(e instanceof APIError) || e.status >= 500 || (mode === 'remove' && e.status === 404)) setConflict(true)
        fail(e)
      }
    } finally { controllers.delete(controller); if (!controller.signal.aborted) setBusy(false) }
  }
  const fieldProps = (id: string) => ({ 'aria-invalid': !!fields[id], 'aria-describedby': fields[id] ? `${id}-error` : undefined })
  const fieldError = (id: string) => fields[id] && <p id={`${id}-error`} className="field-error">{fields[id]}</p>
  const fieldLabel = (id: string) => {
    const endpoint = /^endpoint-(\d+)-(name|port|addresses)$/.exec(id)
    if (endpoint) return t(`resources.endpoint${endpoint[2][0].toUpperCase()}${endpoint[2].slice(1)}`, { n: Number(endpoint[1]) + 1 })
    const keys: Record<string, string> = { 'remove-reason': 'removeReason', 'resource-name': 'resourceName', 'resource-reason': 'resourceReason', 'binding-name': 'bindingName', 'binding-reason': 'bindingReason', 'binding-resource': 'bindingResource', 'binding-endpoints': 'selectEndpoints', 'binding-context': 'context' }
    return t(`resources.${keys[id]}`)
  }
  const shown = <T,>(values: T[] | undefined) => values?.slice(page * 50, (page + 1) * 50)
  const pages = Math.ceil(Math.max(inventory?.resources.length ?? 0, inventory?.bindings.length ?? 0, inventory?.contexts.length ?? 0) / 50)
  return <section className="resources-section" aria-busy={busy}>
    <div className="section-heading"><div><h2>{t('resources.onboarding')}</h2><p>{t('resources.boundary')}</p></div><div className="actions"><Button variant="outline" disabled={busy} onClick={() => void refresh()}>{t('resources.refresh')}</Button><Button variant="outline" disabled={busy} onClick={() => void refresh(true)}>{t('resources.rebase')}</Button></div></div>
    <p className="hint">{t('resources.base', { base: base.slice(1, -1) || '—', observed: inventory?.revision ?? '—' })} · {t('lastRead')}: {inventory ? new Date(inventory.observed_at).toLocaleString(i18n.language) : t('notRead')}</p>
    {error && <div className="feedback error" role="alert" tabIndex={-1} ref={errors}><p>{error}</p><ul>{Object.entries(fields).map(([id, text]) => <li key={id}><a href={`#${id}`} onClick={() => document.getElementById(id)?.focus()}>{fieldLabel(id)}: {text}</a></li>)}</ul></div>}
    {notice && <p className="feedback success" role="status">{notice}</p>}
    {readFailed || conflict || (inventory && now - Date.parse(inventory.observed_at) > 60000) ? <p className="stale" role="status">{conflict ? t('resources.conflictHelp') : readFailed ? t('staleFailed') : t('staleAge')}</p> : null}
    {busy && <p role="status">{t('loading')}</p>}
    {!canWrite ? <p className="boundary">{t('readOnly')}</p> : <>
      <div className="actions resource-modes"><Button variant={mode === 'resource' ? 'secondary' : 'outline'} aria-pressed={mode === 'resource'} disabled={busy} onClick={() => switchMode('resource')}>{t('resources.newResource')}</Button><Button variant={mode === 'binding' ? 'secondary' : 'outline'} aria-pressed={mode === 'binding'} disabled={busy} onClick={() => switchMode('binding')}>{t('resources.newBinding')}</Button></div>
      <form className="grant-form resource-form" onSubmit={e => void submit(e)} noValidate><fieldset className="grant-fields-owner" disabled={busy}>
        {mode === 'remove' ? <>
          <h3 ref={removalHeading} tabIndex={-1}>{t('resources.remove')}</h3>
          <p className="boundary">{t('resources.removeWarning')}</p>
          {binding ? <div className="context-preview"><h4>{binding.name}</h4><p>{binding.uid}</p><h4>{t('resources.list')}</h4><code>{binding.resources}</code><h4>{t('resources.agents')}</h4><code>{binding.agents}</code><h4>{t('resources.execution')}</h4><code>{binding.enforcement}</code></div> : <p className="stale">{t('resources.missingBinding')}</p>}
          <div><label htmlFor="remove-reason">{t('resources.removeReason')}</label><input id="remove-reason" {...fieldProps('remove-reason')} value={reason} onChange={e => setReason(e.target.value)} maxLength={512} required />{fieldError('remove-reason')}</div>
          <div className="endpoint-options"><label><input type="checkbox" checked={acknowledged} onChange={e => setAcknowledged(e.target.checked)} />{t('resources.acknowledge')}</label></div>
          <div className="actions"><Button variant="destructive" type="submit" disabled={busy || conflict || !base || !binding || !acknowledged}>{t('resources.confirmRemoval')}</Button><Button variant="outline" type="button" onClick={() => { clear(); setMode('resource'); setError(''); setNotice('') }}>{t('resources.cancelRemoval')}</Button></div>
        </> : <>
        <h3>{t(mode === 'resource' ? 'resources.create' : 'resources.bind')}</h3>
        <div><label htmlFor={`${mode}-name`}>{t(`resources.${mode}Name`)}</label><input id={`${mode}-name`} {...fieldProps(`${mode}-name`)} value={name} onChange={e => setName(e.target.value)} maxLength={128} required />{fieldError(`${mode}-name`)}</div>
        {mode === 'resource' ? <>
          <p className="hint">{t('resources.endpointHelp')}</p>
          {endpoints.map((endpoint, index) => <fieldset key={index} className="endpoint-editor"><legend>{t('resources.endpoint', { n: index + 1 })}</legend><div className="grant-fields">
            <div><label htmlFor={`endpoint-${index}-name`}>{t('resources.endpointName', { n: index + 1 })}</label><input id={`endpoint-${index}-name`} {...fieldProps(`endpoint-${index}-name`)} value={endpoint.name} onChange={e => editEndpoint(index, 'name', e.target.value)} maxLength={128} required />{fieldError(`endpoint-${index}-name`)}</div>
            <div><label htmlFor={`endpoint-${index}-protocol`}>{t('resources.endpointProtocol', { n: index + 1 })}</label><select id={`endpoint-${index}-protocol`} value={endpoint.protocol} onChange={e => editEndpoint(index, 'protocol', e.target.value)}><option>TCP</option><option>UDP</option></select></div>
            <div><label htmlFor={`endpoint-${index}-port`}>{t('resources.endpointPort', { n: index + 1 })}</label><input id={`endpoint-${index}-port`} {...fieldProps(`endpoint-${index}-port`)} type="number" min={1} max={65535} step={1} value={endpoint.port} onChange={e => editEndpoint(index, 'port', e.target.value)} required />{fieldError(`endpoint-${index}-port`)}</div>
            <div><label htmlFor={`endpoint-${index}-addresses`}>{t('resources.endpointAddresses', { n: index + 1 })}</label><textarea id={`endpoint-${index}-addresses`} {...fieldProps(`endpoint-${index}-addresses`)} value={endpoint.addresses} onChange={e => editEndpoint(index, 'addresses', e.target.value)} maxLength={8256} rows={3} spellCheck={false} />{fieldError(`endpoint-${index}-addresses`)}</div>
          </div><Button type="button" variant="outline" disabled={endpoints.length === 1} onClick={() => { setEndpoints(old => old.filter((_, n) => n !== index)); setFields({}) }}>{t('resources.removeEndpoint', { n: index + 1 })}</Button></fieldset>)}
          <Button type="button" variant="outline" disabled={endpoints.length >= 64} onClick={() => setEndpoints(old => [...old, blankEndpoint()])}>{t('resources.addEndpoint')}</Button>
        </> : <>
          <div><label htmlFor="binding-resource">{t('resources.bindingResource')}</label><select id="binding-resource" {...fieldProps('binding-resource')} value={resourceUID} onChange={e => { setResourceUID(e.target.value); setSelectedEndpoints([]) }} required><option value="">{t('grants.selectResource')}</option>{catalog?.resources.map(r => <option key={r.uid} value={r.uid}>{r.name} · {r.uid}</option>)}</select>{fieldError('binding-resource')}</div>
          <fieldset id="binding-endpoints" tabIndex={-1} {...fieldProps('binding-endpoints')}><legend>{t('grants.endpoints')}</legend><div className="endpoint-options">{resource?.endpoints.map(e => <label key={e.name}><input type="checkbox" checked={selectedEndpoints.includes(e.name)} onChange={event => setSelectedEndpoints(old => event.target.checked ? [...old, e.name] : old.filter(n => n !== e.name))} />{e.name} <span>{e.protocol}/{e.port}</span></label>)}</div>{fieldError('binding-endpoints')}</fieldset>
          <div><label htmlFor="binding-context">{t('resources.context')}</label><select id="binding-context" {...fieldProps('binding-context')} value={contextID} onChange={e => setContextID(e.target.value)} required><option value="">{t('resources.selectContext')}</option>{catalog?.contexts.filter(c => c.selectable).map(c => <option key={contextKey(c)} value={contextKey(c)}>{c.agent_name} · {c.id} · {c.profile}</option>)}</select>{fieldError('binding-context')}{catalog && !catalog.contexts.some(c => c.selectable) && <p className="hint">{t('resources.noContexts')}</p>}</div>
          {context && <div className="context-preview"><h4>{t('resources.reviewContext')}</h4><p>{context.agent_name} · {context.id} · {context.profile}</p><p>{context.class} · {context.stage} · {context.handling}</p><p><code>{context.destinations.join(', ')}</code></p><p className="hint">{t('resources.contextBoundary')}</p></div>}
        </>}
        <div><label htmlFor={`${mode}-reason`}>{t(`resources.${mode}Reason`)}</label><input id={`${mode}-reason`} {...fieldProps(`${mode}-reason`)} value={reason} onChange={e => setReason(e.target.value)} maxLength={512} required />{fieldError(`${mode}-reason`)}</div>
        <div className="actions"><Button type="submit" disabled={busy || conflict || !base}>{t(mode === 'resource' ? 'resources.create' : 'resources.bind')}</Button><Button type="button" variant="outline" disabled={!dirty} onClick={() => { if (window.confirm(t('resources.discard'))) clear() }}>{t('resources.clear')}</Button></div>
        </>}
      </fieldset></form>
    </>}
    <div className="inventory-list"><h2>{t('resources.list')}</h2>{inventory?.resources.length === 0 && !busy && <p className="empty">{t('resources.empty')}</p>}{shown(inventory?.resources)?.map(r => <article key={r.uid} data-resource-id={r.uid}><h3>{r.name}</h3><p className="hint">{r.uid}</p><ul>{r.endpoints.map(e => <li key={e.name}><strong>{e.name}</strong> · {e.protocol}/{e.port} · <code>{e.addresses?.join(', ') ?? t('resources.bindingScope')}</code></li>)}</ul></article>)}</div>
    <div className="inventory-list"><h2>{t('resources.bindings')}</h2><p className="hint">{t('resources.selectionHelp')}</p>{inventory?.bindings.length === 0 && !busy && <p className="empty">{t('resources.emptyBindings')}</p>}{shown(inventory?.bindings)?.map(b => <article key={b.uid} data-binding-id={b.uid}><h3>{b.name}</h3><p className="hint">{b.uid}</p><details><summary>{t('resources.inspectBinding')}</summary><h4>{t('resources.list')}</h4><code>{b.resources}</code><h4>{t('resources.agents')}</h4><code>{b.agents}</code><h4>{t('resources.execution')}</h4><code>{b.enforcement}</code></details>{canWrite && <Button variant="outline" disabled={busy || !catalog?.bindings.some(item => item.uid === b.uid)} onClick={() => selectRemoval(b.uid)}>{t('resources.remove')}</Button>}</article>)}</div>
    <div className="inventory-list"><h2>{t('resources.contexts')}</h2><p className="hint">{t('resources.contextBoundary')}</p>{inventory?.contexts.length === 0 && !busy && <p className="empty">{t('resources.noContexts')}</p>}{shown(inventory?.contexts)?.map(c => <article key={contextKey(c)}><h3>{c.agent_name} · {c.id}</h3><p>{c.profile} · {c.class} · {c.stage} · {c.handling}</p><p><code>{c.destinations.join(', ')}</code></p>{!c.selectable && <p className="hint">{t('resources.unsupportedContext')}</p>}</article>)}</div>
    {pages > 1 && <div className="actions"><Button variant="outline" disabled={page === 0 || busy} onClick={() => setPage(n => n - 1)}>{t('resources.previous')}</Button><span>{t('resources.page', { n: page + 1, pages })}</span><Button variant="outline" disabled={page >= pages - 1 || busy} onClick={() => setPage(n => n + 1)}>{t('resources.next')}</Button></div>}
  </section>
}
