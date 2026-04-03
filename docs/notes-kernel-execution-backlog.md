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
- Status: done
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
  - The first implementation wave prioritized contract metadata, the worst parse-cost offenders, structured mutation confirmation, and the first JSON/local-search query upgrades before broader command polish.
  - The feat is archived under `.bagakit/ft-harness/feats-archived/f-20260322-builtin-aci-dual-readable-query-tooling/`.
- Rollback note:
  - If a default-format change causes excessive compatibility risk, keep the new machine formats and metadata while leaving the old text form behind a compatibility mode; do not roll back the explicit contract layer.

### K-007: Harden effective-core default workspace behavior
- Feat: `f-20260323-effective-core-default-workspace-hardening`
- Status: done
- Why now: Strict core contracts are now relatively mature, but the agent actually experiences `simsh` through the default workspace surface. The remaining leverage is in `engine + builtin + default mounts` behavior, especially where pipe composability, structured modes, and summary/confirmation contracts can still drift without regressions.
- Kernel invariant: the default workspace must stay dual-readable, pipe-aware, and failure-explicit; engine-level command dispatch and builtin output contracts must not regress silently.
- Files to touch:
  - `pkg/engine/engine_test.go`
  - `pkg/builtin/coverage_test.go`
  - selected default-workspace docs if output contracts or priorities need clarification
- Validation command:
  - `go test ./pkg/engine ./pkg/builtin`
- Done gate:
  - High-value default workspace behaviors have regression tests for both normal and failure paths.
  - Pipe-friendly commands keep their composition semantics while structured modes remain explicit and additive.
  - Default inspection and confirmation surfaces do not require agent callers to recover meaning from ambiguous text.
- Notes:
  - Prioritize `find -exec`, `man --list`, mutation confirmation modes, and any other engine-level seams where default ACI regressions would be expensive for agents.
  - Prefer tests that exercise the integrated default workspace over isolated helper coverage.
  - The feat is archived under `.bagakit/ft-harness/feats-archived/f-20260323-effective-core-default-workspace-hardening/`.
- Rollback note:
  - If a change starts widening the command surface instead of hardening it, stop and keep only the regression coverage or documentation tightening.

### K-008: Validate the adapter seam with a committed reference workload
- Feat: `f-20260323-adapter-backed-workload-validation`
- Status: done
- Why now: The adapter contract is already documented and lightly tested, but the next promotion gate is stronger evidence that a real adapter-backed workload survives session lifecycle, projection updates, and managed `/memory` views without special-casing core semantics.
- Kernel invariant: adapter-backed projections and memory views must remain deterministic, lifecycle-aware, and visibly downstream from kernel contracts.
- Files to touch:
  - `benchmarks/simsh_native_reference/`
  - `pkg/adapter/reference/adapter_test.go`
  - adapter-architecture docs only if the validation contract itself changes
- Validation command:
  - `go test ./pkg/adapter/reference ./benchmarks/simsh_native_reference ./pkg/engine/runtime`
- Done gate:
  - At least one committed reference workload exercises create/observe/checkpoint/resume/close with adapter-projected `/knowledge_base/reference` and managed `/memory`.
  - The workload asserts both business-visible behavior and the trace/session evidence it depends on.
  - The benchmark/reference suite treats adapter validation as a first-class scenario, not an implicit side-effect of unrelated tests.
- Notes:
  - This should build on the existing reference adapter, not invent a heavier product layer.
  - Keep `/memory` as an adapter-managed view, not a writable scratch escape hatch.
  - The feat is archived under `.bagakit/ft-harness/feats-archived/f-20260323-adapter-backed-workload-validation/`.
- Rollback note:
  - If the benchmark-level workload proves too broad, keep the scenario contract and move the heaviest assertions down into dedicated adapter/runtime tests rather than dropping adapter validation entirely.

### K-009: Make adapter-side projections and managed memory views more realistic
- Feat: `f-20260323-realistic-adapter-projections-managed-memory-views`
- Status: done
- Why now: The seam is now validated, so the next step is to make at least one adapter behave more like a realistic non-core harness layer. The reference adapter should move beyond a single mirrored document tree and a flat observations log toward richer resource projection, managed `/memory` views, and adapter-backed workflow state.
- Kernel invariant: richer adapter behavior must stay downstream from core contracts; `/memory` remains a managed read-only view, and resource projection remains explicit and deterministic.
- Files to touch:
  - `pkg/adapter/reference/adapter.go`
  - `pkg/adapter/reference/adapter_test.go`
  - `benchmarks/simsh_native_reference/`
- Validation command:
  - `go test ./pkg/adapter/reference ./benchmarks/simsh_native_reference ./pkg/engine/runtime`
- Done gate:
  - The reference adapter can project both source-oriented documents and separate resource trees through stable virtual paths.
  - `/memory` exposes richer managed views than a flat log, including workflow/status-level material useful to an agent or harness.
  - Adapter-backed workflow state advances through trace consumption without turning `/memory` into an uncontrolled write path.
  - The richer projection model is covered by adapter tests and at least one committed benchmark/reference scenario.
- Notes:
  - Keep this implementation realistic but still generic enough to serve as a seam validator rather than a product-specific framework.
  - Prefer explicit projected files and workflow summaries over hidden state or imperative side channels.
  - The reference adapter now projects `/resources`, richer `/memory` workflow views, and adapter-derived workflow state; the feat is archived under `.bagakit/ft-harness/feats-archived/f-20260323-realistic-adapter-projections-managed-memory-views/`.
- Rollback note:
  - If the richer view starts to leak product semantics into core-facing tests, keep the projection model and trim the domain-specific presentation rather than reverting the managed-memory idea itself.

