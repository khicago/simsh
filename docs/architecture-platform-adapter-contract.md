---
title: Platform Adapter Contract
required: false
sop:
  - Read this doc before defining adapter lifecycles, RPC-to-file projections, or memory lifecycle protocols.
  - Keep adapter contracts explicit at the platform boundary; do not push product semantics into core by default.
  - Regenerate `docs/must-sop.md` after SOP/frontmatter changes.
---

# Platform Adapter Contract

## Context
Kernel contracts alone are not enough for a working agent harness. The platform layer is where external systems, domain memory, and runtime results meet.

In AgentOS-style systems, this is also the seam where the execution kernel stops and product/runtime orchestration begins.

If that seam stays implicit, two failure modes appear quickly:
- each adapter reinvents lifecycle and projection rules;
- kernel contracts look clean on paper but fail under real workloads because the consumption side is underspecified.

This document defines the platform-side contract that sits on top of the generic `simsh` runtime.

## Goals
- Standardize how platform adapters project external systems into `simsh`.
- Define explicit lifecycle hooks for optional session memory and adapter state.
- Make RPC-to-file projection and trace consumption testable.
- Require at least one reference implementation to validate the contract end-to-end.

## Non-Goals
- Defining one mandatory business domain for all adapters.
- Moving domain memory or product workflows into core runtime packages.
- Treating raw writable mounts as a substitute for explicit adapter control-plane logic.

## Adapter Responsibilities
Each non-trivial platform adapter SHOULD own four responsibilities:

### 1) Projection
Expose external systems through stable virtual filesystem paths with deterministic read/list/search/describe behavior.

### 2) Lifecycle
Participate in session create/resume/checkpoint/close flows when the adapter has persistent state or memory semantics.

### 3) Trace Consumption
Consume `ExecutionTrace` or `ExecutionResult` fields that matter for platform behavior, such as:
- indexing or refresh triggers;
- memory observation capture;
- audit and policy decisions;
- post-step planning hints.

### 4) Reference Validation
Ship contract tests and at least one end-to-end adapter-backed workload that proves the seam actually works.

## Recommended Namespace Use
- `/knowledge_base`: immutable or source-oriented mirrored artifacts.
- `/task_outputs`: durable derived artifacts produced by the agent workflow.
- `/temp_work`: transient scratch, caches, or disposable adapter intermediates.
- Optional `/memory`: adapter-projected memory view when the adapter implements the lifecycle protocol below.

`/memory` is an adapter contract choice, not a core runtime guarantee.

## RPC-to-File Projection Contract

### Path Identity
- Virtual paths MUST be canonical and stable for the same logical object.
- Adapters SHOULD avoid path shapes that depend on transient pagination or request ordering.

### Freshness
- Adapters MUST define whether projected content is live, cached, or snapshot-based.
- Staleness/freshness metadata SHOULD be surfaced explicitly, either via frontmatter, sidecar metadata, or adapter-specific describe fields.
- Adapters SHOULD prefer a small canonical freshness lifecycle over free-form labels. The current reference shape is:
  - `snapshot`: stable captured projection
  - `live`: currently refreshed projection
  - `stale`: known out-of-date projection awaiting refresh
  - `updated`: control-plane-authored projection that diverged from its prior snapshot
- Refresh or invalidation SHOULD happen through explicit adapter control-plane or lifecycle hooks, not through implicit writes to read-only projected mounts.
- If a refresh does not complete, callers should still be able to tell whether the last visible state is `stale`, `snapshot`, or a failed/absent projection.
- Freshness and materialization SHOULD stay separate. Freshness answers “how up to date is this projection?” while materialization answers “is this projection fully present, partial, or failed right now?”.

### Error Handling
- Projection failures SHOULD surface as explicit errors or absent paths, not silent partial files.
- If partial materialization is unavoidable, the partial state MUST be detectable by the caller.
- A minimal current reference shape is per-record machine-readable materialization metadata in namespace indexes and `/memory/projections.json`, with `failed` records remaining visible in metadata even if their file bodies are absent from the mounted tree.

### Source vs Derived Data
- Mirrored source artifacts belong in source-oriented namespaces such as `/knowledge_base`.
- Agent-authored or transformed artifacts belong in `/task_outputs`.
- Adapters SHOULD NOT blur those categories to save plumbing work.
- Skill projections under `/skills` SHOULD stay read-only and expose eligibility or precedence as metadata, not as implicit side effects on mount writability.
- If adapters expose skill selection, they SHOULD make the competition boundary and loser or winner reason explicit. A bare `selected` bit is acceptable only as a compatibility surface, not as the whole truth contract.
- A minimal current reference shape is: explicit adapter-defined selection scope on competing skills, derived `selection` provenance in namespace indexes and `/memory/projections.json`, and no path-derived fallback competition semantics.
- If adapters evolve skill entries over time, they SHOULD do so through explicit adapter-local control-plane APIs such as add/update/remove, with visibility taking effect on the next projection rebuild rather than through writable `/skills` mounts.
- If adapters audit control-plane mutations, they SHOULD do so through compact machine-readable event views with explicit visibility timing, not through free-form logs or inferred projection diffs.
- If adapters expose projection metrics or denial surfaces, those views SHOULD be derived from existing projection/audit/trace truth and MUST NOT invent unavailable semantics such as cache-hit ratios when no cache exists.

