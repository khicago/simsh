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

### Error Handling
- Projection failures SHOULD surface as explicit errors or absent paths, not silent partial files.
- If partial materialization is unavoidable, the partial state MUST be detectable by the caller.

### Source vs Derived Data
- Mirrored source artifacts belong in source-oriented namespaces such as `/knowledge_base`.
- Agent-authored or transformed artifacts belong in `/task_outputs`.
- Adapters SHOULD NOT blur those categories to save plumbing work.

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

## Contract Tests
Minimum adapter contract tests SHOULD cover:
- stable path projection for the same logical source object;
- explicit freshness/error behavior;
- session checkpoint and resume;
- trace-driven observation or follow-up behavior;
- separation of mirrored source data from derived outputs.

## Promotion Rule
Kernel abstractions that only look good in isolated unit tests SHOULD remain provisional.

Promote an adapter-facing abstraction to stable contract status only after:
- it is exercised by a real adapter-backed workload;
- the consumer behavior is documented;
- the failure modes at the seam are understood and tested.
