---
title: Daily learnings (2026-03-24)
kind: howto
status: inbox
tags:
  - testing
  - shell
  - heredoc
  - redirection
sources:
  - pkg/engine/engine_test.go
  - pkg/engine/script_parser_v2.go
  - pkg/engine/script_runner.go
  - .bagakit/long-run/bk-execution-handoff.md
created: 2026-03-24T03:58:20Z
---

## Candidate

- Context:
  - This session closed the long-run row for heredoc, parser-error, and redirection regressions in the engine execute path.
  - The useful seam was the integrated `eng.Execute(...)` path in `pkg/engine/engine_test.go`, not new parser-only unit helpers.
- Learnings:
  - For this repo, heredoc regression tests should prove both parsing and runner behavior in one script:
    - heredoc success through `<<`
    - parser failures such as missing delimiter, unterminated heredoc, and unclosed quote
    - heredoc-fed `>` or `>>` output behavior
  - When validating redirection safety, pair one success case with one no-partial-side-effects case. A heredoc-fed write-limited failure is a compact way to prove the parser and runner still respect the atomicity contract together.
  - Exact parser error strings are stable enough to assert here because `runScript()` exposes them directly as `execute: <parser error>`. If those strings change intentionally, update the integrated regression tests in the same change.
  - A repo-wide `go test ./...` can fail for environment reasons when the host volume is almost full. Clearing the Go build cache with `go clean -cache -testcache` was enough to recover this gate on 2026-03-24 without changing code.
- Scope:
  - Reuse this approach for future shell-subset regressions in `pkg/engine`.
  - Do not treat it as a reason to broaden shell compatibility beyond the documented deterministic subset in `pkg/sh/runtime_env.go`.

## Promote To
- `docs/.bagakit/memory/howto-learning-20260324-parser-redirection.md` (curated), or
- `docs/<type>-<topic>.md` (normative/deep guide)
