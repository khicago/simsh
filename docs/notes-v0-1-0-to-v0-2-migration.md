---
title: Migration Plan: v0.1.0 to v0.2
required: false
sop:
  - Read this doc before upgrading integrations from the v0.1.0 release baseline to the planned v0.2 contract set.
  - Update this doc when the v0.2 feat order, scope, or compatibility strategy changes.
  - Regenerate `docs/must-sop.md` after SOP/frontmatter changes.
---

# Migration Plan: v0.1.0 to v0.2

## Context

`v0.1.0` is the first tagged release baseline for `simsh`.

The repo's earlier planning docs sometimes call this baseline "v1" in the sense of "first completed implementation scope". For release and migration work, treat the Git tag `v0.1.0` as the source of truth.

`v0.2` is planned as a contract upgrade release. It is not mainly about adding more shell syntax. The main change is that `simsh` will move from a one-shot runtime with text-first results to a session-aware runtime with structured results, execution traces, and explicit adapter lifecycle seams.

## Who Should Read This

- embedders calling `pkg/engine/runtime` or related runtime contracts
- HTTP clients using `/v1/execute`
- adapter authors projecting external systems into `simsh`
- maintainers preparing tests, rollout checks, and compatibility shims

## What Stays Stable Across the Upgrade

- `simsh` remains a deterministic command runtime, not a POSIX shell or container runtime.
- The default virtual roots remain `/knowledge_base`, `/task_outputs`, and `/temp_work`.
- Path metadata stays a first-class contract; v0.2 builds on it rather than replacing it.
- One-shot execution remains supported as an ephemeral execution path even after session support lands.

## What Changes in v0.2

The target v0.2 contract set is driven by the March 1 feat batch:

1. first-class session lifecycle and policy ceiling
2. structured `ExecutionResult` as the core result SSOT
3. structured `ExecutionTrace` side-effect tracking
4. adapter lifecycle and optional memory protocol hooks
5. one reference adapter-backed end-to-end validation path

This is a dependency chain, not five independent enhancements. Do not migrate consumers by feature popularity; migrate in the same order as the contract rollout.

## Migration Sequence

### Phase 0: Freeze the v0.1.0 Baseline

Before starting migration work:

- pin integrations to the `v0.1.0` tag
- record whether you currently depend on:
  - one-shot runtime construction per request
  - flat `output + exit_code` HTTP responses
  - audit logs instead of structured side-effect data
  - adapter-local lifecycle rules not encoded in `simsh` contracts

If you cannot list those assumptions explicitly, you are not ready to migrate safely.

### Phase 1: Prepare for Session Lifecycle

Target feat: `f-20260301-session-lifecycle-policy-ceiling`

Expected change:

- runtime calls gain a first-class session concept
- policy moves from "per-construction local choice" to "session ceiling + per-execution narrowing"
- one-shot execution becomes the compatibility path rather than the only path

Migration actions:

- stop assuming runtime state is rebuilt from scratch every call
- separate "session creation" from "execute one command"
- treat policy elevation as an explicit control-plane concern, not an inline execution toggle
- keep existing one-shot flows working until session-aware callers are ready

### Phase 2: Prepare for Structured Execution Results

Target feat: `f-20260301-structured-execution-result-contract`

Expected change:

- text-only command results stop being the core SSOT
- runtime and HTTP surfaces expose a structured result that carries stdout, stderr, timing, and trace fields
- CLI compatibility is preserved as a rendering concern

Migration actions:

- remove client logic that parses meaning out of a single flat output blob
- model `stdout`, `stderr`, `exit_code`, and future `trace` as separate fields in callers
- keep text rendering only at the outermost UI boundary

### Phase 3: Prepare for Execution Traces

Target feat: `f-20260301-execution-trace-side-effect-tracking`

Expected change:

- builtins and external commands report structured side effects
- read/write/deny path effects become directly consumable by planners and adapters
- audit hooks remain useful, but they are no longer the main machine-consumption path

Migration actions:

- replace audit-log scraping with trace-field consumption
- update tests to assert on requested/read/written/denied paths explicitly
- treat trace absence as a migration bug, not as a normal compatibility fallback

### Phase 4: Prepare Adapter Lifecycle Hooks

Target feat: `f-20260301-adapter-lifecycle-memory-protocol`

Expected change:

- adapters get explicit lifecycle hooks for create/resume/checkpoint/close
- optional `/memory` projections become adapter-managed lifecycle views, not ad hoc writable mounts
- trace consumption and projection freshness become part of the adapter contract

Migration actions:

- move adapter memory updates behind explicit lifecycle hooks
- make freshness, partial materialization, and failure states visible at the adapter boundary
- treat `/memory` as a managed projection surface if you expose it at all

### Phase 5: Gate on Reference Adapter Validation

Target feat: `f-20260301-reference-adapter-e2e-validation`

Expected change:

- at least one adapter-backed workflow proves the contract set end to end
- provisional seams become promotable only after that validation passes

Migration actions:

- do not declare your own integration "fully on v0.2" before the reference path is green
- reuse the reference adapter checks as the minimum rollout gate for downstream adopters

## Impact by Consumer Type

### CLI and TUI Users

- Lowest migration pressure.
- Keep relying on human-readable rendering.
- Be ready for optional session-aware flows and for richer diagnostics behind the same high-level commands.

### HTTP Clients

- Highest near-term migration pressure.
- Plan for a structured response contract instead of a text-first envelope.
- Separate transport parsing from business logic now, before v0.2 lands.

### Runtime Embedders

- Moderate to high migration pressure.
- Introduce a session manager boundary in your embedding layer.
- Stop passing policy as an unconstrained per-call override if you expect long-running sessions.

### Adapter Authors

- Highest total migration cost.
- You will need to implement lifecycle hooks, consume traces, and make projection behavior explicit.
- Delay product-specific abstractions until the reference adapter validation proves the seam.

## Recommended Compatibility Strategy

- Keep v0.1.0 callers working by adding v0.2 support behind compatibility shims first.
- Migrate transport and contract boundaries before moving product logic.
- Prefer additive rollout:
  - add session-aware APIs
  - add structured result consumption
  - add trace consumption
  - finally remove legacy assumptions

Do not invert this order by teaching adapters to depend on trace or session behavior that the runtime core has not stabilized yet.

## Release Checklist for v0.2

- session lifecycle and policy ceiling contract is implemented and tested
- structured result is the runtime SSOT
- trace fields cover read/write/deny/timing/resource cases
- adapter lifecycle hooks are explicit and documented
- one reference adapter-backed workload validates the seam end to end
- migration notes are updated from "planned" to "released behavior"

## Decision Rule

If an integration only needs deterministic one-shot execution, it can stay on the v0.1.0 compatibility path for longer.

If an integration needs resumable sessions, machine-readable side effects, or adapter-managed memory and projection behavior, it should plan for the full v0.2 migration path now rather than waiting for a late rewrite.