### K-010: Add projection metadata and a minimal adapter control plane
- Feat: `f-20260323-adapter-projection-metadata-control-plane`
- Status: done
- Why now: Once the reference adapter can project multiple namespaces and managed workflow views, the next realism gap is missing metadata and explicit update seams. Projected objects should expose source/freshness metadata, and adapter-side updates should go through explicit control-plane methods rather than implicit mutation assumptions.
- Kernel invariant: adapter metadata and control-plane behavior must remain downstream from core contracts; metadata should surface through deterministic projected files, not hidden runtime channels.
- Files to touch:
  - `pkg/adapter/reference/adapter.go`
  - `pkg/adapter/reference/adapter_test.go`
  - `benchmarks/simsh_native_reference/`
- Validation command:
  - `go test ./pkg/adapter/reference ./benchmarks/simsh_native_reference ./pkg/engine/runtime`
- Done gate:
  - Projected documents and resources expose explicit source/freshness metadata via stable sidecars or summary views.
  - The reference adapter exposes a minimal control-plane API for upserting projected documents, resources, and workflows.
  - Managed `/memory` views include projection metadata summaries that an agent or harness can read without special adapter knowledge.
  - Adapter tests and the benchmark/reference suite cover the metadata and control-plane path.
- Notes:
  - Keep the control plane adapter-local; do not add new core-runtime contracts for this.
  - Prefer read-only projected metadata files over embedding mutable semantics in path writeability.
  - The feat is archived under `.bagakit/ft-harness/feats-archived/f-20260323-adapter-projection-metadata-control-plane/`.
- Rollback note:
  - If metadata surfacing becomes too noisy, keep the sidecar/index files and simplify their shape rather than dropping projection metadata entirely.

### K-011: Close runtime truth gaps found by post-implementation audit
- Feat: `f-20260323-runtime-audit-follow-up-hardening`
- Status: done
- Why now: Recent audit findings highlighted three remaining trust gaps: output redirection atomicity under write limits, canonical metadata lookup in the reference adapter constructor, and benchmark success semantics that could still count business-only success without full evidence.
- Kernel invariant: runtime, adapter, and benchmark contracts must agree on truth; no partial side effects, no dual naming of the same projected object, and no success metrics that ignore missing evidence.
- Files to touch:
  - `pkg/engine/script_runner.go`
  - `pkg/engine/engine_test.go`
  - `pkg/adapter/reference/adapter.go`
  - `pkg/adapter/reference/adapter_helpers_test.go`
  - `benchmarks/simsh_native_reference/`
- Validation command:
  - `go test ./pkg/engine ./pkg/adapter/reference ./benchmarks/simsh_native_reference`
  - `go test ./...`
- Done gate:
  - Multi-output redirection does not leave partial filesystem side effects when the last payload violates write limits.
  - Reference adapter metadata binds to canonical normalized names, not raw input keys.
  - Benchmark session/async success requires evidence-complete scenarios whenever trace or assertion checks are defined.
- Notes:
  - Prefer global fixes over case-by-case patches: payload-dependent redirection preflight, canonical metadata maps, and one benchmark success helper.
  - The feat is archived under `.bagakit/ft-harness/feats-archived/f-20260323-runtime-audit-follow-up-hardening/`.
- Rollback note:
  - If any fix changes intended contract semantics, update the benchmark/tests/docs together rather than keeping mismatched behavior and evidence.

### K-012: Add a deterministic projection freshness lifecycle and refresh policy
- Status: done
- Why now: The reference adapter now exposes multiple projected namespaces and metadata, but freshness is still mostly a free-form label instead of a lifecycle. That leaves a realism gap exactly where the adapter contract says projected content must distinguish live, cached, snapshot, stale, or partial/error materialization explicitly.
- Kernel invariant: adapter-side freshness and materialization state must be explicit, deterministic, and machine-visible; refresh or invalidation must not rely on hidden runtime channels.
- Files to touch:
  - `pkg/adapter/reference/adapter.go`
  - `pkg/adapter/reference/adapter_test.go`
  - `pkg/adapter/reference/adapter_helpers_test.go`
  - `benchmarks/simsh_native_reference/suite.go`
  - `benchmarks/simsh_native_reference/README.md`
- Validation command:
  - `go test ./pkg/adapter/reference ./benchmarks/simsh_native_reference ./pkg/engine/runtime`
  - `go test ./...`
- Done gate:
  - Projected documents and resources move through a deterministic freshness/materialization model instead of only carrying free-form freshness strings.
  - Refresh or invalidate flows update sidecars, indexes, and managed `/memory` summaries consistently.
  - The reference workload covers at least one stale-to-refreshed or invalidated-to-refreshed round-trip without special-casing core semantics.
- Notes:
  - Keep the lifecycle adapter-local; do not add new core-runtime contracts for this.
  - Prefer state that is legible in `/memory/projections.json`, `_index.json`, or equivalent sidecars over hidden control-plane-only semantics.
  - The reference adapter now exposes canonical freshness states plus explicit `Refresh*` / `Invalidate*` control-plane methods.
  - Adapter tests and the reference benchmark now cover a stale-to-refreshed round-trip through session close/resume, and managed `/memory/summary.md` exposes projection-freshness counts.
- Rollback note:
  - If a richer lifecycle becomes too noisy, keep canonical state names plus refresh hooks and simplify the rendered summaries instead of reverting to opaque free-form labels.

### K-013: Audit and refine the adapter-side freshness and managed-memory contract
- Status: done
- Why now: Once freshness becomes a real lifecycle, the highest-value review is to verify that the implementation still stays adapter-local, evidence-complete, and aligned with the documented `/memory` contract before another feature wave builds on top of it.
- Kernel invariant: business-visible adapter success must still require evidence-complete freshness and memory semantics; `/memory` remains a managed read-only view rather than an implicit writable control plane.
- Files to touch:
  - `pkg/adapter/reference/adapter_test.go`
  - `benchmarks/simsh_native_reference/suite.go`
  - `docs/architecture-memory-skills-extension.md`
  - `docs/architecture-platform-adapter-contract.md`
  - `docs/notes-kernel-execution-backlog.md`
- Validation command:
  - `go test ./pkg/adapter/reference ./benchmarks/simsh_native_reference ./pkg/engine/runtime`
  - `go test ./...`
