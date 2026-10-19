# Named host Binding Endpoint editing

The console can change the protected Endpoint names of an existing host Binding,
without recreating its identity or moving its execution scope. Only one pinned
Resource, explicit nonempty Endpoint names, one explicit Agent, host-local /
original-destination is supported. All/dynamic/multi-target bindings retain the
advanced publication workflow. No new stack, dependency, schema or kernel change.

PUT `/api/v1/bindings/:uid/endpoints` and its console equivalent accept a complete
nonempty `endpoint_names` list (1..64 unique existing names), audit `reason`, and
`acknowledge_scope_change:true`. Strong revision CAS precedes lookup. Unauthorized,
malformed, missing, unsupported and stale candidates return the existing explicit
401/403/400/404/409/412 contracts. Successful whole-candidate compile and atomic
publication/audit return200 InventoryChange/ETag, never an applied-state claim.

Keep Binding UID/name/metadata, pinned Resource, Agent, full enforcement context,
facts, Grants/Denies and unrelated document bytes. Normalize only the selected
Binding; canonicalize requested names in Resource Endpoint order. Inventory adds
optional `endpoint_edit` with resource_name/resource_uid/endpoint_names only for
supported shapes. Existing raw selector/scope summaries remain authoritative.

Reuse the existing console design. Display current and draft names plus fixed
scope. Adding protection may deny service traffic; removing protection may expose
traffic. Require explicit acknowledgement and reason. Rebase retains intent and
resets acknowledgement; block if Resource/Agent/enforcement signature changes,
shape becomes unsupported, target vanishes or selected names disappear. Exact200,
matching UID/revision/ETag are required; ambiguous outcomes retain the draft and
block retry until reconciliation. Viewers cannot edit. Test real SQLite/API/browser
and existing Nginx application acceptance; no mocked enforcement claims.

中文：仅编辑单 Resource、单 Agent、显式 Endpoint 名单的主机绑定；保留身份和
完整执行作用域。新增保护可能中断访问，移除可能撤下保护，必须确认并填写原因。
冲突保留草稿，作用域改变必须取消后重新打开；审批或发布不代表内核已经应用。
