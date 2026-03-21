---
title: Kernel Execution Backlog
required: false
sop:
  - Read this doc when choosing the next kernel execution item or converting review findings into implementation work.
  - Update this doc when kernel execution items are added, reprioritized, completed, or superseded.
  - Regenerate `docs/must-sop.md` after SOP/frontmatter changes.
---

# Kernel Execution Backlog

## Purpose

This document is the execution-layer companion to `docs/notes-kernel-optimization-plan.md`.

Use it as the current SSOT for:
- what kernel item should be executed next;
- why that item is being prioritized now;
- what files are expected to change;
- how the item will be validated;
- what "done" means;
- how to roll back or contain risk if the change goes wrong.

This document should stay concrete and work-item oriented.

## Item Template

Each item should include:
- `why now`
- `kernel invariant`
- `files to touch`
- `validation command`
- `done gate`
- `rollback note`

Optional but recommended:
- `status`
- `risk`
- `blocked by`
- `notes`

## Current Priorities

### K-001: Harden default filesystem boundary enforcement
- Status: todo
- Why now: P0 trust has to come first. Current review findings indicate that default runtime filesystem implementations still have path-escape edge cases, which makes every higher-level contract less trustworthy.
- Kernel invariant: path capability claims must match actual boundary enforcement; default filesystems must not allow escape writes or removes.
- Files to touch:
  - `pkg/adapter/agentfs/filespace_core.go`
  - `pkg/adapter/localfs/adapter.go`
  - `pkg/adapter/localfs/adapter_test.go`
  - `pkg/fs/filesystem_env_test.go`
- Validation command:
  - `go test ./pkg/adapter/localfs ./pkg/fs ./pkg/engine`
- Done gate:
  - Known symlink and nested-descendant escape flows are rejected in default filesystems.
  - Regression tests exist for both direct and nested-parent escape shapes.
  - No new failing engine/filesystem tests are introduced.
- Rollback note:
  - If stricter path checks break existing expected flows, revert only the new enforcement branch and keep the new regression tests for the failing edge case under a skipped or TODO-marked state until semantics are clarified.

### K-002: Define virtual `cwd` and path resolution model
- Status: todo
- Why now: After P0 boundary trust is hardened, agent ergonomics should improve through an explicit path resolution model. Absolute-path-only behavior is safe but too unnatural for long-running agent work and relative references.
- Kernel invariant: path resolution must be explicit, session-scoped, and capability-safe; path reachability must remain separate from path mutability.
- Files to touch:
  - `pkg/contract/integration_contract.go`
  - `pkg/contract/session_contract.go`
  - `pkg/engine/runtime/session_manager.go`
  - `pkg/builtin/op_command_lookup.go`
  - file-oriented builtins that currently require absolute-only operands
- Validation command:
  - `go test ./pkg/engine ./pkg/engine/runtime ./pkg/builtin`
- Done gate:
  - A session-local virtual `cwd` model is documented and implemented.
  - Relative paths resolve to normalized virtual absolute paths before capability checks.
  - `pwd` reflects virtual `cwd`, not only static root.
  - Mount-backed and synthetic paths remain capability-limited when reached through relative-path flows.
- Rollback note:
  - If relative-path semantics create ambiguity or regressions, keep the resolution layer behind an explicit feature boundary while preserving the documented model and tests.

### K-003: Improve mutation trace fidelity
- Status: todo
- Why now: Once path semantics are trustworthy, the next highest leverage is making `ExecutionTrace` accurate enough for planners and reviewers to consume without heuristics.
- Kernel invariant: traces must faithfully describe file mutations, denials, and resource summaries for core file operations.
- Files to touch:
  - `pkg/engine/trace_collector.go`
  - `pkg/engine/execution_result_test.go`
  - mutation-capable filesystem implementations as needed for accurate accounting
- Validation command:
  - `go test ./pkg/engine ./pkg/builtin`
- Done gate:
  - Bytes-written accounting for edit-heavy operations matches actual mutation behavior closely enough to be trustworthy.
  - Mutation-related trace tests cover write, append, edit, remove, and denial cases.
- Rollback note:
  - If precise accounting requires broader interface changes than expected, ship the smallest truthful improvement first and document any remaining approximation explicitly.

### K-004: Audit cancel/timeout effectiveness across execution and filesystem paths
- Status: todo
- Why now: Policy timeout exists in contracts today, but many filesystem paths still ignore `ctx`, which means timeout/cancel semantics may not yet be operationally meaningful.
- Kernel invariant: runtime interruption controls must have real effect on long-running execution paths.
- Files to touch:
  - `pkg/engine/execution_result.go`
  - `pkg/adapter/agentfs/filespace_core.go`
  - `pkg/adapter/localfs/adapter.go`
  - any related tests added for context-aware cancellation
- Validation command:
  - `go test ./pkg/engine ./pkg/adapter/localfs ./pkg/adapter/agentfs ./pkg/fs`
- Done gate:
  - Long-running filesystem/execution paths honor context cancellation where practical.
  - Timeout/cancel behavior is observable in tests and reflected in trace fields where appropriate.
- Rollback note:
  - If full context propagation is too invasive for one iteration, land the highest-risk path checks first and document remaining blind spots explicitly.

## Backlog Rules

- P0 items outrank convenience items by default.
- Do not start a broader action-surface expansion while known boundary-trust bugs remain.
- Prefer one finished, validated kernel item over multiple partially specified items.
- When a strategic priority changes, update `docs/notes-kernel-optimization-plan.md` first, then reflect the change here.
