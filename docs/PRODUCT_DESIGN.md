# PicketX Product Design

> English (default) | [简体中文](./PRODUCT_DESIGN.zh-CN.md)

| Field | Value |
| --- | --- |
| Version | v0.4.0 |
| Date | 2026-10-08 |
| Status | Agreed product baseline; not a claim of implemented features |
| Architecture | [ARCHITECTURE.md](./ARCHITECTURE.md) v0.7.0 |

## 1. Product positioning

PicketX is a simple, dedicated-client-free, on-demand access authorization system. It uses IP addresses and CIDRs as network authorization subjects, dynamically opens specified resources through predefined policies, self-service requests, or approval, and manages permission expiry and revocation. It supports shared access across devices and applications from an authorized network source, can operate independently or complement existing VPN and application authentication systems, and can evolve toward finer protocol-level authorization.

Short description: **Simple, client-free access authorization built around IP/CIDR sources and time-bounded resource permissions.**

“Client-free” means no dedicated PicketX endpoint client, tunnel software, or per-device enrollment. Browsers and existing SSH/database/application tools remain in use. Server-side components still need deployment. Organization size is not a product restriction.

## 2. Problem and intended outcomes

Many personal, development, test, demonstration and administrative services should not remain reachable from every source. Their owners still need convenient access from changing locations and multiple devices, and must sometimes grant temporary access to testers, customers or external collaborators.

PicketX turns an explicit access decision into a source-scoped permission with a managed lifetime. It reduces manual firewall changes, repeated endpoint setup, and forgotten temporary permissions. Existing application authentication remains in place. A grant does not create connectivity to an otherwise unreachable private network and does not provide traffic encryption.

## 3. Primary scenarios

| Scenario | User action | Expected outcome |
| --- | --- | --- |
| Personal access | Sign in through a browser and select resources and duration | The current approved source can access the selected services |
| Multiple devices | Authorize a shared egress IP or an explicitly approved CIDR | All matching devices share network reachability without separate PicketX clients |
| Multiple applications accessing one service | Authorize the source once, then use an IDE, database client, development application, CLI or script | Applications matching the source and resource scope share network access without separate PicketX integration |
| External testing or collaboration | Request access; resource owner or approver accepts | Only the approved source, resources and duration are granted |
| Existing VPN | Connect through the existing VPN and request resource access | An additional resource gate limits standing reachability without replacing the VPN |
| Static administration | Declare a trusted IP/CIDR and its resource scope | Permanent or absolute-expiry authorization, including Controller-free YAML operation |

Shared access is intentional. The UI must describe the actual address range being authorized. Same Wi-Fi does not guarantee identical egress, and IPv6 addresses may differ or change. Portal and resource traffic can follow different routes; a successful portal login alone does not establish the enforcement source.

### 3.1 Cross-application access to one service

A typical case is one test database accessed from the IDEA database tools, a standalone database client, a local development application, and a CLI or script. After one browser-based IP/CIDR authorization, these tools can establish native-protocol connections when their source and resource match. They do not need separate PicketX plugins, portal login implementations, or shared browser cookies.

What is shared is network access eligibility, not application credentials, database accounts, or business permissions. Service authentication remains in force. If a resource also requires protocol-level user or session authorization, each tool must satisfy that requirement. With source-only authorization, other applications using the same source also receive the same scoped reachability; this is not per-process allowlisting.

Containers, remote development environments and different proxies may use different egress addresses. Coverage depends on the source observed at the enforcement point matching the authorized IP/CIDR. Network Grants do not require the business client to hold portal cookies; validity follows Grant/Lease policy.

### 3.2 First end-to-end scenario: Gitea

After portal source authorization, users access one Gitea service through browsers, Git CLI and IDEs. The network gate controls reachability of declared entries without requiring Git tools to follow browser login redirects or share portal cookies. Gitea accounts, tokens, SSH keys and repository permissions remain in force; HTTP(S) and SSH entries are declared separately.

## 4. Actors and authorization objects

| Concept | Role | Must not be confused with |
| --- | --- | --- |
| Requester | Authenticated person or API principal allowed to ask for access | Every person or device later using the granted source |
| Approver / administrator | Confirms scope, conditions and duration | The network subject receiving the permission |
| Source | IP/CIDR receiving network permissions | A verified person or device identity |
| Resource / Endpoint | A logical protected service with multiple independently selectable entries | Every port on the host or the application's internal permission model |
| Grant | One independent Subject-to-Scope contribution; network Subjects are IP/CIDR | A portal login session or a single kernel set element |
| ProtocolSubject | Identity verified through a trusted protocol integration | A username merely parsed from unverified traffic |

Requester authentication, source authorization and protocol authentication are separate. One requester may request several sources, and several requesters may grant the same source through authorized workflows. Authorization records explain who granted access, not who produced every subsequent packet.

