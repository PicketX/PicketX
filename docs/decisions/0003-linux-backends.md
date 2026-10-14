# ADR 0003: Linux backend validation priority

> English (default) | [简体中文](0003-linux-backends.zh-CN.md)

- Date: 2026-10-14
- Status: Accepted evaluation priority; M0 kernel validation completed, production integration pending

M0 was merged through [PicketXD PR #1](https://github.com/PicketX/PicketXD/pull/1). Its final source passed [17 required kernel scenarios](https://github.com/PicketX/PicketXD/actions/runs/38547987642) on Linux 6.17.0-1022-azure with Go1.27.0, nftables v0.3.0 and netlink v1.3.1. See [measured findings](https://github.com/PicketX/PicketXD/blob/main/docs/validation/M0.md). This does not establish a production Agent or other kernel compatibility.

## Decision

Evaluate `google/nftables` first for nf_tables configuration, initially at v0.3.0. Evaluate `vishvananda/netlink` first for the required strictly scoped conntrack revocation. `ti-mo/conntrack` remains a candidate for NEW/UPDATE/DESTROY event observation; event subscription is not considered solved by selecting a deletion API.

No mandatory C library or system `nft`/`conntrack` command dependency is introduced in the default runtime. Diagnostic/integration-test environments may install those tools independently.

## Why these boundaries

Kernel conntrack state expressions and marks can be configured through nftables without a user-space conntrack library. The latter is needed when the Agent actively queries, observes or deletes connection entries. Strict revocation is required in the current design; it cannot be reduced to graceful expiry.

Kubernetes/kube-proxy, CNI plugins and OVN-Kubernetes use vishvananda/netlink for targeted deletion. Antrea and Kindnet have ti-mo/conntrack uses for flow observation. These are evidence of adoption, not proof that PicketX's expiry, overlap or failure semantics are already implemented.

`ConntrackDeleteFilters` performs a table dump, Go-side matching and per-entry deletion. Validate cost with increasing table size and handle incomplete dumps and partial failures. The application must calculate withdrawn effective permission after union/deny evaluation, restrict deletion to PicketX-owned flows, and preserve connections still allowed by other Grants.

## Acceptance

Run the [M0 design](../superpowers/specs/2026-10-14-linux-data-plane-validation.md) and [implementation plan](../superpowers/plans/2026-10-14-linux-data-plane-validation.md). Validate IPv4/IPv6, interval/concatenated sets, expiry, failed batches, scoped strict/graceful revocation, NAT tuple interpretation and restart. Mock-only or skipped kernel tests do not satisfy M0.

Keep backends replaceable. Select and pin the actual conntrack dependency at validation setup; the plan starts from the published v1.3.1 baseline and treats any missing/error-reporting behavior as a validation finding, not permission to suppress it. Event-observation integration is a later scoped task.

## References

- [google/nftables](https://github.com/google/nftables)
- [vishvananda/netlink](https://github.com/vishvananda/netlink)
- [Kubernetes conntrack code](https://github.com/kubernetes/kubernetes/blob/master/pkg/proxy/conntrack/conntrack.go)
- [CNI plugin conntrack helpers](https://github.com/containernetworking/plugins/blob/main/pkg/utils/conntrack.go)
- [OVN-Kubernetes connection deletion](https://github.com/ovn-kubernetes/ovn-kubernetes/blob/master/go-controller/pkg/util/net_linux.go)
- [Antrea flow exporter](https://github.com/antrea-io/antrea/blob/main/pkg/agent/flowexporter/connections/conntrack_linux.go)
- [Kindnet connection events](https://github.com/kubernetes-sigs/kindnet/blob/main/pkg/conntrack/conntrack.go)
