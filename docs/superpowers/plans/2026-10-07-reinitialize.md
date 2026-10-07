# Repository Initialization Plan

> For agentic workers: use superpowers:executing-plans to implement this plan.

**Goal:** Replace the former Rust scaffold with a minimal Go and React development baseline while retaining Git history.

**Architecture:** One Go module at the root; an independently built Web project under `apps/web`. PicketXD stays outside this repository. This is scaffolding, not an authorization implementation.

**Tech Stack:** Go standard library, React, TypeScript, Vite, shadcn/ui, Tailwind CSS, i18next, npm.

**Spec:** [Architecture](../../ARCHITECTURE.md), [Product](../../PRODUCT_DESIGN.md).

## Global Constraints

- English default, Simplified Chinese alongside.
- Apache-2.0 for this repository; preserve third-party notices.
- No fake access grants or claims of working protection.
- No history rewrite; publish one replacement commit based on the inspected main SHA.

## Review Focus

- Unknown API paths must not report healthy or return an application success.
- Liveness does not mean authorization readiness.
- Browser language controls must be keyboard accessible and update the document language.
- Frontend dependencies must install reproducibly with npm ci.
- No credentials, build output, Agent GPL code or old Rust scaffold in the new tree.

## Task 1: Bootstrap and validate

**Files:** root module/license/readmes, `cmd/picketx-controller/main.go`, `internal/controller/http.go` and tests, `apps/web`, `.github/workflows/ci.yml`, `docs`.

**Interfaces:** Controller serves only `GET /healthz`; Web starts independently through Vite. No business API contract is introduced.

- [x] Add a health-handler test for JSON liveness, unknown paths and unsupported methods; observe failure before implementation.
- [x] Implement the minimal handler and signal-aware local development server.
- [x] Generate Vite React/TypeScript project, configure shadcn/Tailwind and bilingual initialization screen.
- [x] Add development instructions, current bilingual design documents and CI.
- [x] Run Go tests/vet/build and npm clean install/typecheck/lint/build; smoke-check controller endpoints.
- [ ] Review the complete new tree, then update main with an expected-head guard and verify remote content.

## Execution Notes

The user explicitly requested repository reset/reinitialization after approving the architecture. Execution continues in a fresh local checkout without another design approval. Only tracked files are replaced; Git history is retained.

Validation: Go test/race/vet/build, HTTP liveness/404/405/shutdown smoke, npm ci/typecheck/lint/build and documentation links passed. An independent source review found no critical or important issues. Browser smoke remains unverified because the environment could not download a valid browser archive. The official shadcn CLI configuration endpoint was unavailable; Button/Card were installed from official source with the MIT notice retained.