## Memory Lifecycle Protocol
Adapters that expose session memory SHOULD implement a standard lifecycle even if the internal storage differs.

### Conceptual Hooks
- `hydrate(session)`: prepare adapter memory state before execution begins.
- `observe(session, execution_result)`: append raw observations derived from execution trace/results.
- `checkpoint(session)`: persist mutable session memory state at safe checkpoints.
- `close(session)`: flush and release adapter resources.

### Design Rules
- Observation history SHOULD be append-only.
- Mutable belief or summary state SHOULD be checkpointed explicitly, not inferred from log replay on every call.
- Promotion, curation, and long-term indexing remain adapter control-plane responsibilities.
- Adapters SHOULD treat any `/memory` projection as a view over managed state, not as an ungoverned writable scratch mount.
- When `/memory` mirrors projection metadata, keep raw observations, projected indexes, curated summaries, and workflow views conceptually separate even if they share one mount namespace.
- `/memory` MAY expose freshness summaries and workflow status, but those views are evidence surfaces, not the source of truth for control-plane mutation.
- A minimal current reference shape for explicit curation is a structured read-only view such as `/memory/curated.json`, paired with an optional human-readable mirror such as `/memory/curated.md`, where each curated entry retains stable ids and source-path provenance.

## Trace Consumption Contract
- Adapters MUST document which trace fields they consume and why.
- If an adapter relies on write sets, denied paths, or duration budgets for planning, those dependencies SHOULD appear in adapter tests.
- Unused trace fields are acceptable; undocumented hidden dependencies are not.

## Reference Implementation Requirement
At least one adapter-backed workload SHOULD implement:
- session create/resume/checkpoint/close;
- RPC-to-file projection;
- optional memory lifecycle hooks;
- trace consumption that affects subsequent behavior.

The purpose is validation of the seam, not canonization of one business domain.

Current seam evidence intentionally spans more than one adapter shape:
- the richer `reference` adapter exercises `/knowledge_base`, `/resources`, `/skills`, managed `/memory`, control-plane hooks, audit, and metrics;
- the smaller `resourceset` adapter exercises the same lifecycle/projection seam with only `/resources` plus a minimal managed `/memory` view.

This is deliberate. A second adapter should be materially smaller, so seam validation does not silently overfit to the richest implementation.

## Contract Tests
Minimum adapter contract tests SHOULD cover:
- stable path projection for the same logical source object;
- deterministic `VirtualMount` list/search/describe behavior for projected paths;
- explicit freshness/error behavior;
- refresh or invalidation round-trips that preserve path identity while changing visible freshness state;
- session checkpoint and resume;
- trace-driven observation or follow-up behavior;
- separation of mirrored source data from derived outputs.

Once multiple adapter shapes exist, teams SHOULD factor the shared seam checks into a reusable conformance harness rather than copying benchmark-only smoke scenarios into each adapter package. The benchmark remains the end-to-end proof; the conformance harness carries the reusable lifecycle/projection invariants.

A good current shape is a small internal test helper such as `pkg/adapter/internal/contracttest` that sequences `create -> observe -> checkpoint -> resume -> close`, validates shared mount or opaque-state invariants, and leaves domain assertions to adapter-local callbacks. If the helper starts absorbing workflow, audit, selection, or other product-shaped assertions, it has crossed the seam and should be narrowed again.

When these helpers become reusable mechanisms rather than one-off test glue, they SHOULD keep an error-returning core and only use thin `testing.T` wrappers at the edge. That keeps failure semantics directly testable without forcing subprocess tricks or adapter-level indirection.

Mount conformance is now a sibling proof layer, not an afterthought inside lifecycle tests. A good current shape is a dedicated helper such as `pkg/adapter/internal/contracttest/mount.go` that validates `Exists`, `ListChildren`, `CollectFilesUnder`, `ResolveSearchPaths`, `DescribePath`, and read-only access/capability metadata against projected mount shapes. It should not become a generic filesystem DSL and it should not absorb benchmark or product semantics.

Once seam helpers are strong in isolation, the next proof layer SHOULD be composition/evolution stress validation: one harder workload that proves projection state, control-plane mutation, freshness/materialization, audit, metrics, denials, and checkpoint/resume still agree after multi-step evolution. This layer should reuse existing nouns and machine-readable surfaces rather than inventing new product concepts just to make the scenario broader.

## Promotion Rule
Kernel abstractions that only look good in isolated unit tests SHOULD remain provisional.

Promote an adapter-facing abstraction to stable contract status only after:
- it is exercised by a real adapter-backed workload;
- the consumer behavior is documented;
- the failure modes at the seam are understood and tested.