- Done gate:
  - Review produces either explicit no-findings or a tightly bounded follow-up finding list.
  - Benchmark and adapter tests assert freshness/materialization evidence, not only business-visible output.
  - Docs clearly state refresh ownership, managed-memory curation boundaries, and the allowed relationship between workflow state and trace-derived evidence.
- Notes:
  - This is intentionally a review/refine loop, not another broad implementation wave.
  - If the review uncovers contract drift, tighten the contract before widening the adapter surface.
  - The evidence-review slice is complete: the reference benchmark now decodes projection views structurally and covers stale-to-refreshed projection state.
  - The doc/backlog refinement is now complete: the written adapter contract names the freshness lifecycle explicitly and keeps refresh ownership in the adapter control plane.
  - `docs/architecture-memory-skills-extension.md` now makes managed `/memory` boundaries explicit so future adapter work does not blur raw observations, projection indexes, curated summaries, and workflow views.
- Rollback note:
  - If the review suggests the current slice is too broad, keep the finished freshness lifecycle and reduce the next implementation scope rather than weakening evidence requirements.

### K-014: Implement explicit managed-memory curation and workflow transitions
- Status: done
- Why now: The reference adapter already proves projection and trace consumption, but managed `/memory` still behaves mainly like a derived report over trace sets. The next realism step is one explicit curation layer plus workflow transitions that are adapter-owned rather than purely heuristic.
- Kernel invariant: `/memory` remains a managed read-only view over adapter-owned state; raw observations, curated summaries, and workflow transitions must stay explicit and auditable.
- Files to touch:
  - `pkg/adapter/reference/adapter.go`
  - `pkg/adapter/reference/adapter_test.go`
  - `pkg/adapter/reference/adapter_helpers_test.go`
  - `benchmarks/simsh_native_reference/suite.go`
  - `docs/architecture-memory-skills-extension.md`
  - `docs/architecture-platform-adapter-contract.md`
- Validation command:
  - `go test ./pkg/adapter/reference ./benchmarks/simsh_native_reference ./pkg/engine/runtime`
  - `go test ./...`
- Done gate:
  - `/memory` exposes curated read-only promotion views that are distinct from raw projection indexes and raw observation logs.
  - Workflow state advances through explicit adapter-local transition rules rather than ad hoc trace heuristics alone.
  - The reference workload covers at least one non-happy-path transition such as blocked-to-resolved-to-completed across checkpoint or resume.
- Notes:
  - Keep curation and transition semantics adapter-local and deterministic.
  - Prefer one small realistic managed-memory layer over a large pseudo-product framework.
  - The reference adapter now exposes explicit curated `/memory` views (`/memory/curated.json` and `/memory/curated.md`) driven by an adapter-local control-plane rather than trace-derived implicit summaries.
  - Curated entries persist through adapter opaque state and remain distinct from raw observations, projection indexes, and workflow views.
  - The reference benchmark now proves that curated entries are structurally readable, provenance-bearing, and survive checkpoint/resume without widening `/memory` into a writable backdoor.
- Rollback note:
  - If explicit workflow transitions prove too product-specific, keep the curated memory views and narrow transitions to the smallest generic state machine that still matches tests and docs.

### K-015: Add a `/skills` projection driver with explicit eligibility metadata
- Status: done
- Why now: After freshness and managed-memory semantics become trustworthy, the biggest remaining adapter-side namespace gap is `/skills`. The architecture already expects a skill projection seam, but the reference adapter does not yet exercise it.
- Kernel invariant: skill projection remains a read-only adapter concern; eligibility, precedence, and source metadata must be surfaced explicitly rather than by silent omission.
- Files to touch:
  - `pkg/adapter/reference/adapter.go`
  - `pkg/adapter/reference/adapter_test.go`
  - `benchmarks/simsh_native_reference/suite.go`
  - `docs/architecture-memory-skills-extension.md`
- Validation command:
  - `go test ./pkg/adapter/reference ./benchmarks/simsh_native_reference ./pkg/engine/runtime`
  - `go test ./...`
- Done gate:
  - The reference adapter can project a stable read-only `/skills` namespace with index or metadata views.
  - Skill entries expose source, freshness, and eligibility metadata in a deterministic machine-readable shape.
  - The reference workload includes at least one skill-backed workflow that proves selection or non-selection without pushing skill logic into core packages.
- Notes:
  - This is a next-phase candidate after the current freshness and managed-memory loop closes cleanly.
  - Do not let `/skills` widen into a product-specific registry before freshness and workflow evidence are already solid.
  - The reference adapter now projects a read-only `/skills` namespace with canonical paths, `_index.json`, and `/memory/projections.json` visibility.
  - Skill entries now surface `source`, canonical `freshness`, explicit eligibility state/reason, comparable precedence metadata, and a simple `selected` bit without introducing a mutable skill registry.
  - The reference benchmark now proves one selected eligible skill and one visible-but-ineligible fallback skill, and keeps `/skills` read-only under session lifecycle flows.
- Rollback note:
  - If `/skills` is still too early, preserve the contract wording and move the item to the next phase rather than smuggling skill semantics into unrelated adapter work.

### K-016: Make per-item projection materialization and failure state visible
- Status: done
- Why now: The reference adapter already exposes freshness, workflow provenance, curated views, and `/skills`, but it still needed one more truth surface: a caller should be able to tell whether a specific projection is fully materialized, partial, or failed without confusing that state with freshness.
- Kernel invariant: freshness and materialization remain separate truths at the adapter boundary; item-level projection failure must be machine-visible without forcing a new core-runtime contract.
- Files to touch:
  - `pkg/adapter/reference/adapter.go`
  - `pkg/adapter/reference/adapter_helpers_test.go`
  - `pkg/adapter/reference/adapter_test.go`
  - `benchmarks/simsh_native_reference/suite.go`
  - `benchmarks/simsh_native_reference/README.md`
  - `docs/architecture-platform-adapter-contract.md`
- Validation command:
  - `go test ./pkg/adapter/reference ./pkg/engine/runtime ./benchmarks/simsh_native_reference`
  - `go test ./...`
  - `make lint`
  - `make check`
