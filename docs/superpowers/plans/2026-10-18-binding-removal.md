# Binding removal implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:executing-plans; native execution with one fresh whole-branch reviewer.

**Goal:** Safely remove one Binding through real API and browser.
**Architecture:** Source-preserving transform followed by existing complete Replace/CAS/audit; no second persistence authority.
**Tech Stack:** Existing Go/Echo5/GORM/Casbin/SCS and React/TypeScript/Vite/shadcn/Tailwind/i18next/npm.
**Spec:** docs/superpowers/specs/2026-10-18-binding-removal-design.md

## Global Constraints
- JSON16KiB; reason nonblank <=512 bytes; policy4MiB/4096 objects.
- UID-only Binding removal, explicit acknowledge_unprotection:true; no Grant or facts change.
- No applied claim, Agent/kernel change, endpoint edit or Resource deletion.
- User authorizes routine design, push and expected-head merge without repeated confirmation.

## Context Map
| Files | Responsibility |
| --- | --- |
| pkg/configuration/binding_removal.go/test | Remove exact document, semantic equivalence guard |
| pkg/configuration/decode.go, candidate.go, grant_source.go | Reference strict decoding, replacement/compiler and YAML line handling |
| internal/management/inventory.go | Auth/CAS base, transform, existing Replace transaction |
| internal/controller/inventory.go, binding_removal_test.go | Strict bearer/cookie HTTP contract and real SQLite acceptance |
| internal/controller/management.go, browser.go | Routes and stable404 mapping |
| scripts/normalize-management-api.mjs, docs/api/swagger.json | Generated strict acknowledgement schema |
| web/src/features/resources/Resources.tsx, locales | Single removal draft, warning, conflict and cancellation |
| web/e2e/resources.mjs | Real browser/API acceptance and screenshots |
| docs/RESOURCE_MANAGEMENT*, AUTONOMOUS_DELIVERY.md | Usage, evidence and remaining boundaries |

Dependencies: Bundle → Service.grantBase/Replace → publication CAS/audit and full
configuration.Replace → networkplan. Browser consumes existing request/SCS/CSRF.
Context map self-review: no new package or migration, no other worktree touched.

## Review Focus
1. Same-name replacement or wrong-kind UID must never remove another object.
2. Last Binding removal must not masquerade as Grant revocation or access denial.
3. YAML boundary edits must preserve every surviving semantic object and bytes.
4. CAS/audit failure must leave revision, facts and policy unchanged.
5. Stale/ambiguous browser response must not silently rebase or auto-retry removal.

### Task 1: Complete Binding removal slice
**Files:** All mapped above.
**Interfaces:** Bundle.RemoveBinding(uid)([]byte,error); Service.RemoveBinding(p,expected,reason,uid)(InventoryChange,error); POST bindings/{uid}/remove with reason/acknowledge_unprotection,200 UID/revision and ETag.
- [x] Write TestBindingRemovalHTTP and reject/atomic tests; run focused Go command, expect404 RED. Add source boundary tests before transform implementation.
- [x] Implement exact source removal with whole-object comparison; wire service/HTTP/strict schema. Run `go test ./pkg/configuration ./internal/controller ./internal/management -count=1`, expectPASS.
- [x] Add actual browser removal acceptance before UI; publish test-only commit and observe native missing-action RED.
- [ ] Implement removal draft and bilingual warning/confirmation/rebase. Locked npm checks passed; exact final native browser acceptance still required.
- [ ] Run full Go ordinary/race/vet/build; local Unix EPERM recorded, native suite without skip remains required. OpenAPI generated/reproduced by CI.
- [ ] Commit with git-commit skill; fresh full-branch independent review completed, two Important findings in single fix pass; final native receipt GREEN/task-done pending.
- [ ] Record exact heads/trees/raw CI and review; expected-head merge, verify main CI, release lease in PR.

## Integration evidence
PR23 and AUTONOMOUS_DELIVERY.md contain RED evidence, local checks, exact code
trees, independent review and deferred source-comment Minor. Final immutable
native acceptance and merge/main verification are recorded in PR23 after they
actually execute; unchecked integration steps must not be inferred from this plan.
