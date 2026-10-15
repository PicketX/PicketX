# Static configuration check

> English (default) | [简体中文](README.zh-CN.md)

These files are read-only diagnostic examples, not a deployed Agent or live network facts. They open no firewall ports. Example grants deliberately use fixed absolute dates; checking them after their deadlines produces no active permission.

From the repository root:

```bash
go run ./cmd/picketx config check --file examples/static/gitea.yaml --facts examples/static/facts.json --at 2026-10-15T09:00:00Z
```

The result has `state: validated`, `enforcementConfirmed: false`, a policy-only hash, object name/UID mappings, diagnostics and complete per-Agent planner projections. At 09:00 UTC the IPv4 source has HTTPS permission and the independent IPv6 source has SSH permission. At `2026-10-15T10:00:00Z` only SSH remains; at `2026-10-15T12:00:00Z` both Grants are inactive, with both entries still protected. Without `--at`, evaluation uses the current instant. Planner fields in projections/facts use the existing evolving Go diagnostic representation, not a stable transport protocol.

Use `empty.yaml` to check an explicit complete withdrawal. An empty/truncated file is an error. Binding removal withdraws protection; simply removing every Grant keeps the protection and denies new connections. This command performs no withdrawal itself.

The YAML has one complete publication per file. A future runtime provider must atomically publish candidates and retain the previous valid state on reload failure; this command does not implement directory watching or LKG. Never substitute these manually supplied facts for backend observations.

The alpha contract and limits are in [M2A design](../../docs/superpowers/specs/2026-10-15-static-configuration-design.md). Resource references include name and pinned UID. Before using a fresh Resource UID, update its explicit references; this is an authoritative configuration change, not automatic approval. Broad Resource selectors, conditions, renewal, workflow objects, status, protocol identities and unsupported enforcement requests fail rather than being ignored.