- Done gate:
  - Per-item projection records expose machine-readable materialization state such as `materialized`, `partial`, or `failed`, plus detail.
  - Failed projections remain visible in metadata/index surfaces even when their mounted file bodies are absent.
  - Benchmark and adapter tests structurally assert partial/error truth instead of relying on message-only checks.
- Notes:
  - The reference adapter now keeps global `SetProjectionError(...)` for whole-projection failure, but also exposes item-level materialization truth for documents and resources.
  - `invalidate` now degrades fully materialized documents/resources into explicit partial state with reason rather than silently implying incompleteness through freshness alone.
  - The reference benchmark now decodes `materialization` from `/memory/projections.json` and requires partial/error state to carry machine-readable detail.
- Rollback note:
  - If item-level materialization semantics prove too broad, keep the metadata surface and reduce the number of states rather than falling back to freshness-only ambiguity.

### K-017: Make `/skills` selection and precedence truth explicit
- Feat: `f-20260401-skills-selection-precedence-truth`
- Status: done
- Why now: The reference adapter already projects `/skills` with explicit freshness, eligibility, and precedence metadata, but `selected` is still too close to adapter input. The next adapter-side gap is to make skill selection a derived, explainable truth surface rather than a hand-authored bit.
- Kernel invariant: skill selection stays adapter-local and read-only; competition boundaries must be explicit; winner or loser state must be derived from documented inputs rather than path heuristics or mutable mount behavior.
- Files to touch:
  - `pkg/adapter/reference/adapter.go`
  - `pkg/adapter/reference/adapter_test.go`
  - `pkg/adapter/reference/adapter_helpers_test.go`
  - `benchmarks/simsh_native_reference/suite.go`
  - `docs/architecture-memory-skills-extension.md`
  - `docs/architecture-platform-adapter-contract.md`
  - `.bagakit/long-run/bk-execution-handoff.md`
- Validation command:
  - `go test ./pkg/adapter/reference ./benchmarks/simsh_native_reference ./pkg/engine/runtime`
  - `go test ./...`
- Done gate:
  - `/skills` records expose machine-readable selection provenance in addition to a compatibility `selected` bit.
  - Selection is derived from explicit eligibility and precedence inputs within an explicit adapter-defined scope.
  - Non-selected skills remain visible and carry a machine-readable explanation for why they were not selected.
  - Benchmark and adapter tests prove winner, loser, and ineligible cases structurally rather than by prose-only assertions.
- Notes:
  - Keep the scope narrow: do not introduce a mutable skill registry, remote sync, or product-specific orchestration.
  - Reject path-derived implicit competition groups; if skills compete, the adapter should declare that scope explicitly.
  - Preserve the current read-only `/skills` namespace and keep selection semantics downstream from core contracts.
  - The reference adapter now derives selection in one SSOT path from explicit `SelectionScope + Eligibility + Precedence`, surfaces compatibility `selected` plus machine-readable `selection` provenance (`state/mode/scope/reason/winner_path`), and keeps unscoped skills out of implicit path-based competition.
  - Adapter tests now cover winner, loser, ineligible, unscoped, and deterministic tie-break cases; the reference benchmark structurally asserts selection provenance instead of relying on prose-only output.
  - Validated with `go test ./pkg/adapter/reference -count=1`, `go test ./benchmarks/simsh_native_reference -count=1`, `go test ./...`, `make lint`, and `make check`.
  - The feat is archived under `.bagakit/ft-harness/feats-archived/f-20260401-skills-selection-precedence-truth/`.
- Rollback note:
  - If the selection truth surface grows too opinionated, keep explicit scope and reason metadata while narrowing the number of derived states rather than falling back to a naked `selected` flag.

### K-018: Add a minimal explicit `/skills` control plane
- Feat: `f-20260401-skills-control-plane-lifecycle`
- Status: done
- Why now: The reference adapter now projects `/skills` with explicit selection truth, but it still lacks the minimal control-plane seam that the architecture already expects for skills. Without explicit add/update/remove APIs, skill evolution remains artificially static and the only way to change skill state is to rebuild adapter seed input wholesale.
- Kernel invariant: `/skills` remains a read-only projection; skill evolution stays adapter-local; selection truth remains derived from one SSOT after control-plane changes.
- Files to touch:
  - `pkg/adapter/reference/adapter.go`
  - `pkg/adapter/reference/adapter_helpers_test.go`
  - `pkg/adapter/reference/adapter_test.go`
  - `benchmarks/simsh_native_reference/suite.go`
  - `docs/architecture-memory-skills-extension.md`
  - `docs/architecture-platform-adapter-contract.md`
  - `.bagakit/long-run/bk-execution-handoff.md`
- Validation command:
  - `go test ./pkg/adapter/reference ./benchmarks/simsh_native_reference ./pkg/engine/runtime`
  - `go test ./...`
  - `make lint`
  - `make check`
- Done gate:
  - The reference adapter exposes minimal explicit APIs to add, update, and remove skills without making `/skills` writable.
  - `/skills/_index.json` and `/memory/projections.json` reflect control-plane changes deterministically.
  - Selection truth recomputes from the same adapter-local derivation path after skill lifecycle changes.
  - Benchmark and adapter tests prove at least one add-or-update case and one remove-or-fallback case structurally.
- Notes:
  - Keep the slice narrow: no remote sync, marketplace, or product registry semantics.
  - Reuse the existing selection truth surface; do not create a second path for post-update selection decisions.
  - Prefer one coherent skill control plane over ad hoc mutation helpers spread across tests.
  - The reference adapter now exposes explicit `UpsertSkill`, `UpdateSkill`, and `RemoveSkill` APIs while keeping `/skills` itself read-only.
  - Skill add/update/remove changes become visible on the next projection rebuild or resume, and reselection still flows through the same adapter-local derivation path used by static skills.
  - Adapter tests now cover add, update, remove, and fallback restoration across lifecycle boundaries; the reference benchmark structurally asserts control-plane skill addition, update, reselection, and removal.
  - Validated with `go test ./pkg/adapter/reference -count=1`, `go test ./benchmarks/simsh_native_reference -count=1`, `go test ./...`, `make lint`, and `make check`.