## 5. Permission model

A Grant describes one typed Subject, one or more resource/Endpoint Scopes, a shared validity interval and conditions, authorization provenance, and revocation state. For network access the Subject is a source IP/CIDR. Self-service defaults to an observed IPv4 /32 or IPv6 /128. Broader CIDRs are explicitly selected and authorized; they are not inferred from a single observed host.

For a protected resource, effective network permissions are the union of currently valid, matching Grants, constrained by explicit deny policies and conditions. An address may match both exact-host and CIDR Grants. No applicable allow means no network permission. `ALL` means all PicketX-managed resources, not all host traffic.

Each Grant keeps its own lifetime and provenance. Expiring or revoking one Grant removes only its contribution. For example, website permission until 16:00 and database permission until 18:00 must not be compiled into one all-resource permission lasting until 18:00. Conversely, expiry of one website Grant must not remove access still authorized by another valid website Grant.

Fixed expiry is sufficient for the basic workflow. Optional activity renewal must stay within hard deadlines and must not revive expired Grants. Approval, permission validity, and actual enforcement are separate states.

### 5.1 Resource entries and deployment

A Resource represents a service such as Gitea, with HTTPS and SSH as separate Endpoints. Users can request named entries, label-matched entries, or the whole Resource. Omitting an Endpoint restriction means all current and future entries; the UI must show that dynamic scope explicitly. Network entry type and action default to `network` and `connect`.

Endpoint destination addresses are optional IP/CIDR arrays used for traffic matching, not proxy backends. EnforcementBinding assigns multiple selected resource entries to every selected Agent; it defines protection placement, not access permission. Agents receive only their relevant complete protection and authorization state. Configuration ownership and conflict checks provide isolation without introducing Namespace.

## 6. User workflow and visible results

1. The requester opens the browser portal and authenticates.
2. The portal shows the observed source, allowed resources, duration limits, and source-sharing meaning.
3. The requester selects a permitted source scope and resources. Predefined policy grants self-service access or routes it to an authorized approver.
4. Approval records the accepted scope and creates independent Grants. The Controller computes and distributes the resulting desired permissions.
5. The UI shows applying, applied, partial or failed outcomes from enforcement feedback. Approval alone is not success.
6. Users access resources with their existing tools and application credentials.
7. Grants expire or are revoked. The UI distinguishes the desired withdrawal from its confirmed application and identifies unreachable nodes.

The administrator can inspect current effective permissions, contributing Grants, remaining validity and per-node status. Page loads and polling show loading, failure and last successful refresh; stale data is not presented as a confirmed current state.

## 7. Expiry and revocation

| Mode or action | Meaning |
| --- | --- |
| Graceful expiry | Remove the expired contribution from new-connection permissions; existing authorized flows may continue |
| Strict revocation | Recompute permissions, withdraw relevant access, then invalidate affected connection state without indiscriminately disrupting independently authorized access |
| Individual Grant revocation | Withdraw only that Grant's contribution |
| Source-session invalidation | An explicitly broader operation with stated scope; not automatic attribution to a portal user |

Deleting conntrack state is not the same as sending a TCP reset. UI text must explain connection handling and any bounded expiry propagation delay. A fixed deadline is not silently extended by Controller disconnection or Agent restart.

## 8. Authorization records and scope

Record who requested and approved, which IP/CIDR and resources were authorized, the requested and granted duration, reason/policy, contributing Grant IDs, actual apply outcome, and expiry/revocation outcome. Protocol-level decisions may record minimal verified identity and decision metadata where applicable.

Full SQL history, shell recording, page histories, file contents and application behavior reconstruction are outside the current product scope. Operational diagnostics and connection-lifecycle observations remain valid engineering functions; they do not establish who performed a business operation.

PicketX does not replace a VPN tunnel, application accounts, business permissions, WAF, IDS/IPS, or a full audit platform. Its network grant mechanism does not establish endpoint trust or encryption. It can complement those systems through a clear enforcement boundary.

## 9. Core and expansion scope

| Core baseline | Expansion direction |
| --- | --- |
| IP/CIDR source authorization; IPv4 and IPv6 | Protocol-specific user/session/request constraints |
| Resource scope, independent Grants, permission union | Additional protocol adapters and integrations |
| Predefined policy, self-service, basic approval | Advanced approval and external attributes |
| Expiry, revocation, truthful status and authorization records | Policy simulation, richer governance and retention options |
| Controller Mode and Static Manifest Mode | HA, hosted operation and OEM packaging according to demand |

Protocol support and authorization precision are independent dimensions. Network permissions remain IP/CIDR-based even when a protocol adapter verifies a user. Where both layers are configured, both must allow access. Protocol parsing alone is not identity verification, and a protocol allow does not automatically enlarge a network Grant.

