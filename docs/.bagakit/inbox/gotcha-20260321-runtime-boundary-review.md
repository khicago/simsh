---
title: Runtime boundary review: symlink escape and HTTP override risks
kind: gotcha
status: inbox
tags:
  - gotcha
sources:
  - pkg/adapter/agentfs/filespace_core.go
  - pkg/service/httpapi/execute_endpoint.go
  - cmd/simsh-cli/main.go
created: 2026-03-21T15:25:00Z
---

## Candidate
Context:
- 2026-03-21 repo review focused on default runtime boundaries, HTTP service behavior, and whether tests match the architecture docs.

Gotchas:
- `pkg/adapter/agentfs/filespace_core.go` only checks lexical path containment in `toHostPath()`. It does not resolve symlinks before read/write/list operations.
- Repro: create a symlink under `task_outputs/` pointing outside the repo, then run `go run ./cmd/simsh-cli -policy full -c 'echo exploit | tee /task_outputs/escape_link/pwned.txt'`. The file is created outside the repo root.
- `pkg/adapter/localfs/adapter.go` still has a nested-parent escape case. If a path enters a symlink inside root and then creates deeper nonexistent components, `resolveAndCheckPath()` can return early on `EvalSymlinks(parent)` failure and allow writes outside root.
- Repro: create `<root>/task_outputs/escape -> <outside>`, then write to `<root>/task_outputs/escape/subdir/pwned.txt` through `localfs.NewOps(...).WriteFile(...)`. The file lands in `<outside>/subdir/pwned.txt`.
- `pkg/service/httpapi/execute_endpoint.go` allows per-request `host_root`, `profile`, and `policy` overrides for one-shot execution, and `cmd/simsh-cli/main.go` exposes that handler directly via `serve`.
- Repro: start `simsh-cli serve -root <A> -policy read-only`; then POST `/v1/execute` with `{"host_root":"<B>","policy":"full","command":"echo injected | tee /task_outputs/http.txt"}`. The write lands under `<B>/task_outputs/http.txt`.
- `ExternalCommandResult` only carries `stdout + exit_code`, so `/bin` extensions cannot preserve stderr separation or report trace-side path effects through the core contract.

Why it matters:
- This conflicts with the project docs that frame path access metadata and runtime boundaries as explicit, trustworthy safety surfaces for agents.
- It also weakens the stated goal that structured `ExecutionResult` and `ExecutionTrace` become the core SSOT for agent-visible behavior.
- The current test matrix is stronger in `engine` and `httpapi` happy paths than in default boundary enforcement; `pkg/adapter/agentfs` had no direct tests during this review.

Suggested follow-up:
- Harden `agentfs` with resolved-path checks equivalent to `localfs`.
- Fix `localfs.resolveAndCheckPath()` for nested nonexistent descendants under symlinked parents, then add regression tests for that exact shape.
- Decide whether HTTP `host_root/profile/policy` are trusted-control-plane inputs or should be disabled/ceiling-limited in server mode.
- Extend the external command seam so structured stderr and side-effect reporting are not lost at the `/bin` boundary.
- Add boundary regression tests before further adapter/session work.

## Promote To
- `docs/.bagakit/memory/gotcha-20260321-runtime-boundary-review.md` (curated), or
- `docs/<type>-<topic>.md` (normative/deep guide)