- Rollback note:
  - If the control plane starts widening into registry semantics, keep the minimal add/update/remove seam and drop the product-shaped pieces rather than reverting to static-only skills.

### K-019: Add control-plane observability and audit for `/skills`
- Feat: `f-20260402-skills-control-plane-observability-audit`
- Status: done
- Why now: The reference adapter now has real skill control-plane APIs, but it still lacks an explicit audit surface for what changed, when it becomes visible, and how reselection moved. Without that, future adapters and harnesses will have to infer control-plane truth indirectly from projection diffs.
- Kernel invariant: control-plane observability stays adapter-local; `/skills` remains read-only; audit truth and projection truth must stay aligned.
- Files to touch:
  - `pkg/adapter/reference/adapter.go`
  - `pkg/adapter/reference/adapter_helpers_test.go`
  - `pkg/adapter/reference/adapter_test.go`
  - `benchmarks/simsh_native_reference/suite.go`
  - `benchmarks/simsh_native_reference/README.md`
  - `docs/architecture-memory-skills-extension.md`
  - `docs/architecture-platform-adapter-contract.md`
  - `.bagakit/long-run/bk-execution-handoff.md`
- Validation command:
  - `go test ./pkg/adapter/reference ./benchmarks/simsh_native_reference ./pkg/engine/runtime`
  - `go test ./...`
  - `make lint`
  - `make check`
- Done gate:
  - The reference adapter exposes machine-readable audit events for skill add/update/remove flows.
  - Audit events make visibility timing explicit instead of forcing callers to guess from mount diffs.
  - `/memory` exposes a compact human-readable summary and a structured audit view.
  - Benchmark and adapter tests prove audit truth and projection truth stay aligned after control-plane changes.
- Notes:
  - Keep the slice narrow: do not build a generic event bus or product audit platform.
  - Prefer one compact event surface and one compact summary surface over multiple overlapping logs.
  - Keep the audit vocabulary machine-readable and stable.
  - The reference adapter now emits compact machine-readable skill control-plane audit events and mirrors them in `/memory/skills_audit.json` plus `/memory/skills_audit.md`.
  - Audit makes visibility timing explicit through projection generation and `visible_from_generation`, while projection truth remains derived from the existing skill selection SSOT.
  - Adapter tests and the native reference benchmark now prove add/update/remove audit alignment, reselection movement, and `/skills` read-only enforcement together.
  - Validated with `go test ./pkg/adapter/reference -count=1`, `go test ./benchmarks/simsh_native_reference -count=1`, `go test ./...`, `make lint`, and `make check`.
- Rollback note:
  - If the audit surface becomes too broad, keep the event schema and trim fields or views rather than falling back to prose-only logging.

### K-020: Add adapter projection metrics and denial surfaces
- Feat: `f-20260402-adapter-projection-metrics-denial-surfaces`
- Status: done
- Why now: Stage D still has an unfinished gap. The reference adapter now has explicit control-plane audit, but callers still lack a compact machine-readable metrics surface for projection generations and a dedicated denial or policy surface beyond raw path lists.
- Kernel invariant: observability stays truthful and adapter-local; projection metrics do not invent cache semantics that do not exist; denial or policy views stay aligned with actual runtime truth.
- Files to touch:
  - `pkg/adapter/reference/adapter.go`
  - `pkg/adapter/reference/adapter_helpers_test.go`
  - `pkg/adapter/reference/adapter_test.go`
  - `benchmarks/simsh_native_reference/suite.go`
  - `benchmarks/simsh_native_reference/README.md`
  - `docs/architecture-memory-skills-extension.md`
  - `docs/architecture-platform-adapter-contract.md`
  - `.bagakit/long-run/bk-execution-handoff.md`
- Validation command:
  - `go test ./pkg/adapter/reference ./benchmarks/simsh_native_reference ./pkg/engine/runtime`
  - `go test ./...`
  - `make lint`
  - `make check`
- Done gate:
  - The reference adapter exposes a compact machine-readable projection metrics surface.
  - The reference adapter exposes a compact denial or policy surface under `/memory`.
  - Metrics and denial views stay aligned with projection generation, control-plane audit, and observed denied paths.
  - No fake cache-hit metric is introduced while the adapter has no real cache.
  - Benchmark and adapter tests prove the surfaces structurally instead of relying on prose or diff heuristics.
- Notes:
  - Keep the slice narrow: no generic observability platform and no premature caching layer.
  - Prefer compact counts and recent samples over large duplicated snapshots.
  - If latency is exposed, keep it best-effort and clearly scoped to projection rebuilds rather than pretending to be end-to-end system latency.
  - The reference adapter now exposes `/memory/projection_metrics.json|md` and `/memory/denials.json|md` as compact observability surfaces.
  - Metrics are derived from projection generation, projection counts, freshness/materialization counts, control-plane event count, and unique denied paths; cache metrics remain explicitly unavailable until a real cache exists.
  - Denials are classified by adapter-visible namespace (`reference`, `resources`, `skills`, `memory`, `external_or_unknown`) rather than by guessed semantic cause.
  - Adapter tests and the native reference benchmark now prove metrics and denial views stay aligned with projection truth, control-plane audit, and denied paths.
  - Validated with `go test ./pkg/adapter/reference -count=1`, `go test ./benchmarks/simsh_native_reference -count=1`, `go test ./...`, `make lint`, and `make check`.
- Rollback note:
  - If the metrics surface starts widening into speculative telemetry, keep the truthful counts and generation fields and drop the speculative fields.

