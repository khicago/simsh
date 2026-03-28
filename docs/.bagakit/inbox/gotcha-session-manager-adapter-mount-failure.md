---
title: Failed session adapter phases must not clear stored mounts
kind: gotcha
status: inbox
tags:
  - gotcha
  - runtime
  - sessions
  - adapters
sources:
  - pkg/engine/runtime/session_manager.go
  - pkg/engine/runtime/session_manager_test.go
  - pkg/engine/runtime/runtime_core_additional_test.go
created: 2026-03-24T10:47:49Z
---

## Candidate
Context:
- During the 2026-03-24 long-run item `manual-20260324-session-manager-adapter-failure-edges`, new regression tests targeted failed `observe`, `checkpoint`, and `close` adapter phases in `SessionManager`.

Gotcha:
- Multi-value assignment like `nextSession, current.adapterMounts, err = applySessionAdapters(...)` mutates `current.adapterMounts` even when `err != nil`.
- In the pre-fix code, failed adapter phases returned an error but still cleared stored adapter mounts to `nil`.
- That partial mutation was mostly invisible on the immediate failing call because `current.runtime` still held the old mounts.
- The bug becomes user-visible on the next runtime rebuild path, such as a narrower-policy execute, because rebuild uses `record.adapterMounts` and can lose `/memory` or other adapter projections.

Fix shape:
- Stage adapter mounts in a local variable first.
- Assign `current.adapterMounts` only after the adapter hook succeeds.
- In `Execute`, also delay the assignment until the runtime rebuild succeeds, so adapter failures or rebuild failures leave prior state intact.

Evidence:
- `go test ./pkg/engine/runtime -run 'TestSessionManager|TestApplySessionAdapters|TestInvokeAdapterPhase'`
- `go test ./pkg/engine/runtime ./pkg/adapter/reference`
- `go test ./pkg/engine/runtime -cover` -> `93.8%` (up from `87.1%`)

## Promote To
- `docs/.bagakit/memory/gotcha-session-manager-adapter-mount-failure.md` (curated), or
- `docs/<type>-<topic>.md` (normative/deep guide)
