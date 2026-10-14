# ADR 0001: RBAC authorization and periodic policy synchronization

> English (default) | [简体中文](0001-rbac-authorization.zh-CN.md)

| Field | Value |
| --- | --- |
| Date | 2026-10-14 |
| Status | Accepted design; runtime integration pending |
| Scope | PicketX management and requester authorization; applicable trusted request-time identity integrations |

## Context

Default Casbin RBAC loads both permission rules (`p`) and grouping rules (`g`) into memory and builds role relationships. Sharing permissions through roles reduces duplicated `p` rules but does not prevent user-membership `g` rules from growing with the user population. GORM persistence does not make enforcement query each relationship on demand.

Casbin's official scaling guidance distinguishes request throughput from policy size. Multiple replicas distribute requests but do not reduce per-process policy memory when every replica loads the same full policy. Filtered loading can reduce the loaded `p` and `g` scope. External Role Managers offer another integration boundary, but their querying, caching and initialization behavior depends on the implementation.

Argo CD reads the current user's groups from verified identity claims and evaluates those groups against shared authorization rules. Harbor constructs an evaluator from the current user's roles and permissions. These are integration references, not PicketX capacity guarantees.

## Decision

### 1. Start with RBAC

Use Casbin with the official GORM Adapter, initially with RBAC. Add ABAC only when actual requirements need attributes such as resource ownership or device/request conditions; no ABAC model or additional engine is selected now.

Casbin evaluates roles, resources and operations, including PicketX management actions and eligibility to request access. PicketX button visibility maps to these same operation permissions; the backend enforces every protected operation. This does not replace a protected third-party application's internal permissions.

The authorization Core continues to own IP/CIDR matching, independent Grant contributions, permission union, deadlines, Lease renewal, network compilation and connection revocation. Do not expand these into one Casbin policy per packet, connection or temporary Grant, or infer a protocol identity from an IP Grant's requester.

### 2. Keep user membership outside the shared enforcer

| Data | Owner / use |
| --- | --- |
| User-to-group and user-to-role membership | Trusted identity/business layer and its authoritative storage |
| Current user's applicable groups/roles | Backend-resolved request authorization context |
| Group-to-role bindings and role inheritance | Shared Casbin `g` rules |
| Role-to-resource/operation permissions | Shared Casbin `p` rules |

Do not mirror the entire user directory's memberships into the shared Casbin policy. Resolve only the current user's applicable groups/roles through the identity/business boundary. Local users and OIDC users can use the same authorization input; the storage/library for login sessions remains a separate decision.

Names must be stable and distinguish user, group and role identifiers. Where a tenant scope exists, resolve and enforce that scope rather than accepting a caller-selected role or cross-tenant membership. This decision does not introduce a Namespace or finalize tenancy schemas.

The application needs an authorization wrapper/model to consume the trusted subject context; the GORM Adapter alone does not implement this behavior. Evaluate all applicable groups/roles with explicit deny precedence and default denial. A shortcut returning success on the first allowed role must not bypass a matching deny from another role.

A Session may cache the current user's groups/roles if its freshness and membership-revocation behavior are defined. Periodic Casbin policy reload does not refresh those memberships. Computing a complete permission list at login is not selected as the baseline.

### 3. Persistence and process lifecycle

- Use `github.com/casbin/casbin/v3` and `github.com/casbin/gorm-adapter/v3`. Pin and verify compatible versions when runtime integration begins; do not add unused dependencies to the current scaffold.
- Reuse the existing `*gorm.DB` through the official adapter's `NewAdapterByDB` boundary. SQLite and PostgreSQL retain their selected official GORM drivers; no separate business driver choice is introduced for Casbin.
- Keep the official adapter's default table initialization and AutoMigrate behavior for its own Casbin schema. This is separate from the application's versioned business migrations and does not promise arbitrary historical data transformations.
- Create one shared `SyncedEnforcer` per process at startup, load the required policy scope and inject an application-facing authorization interface. Casbin/GORM types remain inside integration implementations; no global service lookup or shared mutable current-user field.
- Keep registered policy independent of login Sessions. Do not create an enforcer or reload policy for every authorization request.
- Repeated-decision caching is an optional later optimization. It is not selected by this baseline and does not solve policy-size growth.

### 4. Periodic synchronization only

Use a configurable, positive policy synchronization interval, supplied through the normal typed runtime configuration boundary. Each replica reloads the authoritative policy on that schedule. Do not introduce a Redis/Valkey Watcher, database notification listener or dispatcher for Casbin policy synchronization in this baseline. A separate login-session backend may still use a database, Redis or Valkey.

Changes from other replicas become visible after a successful scheduled reload. A larger interval reduces synchronization work but lengthens authorization-change and revocation propagation. Startup diagnostics must make that trade-off visible. Failed reloads are observable errors, not a successful freshness update; a healthy-reload interval is not a bound during a persistent failure.

This interval applies only to Casbin policy refresh. It does not redefine Agent desired-state synchronization, Grant deadlines, Lease activity checks, UI polling or connection-revocation contracts.

If filtered loading is introduced later, retain the filter during every scheduled reload. A full `LoadPolicy()` must not accidentally replace the scoped load with all global rules.

### 5. Startup guidance by measured policy scale

At startup report the actual loaded `p` and `g` counts separately, their total and the initial loading duration. Use rule-count bands together with loading duration to provide guidance:

| Observed scale / cost | Guidance |
| --- | --- |
| Small policy and inexpensive loading | Report the selected interval; no enlargement recommendation is necessary |
| Larger policy or loading that occupies a substantial part of the interval | Recommend a longer interval and explain the propagation delay |
| Very large policy or unacceptable load/memory cost | Also recommend reviewing shared-role modeling and tenant/domain subset loading |

Rule-count bands and diagnostic thresholds are configurable. Guidance does not silently change the configured interval. Exact keys, default interval and numerical thresholds remain implementation decisions; official examples and benchmarks are not universal capacity limits.

Measure resident memory, enforcement latency and reload cost with representative models before adopting a capacity tier. Increasing the reload interval does not reduce the resident size of a loaded policy. Button-level operations do not by themselves cause user-count multiplication, but group/role bindings and permission rules can still grow.

## Implementation and validation boundaries

This ADR records selection, not implemented authentication, authorization, storage or configuration fields. The current scaffold remains unchanged.

Before claiming runtime support, verify trusted membership resolution, multi-role deny precedence, backend checks matching UI operation permissions, both selected databases, scoped concurrent reloads and visible refresh failures. Verify that Session membership freshness, Casbin policy propagation and Agent Grant expiry remain independent. Review startup suggestions under different policy sizes and loading costs without automatically rewriting configuration.

## References

- [Casbin RBAC](https://casbin.apache.org/docs/rbac/)
- [Production deployment](https://casbin.apache.org/docs/production-deployment/)
- [Performance optimization](https://casbin.apache.org/docs/performance/)
- [Policy subset loading](https://casbin.apache.org/docs/policy-subset-loading/)
- [Role Managers](https://casbin.apache.org/docs/role-managers/)
- [Official GORM Adapter](https://github.com/apache/casbin-gorm-adapter)
- [Argo CD authorization](https://argo-cd.readthedocs.io/en/stable/operator-manual/security/#authorization)
- [Argo CD claims enforcement](https://github.com/argoproj/argo-cd/blob/master/server/rbacpolicy/rbacpolicy.go)
- [Harbor current-user policy adapter](https://github.com/goharbor/harbor/blob/main/src/pkg/permission/evaluator/rbac/casbin_adapter.go)