### K-021: Validate the seam with a second adapter shape
- Feat: `f-20260403-second-adapter-seam-validation`
- Status: done
- Why now: The current seam is strong, but most of its evidence still flows through one rich reference adapter. The next meaningful validation step is a second, smaller adapter shape that proves the seam does not implicitly require `/skills`, workflow control planes, or richer `/memory` features.
- Kernel invariant: adapter seam contracts must remain generic enough that a smaller adapter can participate in the same lifecycle/projection model without forcing reference-specific semantics into core or into every adapter.
- Files to touch:
  - `pkg/adapter/<new-adapter>/`
  - `pkg/adapter/<new-adapter>/*_test.go`
  - `benchmarks/simsh_native_reference/suite.go`
  - `benchmarks/simsh_native_reference/README.md`
  - `docs/architecture-platform-adapter-contract.md`
  - `.bagakit/long-run/bk-execution-handoff.md`
- Validation command:
  - `go test ./pkg/adapter/... ./benchmarks/simsh_native_reference ./pkg/engine/runtime`
  - `go test ./...`
  - `make lint`
  - `make check`
- Done gate:
  - A second adapter package exists and implements the same adapter seam with a materially smaller feature shape.
  - The second adapter participates in session lifecycle and projection without inheriting unnecessary `reference`-specific semantics.
  - The native reference benchmark (or a companion workload inside it) proves the second adapter-backed flow structurally.
  - The second adapter improves confidence that current seam contracts are generic rather than overfit to `reference`.
- Notes:
  - Favor a resource-first or document-first adapter with a simpler `/memory` view.
  - The goal is seam generality, not feature parity with `reference`.
  - Keep the adapter small enough that any extra required abstraction pressure becomes obvious.
  - The new `resourceset` adapter now validates the seam with a materially smaller shape: `/resources` plus minimal managed `/memory`, no skills, no workflows, no curation, and no control-plane semantics.
  - The native reference benchmark now includes a separate companion scenario for the smaller adapter instead of overloading the rich reference scenario with capability branches.
  - Validated with `go test ./pkg/adapter/resourceset ./benchmarks/simsh_native_reference -count=1`, `go test ./...`, `make lint`, and `make check`.
- Rollback note:
  - If the second adapter starts turning into a second rich product model, shrink it back to the smallest seam-proving workload instead of deleting the effort.

### K-022: Extract a shared adapter seam conformance harness
- Feat: `f-20260403-adapter-seam-conformance-harness`
- Status: done
- Why now: `reference` and `resourceset` now prove the seam through materially different shapes, but the core lifecycle/projection invariants still live in adapter-specific tests and benchmark code. The next hardening step is to make those shared invariants reusable so future adapters validate the seam without copying benchmark-specific logic.
- Kernel invariant: seam conformance must stay generic and reusable; the shared harness should validate lifecycle, projection, opaque-state, and managed-memory invariants without pulling richer product semantics into every adapter test.
- Files to touch:
  - `pkg/adapter/<shared-test-package>/`
  - `pkg/adapter/reference/*_test.go`
  - `pkg/adapter/resourceset/*_test.go`
  - `docs/architecture-platform-adapter-contract.md`
  - `docs/notes-kernel-execution-backlog.md`
  - `.bagakit/long-run/bk-execution-handoff.md`
- Validation command:
  - `go test ./pkg/adapter/reference ./pkg/adapter/resourceset -count=1`
  - `go test ./...`
  - `make lint`
  - `make check`
- Done gate:
  - A shared adapter conformance harness exists and is small enough to encode seam invariants rather than adapter-specific product logic.
  - `reference` and `resourceset` both use the harness for the common lifecycle/projection checks.
  - Existing richer adapter-specific tests remain in place for product-shaped behavior; the harness does not replace them.
  - Docs clearly distinguish reusable conformance coverage from benchmark-level end-to-end validation.
- Notes:
  - Favor explicit callbacks and small expectations over generic reflection-heavy test frameworks.
  - Do not move benchmark assertions into the conformance harness.
  - Keep adapter-specific helper logic close to each adapter when it is not truly shared.
  - The shared conformance helper now lives in `pkg/adapter/internal/contracttest` and owns only the reusable lifecycle sequence, mount-presence checks, opaque-state round-trip, and managed-memory visibility helpers.
  - `reference` and `resourceset` now each carry one focused conformance test that uses the helper, while the richer reference end-to-end test and the native reference benchmark remain separate proof layers.
  - Validated with `go test ./pkg/adapter/reference ./pkg/adapter/resourceset -count=1`, `go test ./...`, `make lint`, and `make check`.
  - The feat is archived under `.bagakit/ft-harness/feats-archived/f-20260403-adapter-seam-conformance-harness/`.
- Rollback note:
  - If the harness starts widening into a bespoke testing DSL, keep only the reusable lifecycle/projection checks and return richer behavior checks to adapter-local tests.

### K-023: Extract a shared adapter mount conformance harness
- Feat: `f-20260403-adapter-mount-conformance-harness`
- Status: done
- Why now: `K-022` made lifecycle/projection conformance reusable, but mount-level list/search/describe/read-only metadata checks still live in scattered adapter assertions. The next seam-hardening step is a focused reusable proof for `VirtualMount` behavior across adapter shapes.
- Kernel invariant: mount conformance must stay generic and mount-focused; the shared helper should validate deterministic list/search/describe/read-only metadata semantics without absorbing adapter-specific workflow, skill, audit, or benchmark behavior.
- Files to touch:
  - `pkg/adapter/internal/contracttest/`
  - `pkg/adapter/reference/*_test.go`
  - `pkg/adapter/resourceset/*_test.go`
  - `docs/architecture-platform-adapter-contract.md`
  - `docs/notes-kernel-execution-backlog.md`
  - `.bagakit/long-run/bk-execution-handoff.md`
- Validation command:
  - `go test ./pkg/adapter/reference ./pkg/adapter/resourceset -count=1`
  - `go test ./...`
  - `make lint`
  - `make check`
- Done gate:
  - A shared mount conformance helper exists and stays narrow enough to encode `VirtualMount` invariants rather than adapter-specific semantics.
  - `reference` and `resourceset` both use the helper for deterministic list/search/describe/read-only metadata checks on their projected mounts.
  - Existing benchmark scenarios and richer adapter tests remain in place; the helper does not replace them.
  - Docs clearly distinguish lifecycle conformance, mount conformance, and benchmark validation as separate proof layers.
