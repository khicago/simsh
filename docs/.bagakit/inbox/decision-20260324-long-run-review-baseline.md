---
title: 2026-03-24 long-run review baseline
kind: decision
status: inbox
tags:
  - decision
  - long-run
  - quality-gates
  - tests
sources:
  - .bagakit/long-run/bk-execution-table.json
  - Makefile
  - pkg/adapter/agentfs/filespace_core.go
  - pkg/adapter/internal/pathguard/pathguard.go
  - pkg/engine/script_parser_v2.go
  - pkg/service/httpapi/execute_endpoint.go
created: 2026-03-23T19:04:46Z
---

## Candidate
- Context:
  - A repo-wide endless-expand review was run on 2026-03-24 to recover `long-run` from a no-next-step state.
  - `go test ./...` was green, but `make lint` and `make check` were red, so the default evidence gate was already broken.
- Decision:
  - The active backlog should start with restoring the lint/check gate before any new kernel feature work.
  - The next two kernel-first closures should target direct `agentfs/pathguard` coverage and parser or redirection regressions, because those areas are core runtime surfaces with weak direct coverage.
  - HTTP/CLI entry-surface hardening stays behind kernel-first fixes: important, but downstream from the kernel trust boundary.
- Rationale:
  - Staticcheck currently reports one bad test context, one vacuous assertion, and three unused helpers; as long as that stands, `make check` is not a trustworthy gate.
  - `pkg/adapter/agentfs` and `pkg/adapter/internal/pathguard` are the default workspace safety seam, yet both packages still show 0.0% package coverage.
  - `pkg/engine/script_parser_v2.go` supports heredoc parsing, but the current engine tests do not explicitly cover heredoc behavior or several parser failure branches.
  - `pkg/service/httpapi/execute_endpoint.go` still has unexecuted metadata-fallback code, and `pkg/service/service.go` has no direct tests.

## Promote To
- `docs/.bagakit/memory/decision-20260324-long-run-review-baseline.md` (curated), or
- `docs/notes-kernel-execution-backlog.md` if these priorities become the durable narrative outside long-run state
