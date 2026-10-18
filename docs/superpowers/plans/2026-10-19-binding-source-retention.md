# Binding source retention implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans. The user authorized native autonomous execution and independent whole-branch review.

**Goal:** Preserve inter-document comments when the management API removes a Binding.

**Architecture:** Keep the current byte-preserving editor and full semantic comparison. Bound deletion at an explicit top-level YAML document end marker when present, retaining the following comments/directives. No reserialization, API, UI, schema, or authorization change.

**Tech Stack:** Existing Go and go.yaml.in/yaml/v3; SQLite management transaction.

**Spec:** Bounded design in this document; PR23's deferred source-presentation defect, not new lifecycle semantics.

## Global constraints

- Only UID-selected Binding is removed; surviving semantic objects and facts remain unchanged.
- Detect only column-zero `...` followed by whitespace, a YAML line break or EOF; do not mistake quoted or indented scalar content for a marker.
- Remove the marker line and its own inline comment with the selected document; retain all following bytes, including blank lines/comments/directives.
- Keep UTF-8 validation, six supported YAML line breaks, full Decode/semantic comparison and CAS/audit boundaries.
- No frontend change; existing browser acceptance still runs. No new Agent/kernel proof or overall completion claim.

## Context map

| File | Purpose/change |
| --- | --- |
| `pkg/configuration/binding_removal.go` | Narrow the byte deletion interval at an explicit document terminator |
| `pkg/configuration/binding_removal_test.go` | Exact surviving-byte and scalar-marker regressions |
| `internal/controller/binding_removal_test.go` | Actual authenticated API/SQLite accepted source and audit regression |
| `docs/RESOURCE_MANAGEMENT{,.zh-CN}.md` | Replace resolved limitation with exact preservation contract |
| `docs/AUTONOMOUS_DELIVERY.md` | Lease, evidence, next scope |

Dependencies: `grant_source.go` supplies YAML line-break recognition; `internal/management/inventory.go` compiles/replaces source; existing browser removal UI unchanged.
No breaking API, migration, dependency or configuration changes. Context map self-reviewed.

## Review focus

1. Marker-looking text inside scalars must not truncate a document (regression test).
2. CRLF/CR/NEL/LS/PS offsets must preserve exact bytes (table test).
3. Final document comments must survive without an extra policy object (last-position test).
4. Next-document directives must retain required separators (directive test).
5. API persistence/audit must reflect the source-preserving removal, not merely an in-memory transform (SQLite test).

### Task 1: Preserve explicit-boundary comments

**Interfaces:** consumes `Bundle.RemoveBinding(uid string) ([]byte, error)` and existing POST removal API; produces unchanged interfaces with narrower byte deletion.

- [ ] Add `TestRemoveBindingPreservesExplicitBoundaryComments`: first/middle/last position, six YAML breaks, explicit markers and trailing comments; assert retained suffix bytes and absence of only the selected UID.
- [ ] Add `TestBindingRemovalRetainsBoundaryCommentsHTTP`: PUT valid source then POST removal; assert revision, exact persisted suffix, unchanged facts and audit reason.
- [ ] Run focused tests and observe lost comments before production change.
- [ ] Adjust deletion end without reserializing source; add scalar/directive edge coverage.
- [ ] Run focused/full Go ordinary/race/vet/build and locked npm ci/typecheck/lint/build; native CI must cover local Unix/Nginx restrictions without skips.
- [ ] Update bilingual contract and checkpoint, commit; independent whole-branch review, one tested fix pass for blockers.
- [ ] Verify exact remote/local/CI tree, expected-head merge, main CI raw logs, release lease in PR body.

Self-review: single task, no shared interfaces or unspecified domain decisions. Broader Resource/Endpoint editing, identity/request/approval and combined Linux/browser acceptance remain next.