- Notes:
  - Favor explicit expectations and callbacks over reflective helper magic.
  - Reuse `pkg/mount` unit-test knowledge, but do not couple the helper to the concrete static-mount implementation.
  - The helper should prove read-only metadata and deterministic path surfacing, not runtime write denial flows.
  - The shared mount helper now lives in `pkg/adapter/internal/contracttest/mount.go` and owns only `VirtualMount` invariants: `Exists`, list/search, `DescribePath`, and read-only capability truth.
  - `reference` and `resourceset` now each use the helper for focused mount conformance, while lifecycle conformance and benchmark validation remain separate proof layers.
  - Validated with `go test ./pkg/adapter/reference ./pkg/adapter/resourceset -count=1`, `go test ./...`, `make lint`, and `make check`.
  - The feat is archived under `.bagakit/ft-harness/feats-archived/f-20260403-adapter-mount-conformance-harness/`.
- Rollback note:
  - If the helper starts becoming a generic filesystem test DSL, keep only the reusable `VirtualMount` invariants and move richer assertions back into adapter-local tests.

### K-024: Add direct tests for shared contracttest helpers
- Feat: `f-20260403-contracttest-helper-self-coverage`
- Status: done
- Why now: `K-022` and `K-023` turned adapter seam proof into reusable helpers, but `pkg/adapter/internal/contracttest` itself still lacks direct package-local tests. The next hardening step is to make that reusable layer self-tested, especially around failure semantics, instead of trusting only indirect adapter coverage.
- Kernel invariant: reusable proof helpers must stay smaller than the runtime and adapter logic they validate; direct tests should cover helper semantics without recreating a hidden runtime or adapter product layer.
- Files to touch:
  - `pkg/adapter/internal/contracttest/*`
  - `docs/notes-kernel-execution-backlog.md`
  - `docs/notes-reusable-items-coding.md`
  - `.bagakit/long-run/bk-execution-handoff.md`
- Validation command:
  - `go test -cover ./pkg/adapter/internal/contracttest ./pkg/adapter/reference ./pkg/adapter/resourceset`
  - `go test ./...`
  - `make lint`
  - `make check`
- Done gate:
  - `pkg/adapter/internal/contracttest` has direct tests for lifecycle and mount helpers.
  - Success paths and key failure semantics are both covered.
  - The direct tests raise the package coverage materially, targeting full or near-full helper coverage without reward hacking.
  - Docs clearly record that reusable proof helpers are self-tested assets, not only adapter side effects.
- Notes:
  - Prefer small fake adapters and fake mounts over broad test scaffolding.
  - Do not duplicate benchmark scenarios in package-local helper tests.
  - Failure cases should be chosen for seam value, not for synthetic branch farming.
  - `pkg/adapter/internal/contracttest` now has direct package-local tests for both lifecycle and mount helpers.
  - The helper layer now follows an `error`-returning core plus thin `testing.T` wrappers so failure semantics are directly testable.
  - `go test -cover ./pkg/adapter/internal/contracttest` now reports `92.1%` coverage, up from `0.0%`, while `reference` and `resourceset` remain above `90%`.
  - The feat is archived under `.bagakit/ft-harness/feats-archived/f-20260403-contracttest-helper-self-coverage/`.
- Rollback note:
  - If helper tests start reproducing full adapter behavior, pull them back to targeted helper semantics and keep end-to-end behavior in adapter tests or benchmarks.

### K-025: Validate adapter composition and evolution truth under stress
- Feat: `f-20260403-adapter-composition-evolution-stress-validation`
- Status: done
- Why now: the seam, mount, and helper layers are now individually strong, but the remaining risk is composition drift. Projection, control-plane mutation, freshness/materialization, audit, metrics, denials, and checkpoint/resume must also stay coherent when exercised together in one evolving workload.
- Kernel invariant: adapter truth surfaces must remain aligned under multi-step evolution; composition validation should prove coherence, not introduce new product semantics.
- Files to touch:
  - `pkg/adapter/reference/*_test.go`
  - `benchmarks/simsh_native_reference/*`
  - `docs/notes-kernel-execution-backlog.md`
  - `docs/architecture-platform-adapter-contract.md`
  - `.bagakit/long-run/bk-execution-handoff.md`
- Validation command:
  - `go test ./pkg/adapter/reference ./benchmarks/simsh_native_reference -count=1`
  - `go test ./...`
  - `make lint`
  - `make check`
- Done gate:
  - At least one multi-step composition/evolution workload proves projection, control-plane, freshness/materialization, audit, metrics, and denials stay structurally aligned.
  - Checkpoint/resume preserves the relevant truth surfaces across the workload.
  - The validation lives as proof, not as a new product feature branch.
  - Docs clearly record composition/evolution stress validation as a proof layer above isolated helper conformance.
- Notes:
  - Prefer one hard scenario over many shallow ones.
  - Do not add new adapter nouns just to make the scenario interesting.
  - Keep assertions structural and machine-readable; do not fall back to prose-only output matching.
  - `reference` now has a focused composition/evolution alignment stress test and a separate workflow-override composition roundtrip test so adapter-local proof stays readable.
  - The native reference benchmark now carries an explicit `adapter_composition_evolution_stress` scenario that validates the same truth surfaces at the benchmark layer.
  - Validated with `go test ./pkg/adapter/reference ./benchmarks/simsh_native_reference -count=1`, `go test ./...`, `make lint`, and `make check`.
- Rollback note:
  - If the stress workload starts smuggling in new product semantics, strip it back to composition proof and keep product expansion as a separate feat.

