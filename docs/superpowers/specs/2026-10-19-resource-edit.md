# Unreferenced Resource network editing

Operators need to correct destinations, transport and ports before connecting a
Resource to authorization/enforcement. This extends existing inventory management
with an explicit narrow lifecycle decision. Routine design/implementation and
verified integration are preauthorized by the user; execute without checkpoints.

## Contract

PUT `/api/v1/resources/{uid}/network` and its session/CSRF console equivalent
accept `{endpoints:[{name,protocol,port,addresses?}],reason}`. Require strong
If-Match, publication:write, managed authority and current revision. Return200
`{uid,revision}` with matching ETag, never confirmed Agent application.
Keep Resource UID/name/metadata and the exact set of Endpoint names, type and
labels. Only network values change; input reordering must not reorder Endpoints.
No add/remove/rename, label editing, cascade or Binding retargeting in this slice.
Any Grant (including revoked/expired), NetworkDeny, EnforcementBinding or HTTPRoute
reference blocks409 resource_in_use, including zero-match dynamic selections.
Missing/wrong-kind UID404; stale CAS412 precedes lookup. Strict JSON rejects
unknown/duplicate/null/empty values using existing creation rules.

Recompile and publish the whole candidate through transactional publication+audit.
Preserve facts and unrelated source bytes/semantics. Structured editing normalizes
and moves the selected Resource document to the end; its comments/formatting are
not retained. Metadata and Endpoint labels are retained. Reject unsupported UTF-8
presentation. No schema migration or dependency change.

## UI

Reuse Resource editor and tokens: background#f5f7fa, foreground#18283b,
primary#145da0, muted#526174, warning#8a4900; system sans16px/1.5. Left-aligned
labeled fields, existing spacing/mobile stacking, no redesign. Edit network
parameters loads actual catalog/revision. Names read-only, UID visible, no
add/remove Endpoint actions. Explain reference refusal and normalization.
Save requires reason and acknowledgement. Background refresh never replaces
draft/base. Explicit rebase retains intent, resets acknowledgement and shows
latest target values; missing/same-name-recreated UID cannot substitute. Cancel
does not write. Busy/failed/stale/empty/viewer states remain explicit. Require
exact200, matching UID, increasing safe revision and matching ETag; ambiguous
writes lock retry until reconciliation. ui-ux-pro-max error-recovery guidance
requires actionable announced errors; frontend-design retains repository visuals.

## Acceptance and boundaries

Real API/SQLite tests prove permitted edit, retention, all reference classes,
CAS/auth/strict decoding and audit rollback. Real Chromium→React→Controller tests
prove cancel/save/rebase,409 refusal, malformed completion recovery, missing UID,
viewer and375px/1440px views. Existing real Nginx regressions remain mandatory.
Full native Go/race/vet/build and locked Web checks plus fresh independent review
precede expected-head merge. Local Unix EPERM is not a pass/skip. Referenced
Resource edits, Binding edits, identity/approval and combined Agent/browser Linux
acceptance remain; this makes no kernel/enforcement completion claim.

## 中文摘要

只允许修改无引用 Resource 的既有 Endpoint 协议、端口、目的地址；UID、名称、
Endpoint 集合及标签保持不变。任何授权（含撤销/过期）、拒绝、绑定或路由引用
均阻止编辑。复用鉴权、CSRF、版本 CAS、全量编译和事务审计；只规范化所选文档，
其注释/格式不保留，其他文档和 facts 保持。真实页面保留草稿，显式重基准后重新
确认；提交成功不代表 Agent 应用。引用编辑、绑定编辑和完整产品验收仍待完成。
