---
title: Runtime boundary review: symlink escape and HTTP override risks
kind: gotcha
confidence: medium
tags:
  - gotcha
  - boundaries
  - review
sources:
  - pkg/adapter/agentfs/filespace_core.go
  - pkg/service/httpapi/execute_endpoint.go
  - cmd/simsh-cli/main.go
created: 2026-03-21T15:25:00Z
updated: 2026-03-23T00:00:00Z
---

## Context
The 2026-03-21 runtime review focused on boundary truth: whether path access metadata, default filesystem behavior, and entry-surface defaults actually matched the trust story told by the docs.

## Gotchas
- `pkg/adapter/agentfs/filespace_core.go` only checks lexical path containment in `toHostPath()`. It does not resolve symlinks before read/write/list operations.
- Repro: create a symlink under `task_outputs/` pointing outside the repo, then run `go run ./cmd/simsh-cli -policy full -c 'echo exploit | tee /task_outputs/escape_link/pwned.txt'`. The file is created outside the repo root.
- `pkg/adapter/localfs/adapter.go` still has a nested-parent escape case. If a path enters a symlink inside root and then creates deeper nonexistent components, `resolveAndCheckPath()` can return early on `EvalSymlinks(parent)` failure and allow writes outside root.
- Repro: create `<root>/task_outputs/escape -> <outside>`, then write to `<root>/task_outputs/escape/subdir/pwned.txt` through `localfs.NewOps(...).WriteFile(...)`. The file lands in `<outside>/subdir/pwned.txt`.
- `pkg/service/httpapi/execute_endpoint.go` allows per-request `host_root`, `profile`, and `policy` overrides for one-shot execution, and `cmd/simsh-cli/main.go` exposes that handler directly via `serve`.
- Repro: start `simsh-cli serve -root <A> -policy read-only`; then POST `/v1/execute` with `{"host_root":"<B>","policy":"full","command":"echo injected | tee /task_outputs/http.txt"}`. The write lands under `<B>/task_outputs/http.txt`.
- `ExternalCommandResult` only carried `stdout + exit_code`, so `/bin` extensions could not preserve stderr separation or report trace-side path effects through the core contract.

## Why It Matters
- This conflicts with the project docs that frame path access metadata and runtime boundaries as explicit, trustworthy safety surfaces for agents.
- It also weakens the stated goal that structured `ExecutionResult` and `ExecutionTrace` become the core SSOT for agent-visible behavior.
- The current test matrix is stronger in `engine` and `httpapi` happy paths than in default boundary enforcement; `pkg/adapter/agentfs` had no direct tests during this review.

## Outcome
- The default-filesystem escape issues were later used to drive `f-20260321-default-filesystem-boundary-enforcement` and are now treated as a closed kernel hardening gap.
- The entry-adapter HTTP override concern remains useful as a boundary-design reminder, but it is not the current kernel-first priority.
- The durable value of this note is the review checklist: lexical containment is not enough, preflight and mutation paths must agree, and entry-surface defaults should not be mistaken for trusted ceilings.

## Practical Checklist
- Lexical path containment is not enough; real-path and descendant handling must agree with policy claims.
- Preflight and mutation paths should reject the same boundary violations.
- Entry-surface defaults should not be mistaken for trusted ceilings in architecture discussions.
- If a trace or result seam drops key structure, the contract is probably not yet truthful enough for agent-facing use.