### K-026: Research comparable runtimes and benchmark fit for the next wave
- Feat: `f-20260403-runtime-comparables-benchmark-fit-research`
- Status: done
- Why now: the current adapter-seam proof wave is closed, so the next investment choice became a strategy question rather than an implementation-debt question. Before opening another build wave, `simsh` needed a narrow comparison against directly relevant runtimes and benchmark families.
- Kernel invariant: research must stay decision-oriented and pressure-tested against current `simsh` scope; it must not become an open-ended survey or a hidden product-expansion wave.
- Files to touch:
  - `task_outputs/research/*`
  - `docs/notes-kernel-execution-backlog.md`
  - `.bagakit/long-run/bk-execution-handoff.md`
- Validation command:
  - review the resulting research brief for a concrete next-feat recommendation, explicit rejected alternatives, and explicit non-goals
- Done gate:
  - The research output compares directly relevant runtime implementations.
  - The research output compares benchmark families by fit to current `simsh` scope.
  - The output ends with one recommended next feat and a short rejected-alternatives section.
  - The slice leaves behind a decision-quality artifact rather than only link collection.
- Notes:
  - The formal output is `task_outputs/research/runtime-comparables-benchmark-fit-study-2026-04-04.md`.
  - The strongest conclusion is to start with a benchmark-mapping / evaluation-feasibility wave rather than another raw feature wave.
  - Terminal-Bench is the closest external benchmark family for current `simsh`; SWE-bench-Live is the strongest dynamic-workload reference; EnvBench/ResearchEnvBench stay boundary references.
- Rollback note:
  - If the research starts broadening into a generic field survey, cut it back to runtime fit, benchmark fit, and one next-feat recommendation only.

### K-027: Map simsh to external benchmark families and evaluation feasibility
- Feat: `f-20260403-external-benchmark-mapping-evaluation-feasibility`
- Status: done
- Why now: `K-026` concluded that the highest-value next move is not another primitive or adapter noun, but a benchmark-mapping / evaluation-feasibility wave. `simsh` now needs an explicit answer for what can be evaluated against Terminal-Bench or adjacent families as-is, what needs translation, and what should be intentionally excluded.
- Kernel invariant: external benchmark mapping should reuse existing `simsh` truth surfaces and benchmark assets; it must not weaken native gates or push the runtime into full environment synthesis.
- Proposed scope:
  - Map current `simsh`-native benchmark scenarios to one or two external benchmark families.
  - Define what `simsh` can evaluate faithfully, what requires translation, and what is out of scope.
  - Produce one or two lightweight evaluation adapters or mapping artifacts, not full benchmark adoption.
- Files to touch:
  - `benchmarks/simsh_native_reference/*`
  - `benchmarks/internal/scenarios/*`
  - `benchmarks/external_mapping/*`
  - `task_outputs/research/*`
  - `docs/notes-kernel-execution-backlog.md`
  - `docs/architecture-platform-adapter-contract.md`
  - `docs/.bagakit/memory/*`
- Validation command:
  - `go test ./benchmarks/external_mapping ./benchmarks/simsh_native_reference -count=1`
  - `go test ./...`
- Done gate:
  - A checked-in machine-readable scenario inventory captures every native benchmark scenario with stable ids and truth-surface summaries.
  - Checked-in mapping artifacts exist for Terminal-Bench and SWE-bench-Live and classify every native scenario as `as_is`, `translated`, or `excluded`.
  - Guardrail tests fail if native scenario ids drift without mapping artifact updates.
  - Docs and research output explain the mapping layer as an evaluation artifact, not as benchmark adoption or product expansion.
- Notes:
  - Keep Terminal-Bench as the primary external comparison family and SWE-bench-Live as the dynamic-workload reference.
  - Native benchmark scenarios remain the primary SSOT; translation stays in the mapping layer.
  - `benchmarks/internal/scenarios/catalog.go` now keeps stable native scenario ids/categories separate from curated evaluation inventory metadata.
  - `benchmarks/external_mapping/scenario_inventory.json` records the checked-in evaluation inventory and explicitly distinguishes canonical identity fields from curated task-shape/truth-surface metadata.
  - `benchmarks/external_mapping/terminal_bench_mapping.json` currently lands at 1 `as_is`, 4 `translated`, and 4 `excluded`.
  - `benchmarks/external_mapping/swe_bench_live_mapping.json` currently lands at 4 `translated` and 5 `excluded`.
  - Guardrails now fail if native benchmark scenario ids drift, if mapping files lose scenario coverage, or if mapping-family headers/status fields drift.
  - Validated with `go test ./benchmarks/external_mapping ./benchmarks/simsh_native_reference -count=1` and `go test ./...`.
- Non-goals:
  - Do not adopt an external benchmark wholesale.
  - Do not add new product nouns or a third adapter just to chase benchmark fit.
  - Do not widen `simsh` toward environment synthesis.
- Rollback note:
  - If the mapping work starts mutating native scenarios to look more benchmark-compatible, pull that behavior back out and keep the slice limited to inventory, mapping, and evaluation-feasibility artifacts.

### K-028: Prototype a lightweight Terminal-Bench comparison layer
- Status: proposed
- Why now: `K-027` showed that Terminal-Bench is the only near-term family with a meaningful direct fit, but most relevant native scenarios still require translation. The next highest-value move is a small comparison prototype, not full benchmark adoption.
- Kernel invariant: any external comparison layer must stay downstream from the native benchmark SSOT and must not require changing runtime semantics or widening `simsh` toward environment synthesis.
- Proposed scope:
  - Build one small comparison/export layer around the strongest current fit, starting with `inspect_edit_write_loop`.
  - Optionally include one translated Terminal-Bench-aligned slice to prove the translation approach without broadening the runtime.
  - Emit a compact comparison artifact or report, not a full external benchmark harness.
- Non-goals:
  - Do not adopt Terminal-Bench wholesale.
  - Do not map SWE-bench-Live end to end.
  - Do not add new runtime or adapter nouns just to improve external coverage.

## Backlog Rules

- P0 items outrank convenience items by default.
- Do not start a broader action-surface expansion while known boundary-trust bugs remain.
- Prefer one finished, validated kernel item over multiple partially specified items.
- When a strategic priority changes, update `docs/notes-kernel-optimization-plan.md` first, then reflect the change here.