HTTP integration through existing proxies is a natural extension for websites. Further database or SSH integrations must explicitly address identity verification, encryption, enforcement and lifecycle. None implies a commitment to universal traffic inspection or full behavior recording.

### 9.1 Unified authorization core and multiple execution implementations

The product uses a typed Subject–Resource–Action–Conditions core model. IP/CIDR is the network subject type, not the only possible authorization subject. Verified users, service identities and request context require corresponding trusted integrations. Requester, granted subject and actual request identity remain separate.

PicketX provides the unified control plane and authorization core; PicketXD is the official Linux enforcement implementation. Further capabilities integrate through Agents/adapters or AuthZ hooks. Adapters must establish that they can observe and enforce policy requirements; unsupported identity or revocation semantics cannot silently degrade. WASM is a controlled policy extension introduced as needed, outside the packet fast path.

### 9.2 Separate workflow from standard configuration

The control plane owns users, requests, approvals and Web/API business models. Authorized results are translated into the standard Resource/Endpoint/Grant/Policy/EnforcementBinding contract. Agents consume this contract through mutually exclusive Controller or Static Manifest providers, normalize the same defaults and semantics, and report execution status through a separate feedback contract.

This permits independent workflow and Agent evolution without duplicating authorization semantics. A local configuration need not include the control plane's user or approval database. Export, validation, explanation and reproducible troubleshooting should use the same configuration contract. Full schemas and unresolved lifecycle details are tracked in [Configuration and Domain Model](CONFIGURATION_MODEL.md); they are not implemented features.

## 10. Deployment and retained decisions

- Controller Mode uses the Controller as the sole security desired-state authority. Local runtime settings never add Resource, Policy, Grant/Lease or AuthZ permissions.
- Static Manifest Mode uses one YAML directory as the sole security authority. Full candidate validation, reconciliation, source-bound LKG and reload apply; browser self-service and centralized approval require a Controller.
- Host installation is the default; explicit gateway enforcement on existing routes remains supported by the design, subject to adapter capabilities and a future NAT/traffic-scope specification. PicketX does not configure network routing.
- The Linux implementation retains nftables/Netfilter and conntrack. NFQUEUE/TPROXY are optional enforcement capabilities, not the product identity.
- The default Agent capability model remains intact; individual resources select the required path and optional capabilities can be disabled.
- The frontend uses React, TypeScript, Vite, shadcn/ui, Tailwind CSS, i18next and npm, with explicit polling/freshness feedback.
- Licensing intent remains `picketxd` GPL-3.0-or-later plus alternative OEM terms and contributor relicensing rights; other designated components use Apache-2.0. See the architecture for third-party dependency constraints.

- Both backend and official Linux Agent use Go; `PicketX/PicketX` holds Controller/Core/Web/public protocols and SDKs, while `PicketX/PicketXD` holds `picketxd`.
- User portal and administration share one React application, common components and theme, with separate layouts and feature pages.
- Maintain a small set of project pagination/table compositions; minimize changes inside official UI components, attach business behavior through props/events, and review component-source diffs during upgrades.
- i18next supplies English (default) and Simplified Chinese; explicit requests and `usePolling` form the initial loading strategy without TanStack Query. TanStack Table may be adopted independently for tables.

## 11. Acceptance and validation

The first usable workflow must demonstrate multi-device sharing, cross-application access to one service, an external request approved from a browser, truthful apply status, and expiry/revocation. It must also demonstrate overlapping Grant correctness, IPv4/IPv6 source handling, Controller/Agent interruption behavior, and strict isolation of configuration providers.

Deployment and operation should reduce the combined effort of requesters and administrators. User validation measures whether requests can be completed without coaching, whether teams return to use the tool, and whether they accept a concrete paid offering. These are validation goals; adoption, market size, pricing and willingness to pay are not established by this document. Product positioning is not restricted to small teams.

Cross-application acceptance uses one test service and at least two existing tools, such as an IDE database tool and a CLI client. After one source authorization, both matching clients must establish connections without PicketX plugins or portal cookies. After expiry, with no other valid Grant, both must be denied new connections; existing flows follow the selected policy. Service authentication must still succeed. Other egress sources outside the authorized scope do not inherit permission.

## 12. Revision and authority

This v0.4.0 product baseline supersedes the positioning and scope of the archived v0.1 product document. Detailed technical decisions are maintained in architecture v0.7.0. Older business plans and research reports are historical inputs, not current scope or pricing commitments. English is the default repository document; the Chinese edition is maintained alongside it.

Revision v0.4.0 separates business, standard configuration and execution models; adds multi-Endpoint Resource selection, multi-resource/multi-Agent bindings and per-Agent projection; defers Namespace; synchronized with architecture v0.7.0 and configuration model v0.1.0. The previous v0.3.0 stack, repository and Gitea decisions remain in force.
