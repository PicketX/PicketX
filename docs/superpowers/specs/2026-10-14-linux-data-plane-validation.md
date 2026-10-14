# M0: Linux data-plane validation design

> English (default) | [简体中文](2026-10-14-linux-data-plane-validation.zh-CN.md)

> Execution record (2026-10-15): user accepted M0; [Agent PR #1](https://github.com/PicketX/PicketXD/pull/1) is merged. [Final tested source](https://github.com/PicketX/PicketXD/commit/13fdf2d9aacd79be74578ebe9c23af19a6b33d6a) passed [all 17 required real-kernel scenarios](https://github.com/PicketX/PicketXD/actions/runs/38547987642), unit/race/vet/build and the independent-review fix pass. Original planning text below is retained as the reviewed historical input. Production integration remains pending.

- Date: 2026-10-14
- Status: Proposed execution design for review; no kernel acceptance claimed
- Implementation repository: [PicketX/PicketXD](https://github.com/PicketX/PicketXD)
- Authority: [Architecture](../../ARCHITECTURE.md), [configuration boundaries](../../CONFIGURATION_MODEL.md), [ADR 0003](../../decisions/0003-linux-backends.md)
- Plan: [Implementation plan](../plans/2026-10-14-linux-data-plane-validation.md)

## 1. Outcome

Build a repeatable Linux integration suite that proves whether the selected Go backends can implement PicketX's network enforcement contract. Produce real packet/connection observations and explicit pass/fail/unsupported evidence before implementing the full Agent or Controller workflow.

The first product workflow remains Gitea: browser approval, source-scoped HTTPS/SSH reachability, apply status and expiry/revocation. M0 is its data-plane prerequisite, not that completed workflow.

## 2. Verified starting point

On 2026-10-14, PicketX main is `dc62c73fc3017f2fb2db1f60e8cd45eec02d27a6`: Go 1.27.0, Echo liveness and a React scaffold. PicketXD main is `2b4400d05c9c7348ca6ffc436f63e0819a4d1dc7` and contains only LICENSE.

PicketXD's actual LICENSE is Apache-2.0, while the agreed Agent licensing direction is GPL-3.0-or-later/OEM. Align the empty Agent repository's community license and notices in its bootstrap change before introducing Agent implementation. The current documentation PR does not change that LICENSE or grant any additional OEM rights.

The current execution environment has no Go compiler, no effective/bounding Linux capabilities, and cannot create a user/network namespace. It cannot provide real M0 kernel evidence. Run the future suite on an explicitly capable Linux machine or isolated CI runner.

## 3. Scope and boundaries

Include IPv4/IPv6 TCP/UDP admission, explicit denies, source/destination/port interval and concatenation support, native element timeout, absolute-deadline recovery, failed nftables batches, graceful expiry, strict scoped revocation, NAT tuple matching and namespace ownership.

Exclude production config parsing, stable manifest/protobuf/API schemas, users/RBAC/sessions, Controller transport, approval UI, dynamic activity renewal, event subscription, production LKG storage and HA. The required event-observation capability remains a later task; selecting deletion does not remove it.

Use internal test-only normalized snapshots with explicit addresses, protocol, port range and absolute deadlines. Overlap scenarios supply before/after snapshots representing expected permission union. They validate backend behavior and connection selection, not a completed production Grant compiler; production union/defaulting will require shared Core conformance tests in M1.

The implementation lives only in PicketXD. Do not copy GPL Agent implementation into the Apache-2.0 Controller/Core repository. Keep the fixture DTOs private to the validation suite.

## 4. Dependency and environment baseline

Use Go 1.27.0 or a reviewed compatible patch, `github.com/google/nftables` v0.3.0, and initially `github.com/vishvananda/netlink` v1.3.1; pin go.mod/go.sum. Record failures or missing behaviors before changing a dependency version. No additional runtime C library or nft/conntrack executable dependency is selected.

Tests may use Linux `ip`/`unshare` and diagnostic tools. Build/download dependencies before the privileged test phase. Enable kernel tests only with `PICKETX_KERNEL_TESTS=1`. Without opt-in, they may explicitly skip; with opt-in, missing capability, setup failure or unsupported required kernel behavior is a failure, never an acceptance pass.

Run inside an isolated parent network namespace; create test client/server namespaces and links there. Use documentation-only address ranges (IPv4 192.0.2.0/24 and IPv6 2001:db8:1::/64) and local TCP/UDP echo fixtures, not production SSH/Gitea. No test modifies host-owned firewall/routing state.

## 5. Ownership and application order

Within the isolated enforcement namespace, own only `table inet picketx` and `picketx_` objects. Use PREROUTING at priority -110 as the baseline, with explicit destination matching. The test-only connection marker is `0x5049434b`; this is not a production mark allocation.

Apply an entire candidate nftables batch, confirm its result, then perform scoped conntrack invalidation when strict withdrawal is requested. A failed nftables batch retains the preceding usable rules and does not authorize conntrack deletion for an unapplied candidate.

Derive withdrawn access from the old and new effective snapshots. Delete only owned flows that lost effective permission; retain independently allowed, out-of-scope and unmarked flows. NAT matching uses the original tuple corresponding to the pre-DNAT policy boundary and tests the reply tuple explicitly. Conntrack deletion is not a TCP reset or an application socket-close guarantee.

Treat an interrupted dump, deletion error or observed remaining matching flow as a failed/incomplete operation. Return evidence of partial results. Never flush the complete host conntrack table as recovery.

## 6. Scenario acceptance

| Scenario | Required observation |
| --- | --- |
| IPv4/IPv6 default deny | Unapproved sources cannot create TCP/UDP access to protected endpoints |
| Scoped allow | Approved source reaches the declared endpoint; another source/port remains denied |
| Overlap | Removing an exact-host contribution retains access covered by another valid prefix contribution |
| Independent deadlines | Website access expires while separately authorized database-port access remains |
| Deny precedence | A matching deny defeats an allow for the same protected scope |
| Set combination | CIDR, destination, protocol and port interval combinations are verified on the actual kernel |
| Native timeout | New access stops after the configured element timeout without a Controller process |
| Graceful expiry | Existing authorized TCP traffic remains usable; new connections are denied |
| Strict revoke | Subsequent traffic on withdrawn flows is blocked; unrelated/still-authorized flows continue |
| No TCP resurrection | Old ACK/PSH packets cannot recreate access after state deletion without a valid new SYN authorization |
| UDP re-check | A datagram after deletion creates a new pseudo-flow and must satisfy current policy |
| Invalid batch | The candidate fails and preceding network behavior remains usable |
| NAT scope | Strict deletion matches the intended original/reply mapping and preserves an unrelated mapping |
| Restart/deadline | Re-applying a test snapshot with the same absolute expiry does not restart its lifetime |
| Missing privilege | Opted-in kernel execution fails visibly before any rule mutation |
| Partial cleanup | Incomplete/error cleanup is reported and cannot be represented as successful revocation |

Use polling with explicit socket/test deadlines, not fixed sleeps as proof of expiry. Generous test deadlines prevent scheduling noise; they do not define a production expiry-delay promise.

## 7. Evidence and completion gate

Archive Go test JSON and a machine-readable environment/scenario report containing kernel release, Go/library versions, actual scenarios executed, results, capability failures and deletion counts. Separate unit, kernel, skipped and unsupported outcomes.

M0 is complete only when required scenarios execute successfully on real Linux and the findings are reviewed. A green job that skipped kernel tests is insufficient. Record the tested kernel envelope; do not claim every distribution or older kernel works.

The suite is reusable engineering validation, not a production Agent binary. Start M1 schema/Core/workflow implementation only after reviewing the measured findings and any explicit design adjustments.

## 8. Workflow

Review this design and the linked plan, then execute tasks in order with failing tests before code, focused commits, real checks, code review and a PR. Prefer direct execution in the current session for this small dependent sequence; delegated execution is available only if requested. Future frontend work uses the installed UX/UI skills and repository-owned design decisions.
