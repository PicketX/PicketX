# Packaged Nginx Includes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:executing-plans inline;
> one independent whole-branch review before integration.

**Goal:** Generate usable private trusted ingress includes from the installed CLI.
**Architecture:** Thin Cobra command validates bounded installation inputs and
creates two context-specific includes; existing ingress owns traffic/TLS.
**Tech Stack:** Existing Go/Cobra, Nginx auth_request, npm/Playwright/native Linux.
**Spec:** docs/superpowers/specs/2026-10-20-nginx-includes.md

## Global Constraints
- Core base cdee150201a5e33036cd3d73e3be5a8201000698; Agent c0eb1b8 unchanged.
- No domain/policy/schema/UI/kernel changes or new runtime dependencies.
- Only controller-init64lowercase-hex token subset; private files/no secret logs.
- Full native CI/source/archive/main proof and expected-head merge remain gates.

## Review Focus
- Nginx variables/directive injection through route/token/address or include paths.
- Multiple route locations must not share or overwrite original-method context.
- Inherited satisfy/auth_request and untrusted request headers must not bypass.
- Existing/symlink/public/oversized input or partial output must never overwrite.
- Packaged test must use shipped binary and actual Controller, not fake decisions.

### Task 1: Private CLI generation
**Files:** internal/cli/nginx_init{,_test}.go, config.go; bilingual deployment docs.
**Interfaces:** `nginx-init --directory ABS --token-file ABS --route-id UID
--authz-address IP:PORT` produces location.conf/server.conf, prints paths only.
- [ ] CLI real filesystem tests missing command/validation/private outputs/multiple
  routes/IPv6/token subset/no-overwrite RED. Run go test ./internal/cli -run Nginx;
  Expected: unknown command, then after implementation PASS.
- [ ] Minimal CLI implementation; run full cli package. Expected PASS.
- [ ] Document exact placement/trust/rotation/no-grant semantics in both guides.
- [ ] Commit feat(cli): generate private Nginx authorization includes.

### Task 2: Native installed ingress acceptance
**Files:** e2e/ingress.mjs, installed-ingress.mjs, install.mjs, CI install dependency.
**Interfaces:** Task1's exact two filenames/flags; actual binary PATH empty; existing
real ingress fixture exposes request count and authorization assertions.
- [ ] Add actual installed binary consumer before implementation; existing CLI must
  fail unknown command. Native test checks allowed/revoked/wrong context/caller and
  outage with no denied app hit, real dual-stack callback, multiple route mapping.
- [ ] Integrate real installed proof without changing existing install revisions/
  service recovery assertions. Record helper proof in install-report.
- [ ] Full Go normal/race/vet/build, locked Web and Node; inherited local Unix EPERM
  explicitly reported, native nofail/skip required. Expected native3 jobs green.
- [ ] Commit acceptance; one fresh whole-branch review; one root RED→GREEN fix pass;
  verify actual PR source/four archives, expected-head merge, repeat main evidence.

Self-review: tasks share CLI flags/files only; Task2 consumes literal interface
above. Native suite is mandatory for actual nginx and privileged service support.
