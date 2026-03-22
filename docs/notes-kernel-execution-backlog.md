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

## Legacy Ft-Harness Note

The 2026-03-01 feat batch under `.bagakit/ft-harness/feats/` is currently treated as historical foundation work, not as the active execution backlog for the current kernel plan.

Current status:
- those feats are marked `done`;
- they remain unarchived because they predate the current `workspace_mode` field and are not safely archivable through the current harness scripts;
- they should not block creation or execution of the new kernel feats created from this backlog.

Follow-up policy:
- keep them as historical evidence for the v0.2 contract batch;
- do not delete or hand-edit their runtime state casually;
- handle schema/merge/archive cleanup in a separate harness-maintenance pass if needed.

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
- Feat: `f-20260321-default-filesystem-boundary-enforcement`
- Status: done
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
- Notes:
  - Default filesystems now use the same real-path guard for existing symlinks and missing descendants before mutation begins.
  - `CheckPathOp` is wired through default filesystem implementations, and multi-path mutation builtins preflight all operands before mutating.
  - Validated with `go test ./pkg/adapter/localfs ./pkg/fs ./pkg/engine` and `go test ./...`.
- Rollback note:
  - If stricter path checks break existing expected flows, revert only the new enforcement branch and keep the new regression tests for the failing edge case under a skipped or TODO-marked state until semantics are clarified.

### K-002: Define virtual `cwd` and path resolution model
- Feat: `f-20260321-virtual-cwd-path-resolution`
- Status: done
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
- Notes:
  - Added an engine-level path resolution layer that keeps `RootDir` as the filesystem root but tracks `WorkingDir` as mutable session state.
  - Added `cd`, updated `pwd`/default directory commands to honor session-local `cwd`, and preserved mount/synthetic capability limits through relative-path flows.
  - Validated with `go test ./pkg/builtin ./pkg/engine ./pkg/engine/runtime` and `go test ./...`.
- Rollback note:
  - If relative-path semantics create ambiguity or regressions, keep the resolution layer behind an explicit feature boundary while preserving the documented model and tests.

### K-003: Improve mutation trace fidelity
- Feat: `f-20260321-mutation-trace-fidelity`
- Status: done
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
- Notes:
  - `T-001` fixed edit-byte accounting so full-file rewrite semantics are reflected in trace resource summaries.
  - `T-002` preserved external-command `stderr` in `ExecutionResult` and added dedicated external stdout/stderr byte counters to `ExecutionTrace`, instead of collapsing everything into stdout or file-read counters.
  - Validated with `go test ./pkg/engine ./pkg/builtin`, `go test ./pkg/engine ./pkg/service/httpapi`, and `go test ./...`.
- Rollback note:
  - If precise accounting requires broader interface changes than expected, ship the smallest truthful improvement first and document any remaining approximation explicitly.

### K-004: Audit cancel/timeout effectiveness across execution and filesystem paths
- Feat: `f-20260321-cancel-timeout-effectiveness`
- Status: done
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
- Notes:
  - Default filesystem operations now short-circuit on canceled contexts instead of ignoring `ctx`.
  - Engine statement/pipeline/redirection loops now stop promptly when `ctx` is already canceled or timed out, and `ExecutionTrace` exposes `canceled/timed_out` in regression tests.
  - Validated with `go test ./pkg/adapter/localfs ./pkg/fs ./pkg/engine` and `go test ./...`.
- Rollback note:
  - If full context propagation is too invasive for one iteration, land the highest-risk path checks first and document remaining blind spots explicitly.

### K-005: Build simsh-native reference validation and metric gates
- Feat: `f-20260322-simsh-native-reference-validation-metric-gates`
- Status: done
- Why now: The kernel core has now crossed the boundary from contract-hardening into proof-of-usefulness. The next question is not whether the primitives exist, but whether they make real agent file workflows measurably better than heavier alternatives.
- Kernel invariant: reference validation must prove real agent leverage, not only interface cleanliness; benchmark output must report explicit pass/fail against metric gates.
- Files to touch:
  - `docs/notes-kernel-optimization-plan.md`
  - benchmark or validation artifacts under a project-local non-product path
  - any metric gate definitions or benchmark runners needed for repeatable evaluation
- Validation command:
  - benchmark command(s) added by the feat, plus `go test ./...`
- Done gate:
  - A small `simsh`-native benchmark covers relative navigation, inspect/edit/write loops, mount/synthetic boundaries, trace-consumable planning, and cancel/timeout scenarios.
  - First-pass metric gates are encoded with explicit thresholds for trace completeness, session success, reviewable patch latency, and async completion success.
  - At least one reference workload produces a baseline report with threshold pass/fail output rather than raw numbers only.
- Notes:
  - Added a committed benchmark runner under `benchmarks/simsh_native_reference/` plus a checked-in baseline report under `benchmarks/simsh_native_reference/reports/`.
  - The first-pass suite currently covers relative navigation, inspect/edit/write loops, mount boundaries, trace-planning, and cancel/timeout scenarios.
  - The first baseline passes all configured gates and is validated by `go test ./benchmarks/simsh_native_reference` and `go test ./...`.
- Rollback note:
  - If the first benchmark cut is too broad, keep the thresholds and scoring schema stable, then narrow the workload set rather than weakening the gates.

### K-006: Harden builtin ACI contracts and machine-friendly output modes
- Feat: `f-20260322-builtin-aci-dual-readable-query-tooling`
- Status: in_progress
- Why now: The kernel now has stronger path semantics, trace fidelity, and a baseline reference benchmark. The next agent-leverage gap is the builtin surface itself: many default commands are still text-first and syntax-first even though `simsh` is meant to be an agent-native runtime.
- Kernel invariant: builtin commands and manuals should minimize parse cost, confirmation cost, failure-attribution cost, and token cost for agent callers; command summaries should be derived from explicit contracts rather than prose inference.
- Design bias:
  - default outputs should stay readable by humans and agents at the same time;
  - optimize for signal-to-noise, not for machine-only serialization;
  - preserve efficient pipeline composition where a command is already naturally pipe-friendly;
  - strengthen structure-aware query tools so agents can read only the needed part of structured files instead of repeatedly dumping whole files.
- Design input:
  - `docs/notes-requirements.md`
  - `docs/notes-builtin-aci-review.md`
- Files to touch:
  - `pkg/engine/builtin_catalog.go`
  - `pkg/contract/runtime_types.go`
  - `pkg/builtin/op_help_manual.go`
  - `pkg/builtin/commands/*/manual.md`
  - selected builtin implementations under `pkg/builtin/`
- Validation command:
  - `go test ./pkg/builtin ./pkg/engine`
- Done gate:
  - Builtin metadata exposes explicit ACI contract fields for summary rendering.
  - `man` and `man --list` can surface machine-relevant command facts without relying on prose-only hints.
  - High-value inspection commands gain machine-friendly output modes where the review identified clear parse-cost wins (`tree`, `grep`, `find`, `wc`, `type`, `env`).
  - Default text outputs remain dual-readable and token-efficient instead of collapsing into machine-only JSON by default.
  - Commands that naturally expose structured records support an explicit structured mode such as `--json` or an existing `--fmt json` family.
  - Structured modes are additive and documented, rather than silently replacing pipe-friendly default text behavior.
  - The builtin surface includes at least one stronger structure-aware query step beyond generic text dumping, using `frontmatter` as the design reference for future structure parsers.
  - The first structure-aware follow-up includes stronger JSON processing and narrower local search/query flows.
  - Mutation commands provide an explicit low-noise confirmation mode instead of forcing post-hoc verification in text-only harnesses.
- Notes:
  - The design review recommends treating `ls -l` and `frontmatter stat` as the reusable pattern for future output contracts: compact default text plus explicit `--fmt` machine formats.
  - The same review also recommends investing in structure-aware query tools, not only more serialized output modes, so agents can write structured files and then query just the relevant subset.
  - The requirements baseline sharpens this further: default outputs stay dual-readable, structured output is explicit and opt-in, pipe-friendly commands keep their composition value, and JSON/local-search tooling should improve in the same wave.
  - The first implementation wave should prioritize contract metadata, the worst current parse-cost offenders, and the first JSON/local-search query upgrades before broader command polish.
- Rollback note:
  - If a default-format change causes excessive compatibility risk, keep the new machine formats and metadata while leaving the old text form behind a compatibility mode; do not roll back the explicit contract layer.

## Backlog Rules

- P0 items outrank convenience items by default.
- Do not start a broader action-surface expansion while known boundary-trust bugs remain.
- Prefer one finished, validated kernel item over multiple partially specified items.
- When a strategic priority changes, update `docs/notes-kernel-optimization-plan.md` first, then reflect the change here.
