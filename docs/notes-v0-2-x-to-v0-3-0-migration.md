---
title: Migration Guide: v0.2.x to v0.3.0
required: false
sop:
  - Read this doc before upgrading integrations from the v0.2.x release line to the planned v0.3.0 line.
  - Update this doc when the v0.3.0 scope, compatibility strategy, or release gates change.
  - Regenerate `docs/must-sop.md` after SOP/frontmatter changes.
---

# Migration Guide: v0.2.x to v0.3.0

## Context

`v0.2.x` established the core runtime contract line for `simsh`:

- session lifecycle
- structured execution results
- execution traces
- adapter-backed seam validation

`v0.3.0` should not be treated as another shell-surface expansion release.
It is the line where `simsh` should read clearly as an agent sandbox kernel for harnesses, memory-aware runtimes, and AgentOS-style systems.

That means the migration pressure is not mainly "learn more commands".
It is:

- tighten execution-substrate assumptions
- tighten mount and projection contracts
- treat agent efficiency evidence as a first-class release input

## Who Should Read This

- harness authors embedding `simsh` as an execution substrate
- AgentOS-style platform teams using `simsh` beneath planning, review, or workflow layers
- adapter authors exposing memory, skills, resources, or RPC-backed projections
- maintainers preparing the `v0.3.0` release cut and rollout notes

## What Stays Stable

- `simsh` remains a deterministic command runtime, not a POSIX shell and not a container runtime
- the virtual root model remains explicit
- session, trace, and adapter seams remain core runtime truth
- one-shot execution stays supported
- core packages remain `pkg/contract`, `pkg/sh`, `pkg/fs`, and `pkg/engine/runtime`

## What Changes In v0.3.0

### 1. Project Positioning Becomes Clearer

`v0.2.x` already shipped the core contract chain, but `v0.3.0` should make the project legible in one sentence:

`simsh` is one of the most important infrastructure layers beneath a harness or AgentOS stack: the sandbox kernel that makes execution, path semantics, session state, and side effects reliable enough to build on.

This is partly a docs change, but not only a docs change.
It affects how integrations should think about ownership boundaries.

### 2. Mount Contracts Become More Operational

`v0.2.x` already had adapter and mount seams.
`v0.3.0` should make capability-first mount dispatch part of the expected integration contract, especially for remote-backed or high-latency mounts.

The important migration pressure is:

- stop treating all mounts as transparent local directories
- declare truth/materialization/write/latency/consistency semantics explicitly
- provide listing, bulk-read, search, and mutation capabilities where the workload requires them
- fail closed or narrow scope instead of silently degrading into per-file RPC fanout

Primary reference:
- `docs/architecture-high-performance-mount-system.md`

### 3. Builtin ACI And Query Tooling Matter More

`v0.3.0` should assume that builtin quality is part of kernel quality.
The important changes are:

- stronger structure-aware JSON querying
- `rg` as the agent-oriented search front door
- more explicit command contracts and structured output modes
- lower confirmation cost for mutation flows

This is not a call to become shell-complete.
It is a call to reduce agent parse cost and wasted model work.

### 4. Benchmark Evidence Moves Closer To The Release Gate

`v0.2.x` proved the contract chain.
`v0.3.0` should also prove that the kernel is materially useful for agent execution.

Current proof layers:

- native reference validation
- Terminal-Bench comparison prototype
- paired A/B uplift proof against a thinner repo-controlled substrate

The release question is no longer only "is the contract coherent".
It is also "does the kernel reduce wasted model work under controlled tasks".

## Migration Sequence

### Phase 0: Freeze The Consumer Baseline

Before upgrading:

- record the exact `v0.2.x` tag you depend on
- record whether you rely on:
  - shell-like fallback assumptions
  - implicit mount locality
  - text-only builtin parsing
  - entry-surface-specific behavior instead of kernel behavior

If you cannot list those assumptions, you are not ready to upgrade cleanly.

### Phase 1: Re-center On The Kernel

Treat CLI, TUI, and HTTP as wrappers over the kernel, not as competing sources of truth.

Migration actions:

- move integration reasoning to runtime/session/trace contracts first
- stop baking semantics into one entry surface only
- make "sandbox kernel beneath harness" the primary mental model

### Phase 2: Audit Mounts And Adapters

Review every non-trivial mount against:

- truth model
- materialization mode
- write semantics
- latency class
- consistency guarantees
- supported CLI classes

Migration actions:

- identify remote or high-latency mounts
- add capability support where workload pressure requires it
- remove silent fanout fallbacks where they are no longer acceptable

### Phase 3: Audit Agent-Facing Tooling

Review your harness or agent prompts and wrappers for outdated assumptions.

Migration actions:

- prefer `json`, `rg`, and structured modes where they reduce token cost
- stop reconstructing meaning from decorative output where structured output now exists
- update any tool-selection heuristics that still assume older builtin gaps

### Phase 4: Adopt Evidence Gates

Use the benchmark layers as rollout evidence rather than as sidecar demos.

Migration actions:

- run the native reference suite
- run the paired uplift harness when your change claims agent-efficiency benefit
- keep comparison layers downstream from native truth, not as a second SSOT

## Impact By Consumer Type

### Harness And AgentOS Teams

This is the main audience for `v0.3.0`.
The biggest gain is a clearer execution substrate contract and stronger proof that the kernel reduces agent confusion and wasted work.

### Memory-Aware Runtime Teams

The main change is sharper projection and mount guidance.
You should be more explicit about what is managed memory, what is projected truth, and what remains adapter control-plane behavior.

### Adapter Authors

You should treat mount capability and latency contracts as first-class release concerns, not as implementation details.

### CLI-Only Users

Lowest migration pressure.
The main change for you is documentation and higher-signal defaults, not a whole new product model.

## Release Gate For v0.3.0

Before cutting `v0.3.0`, the repo should satisfy all of these:

- docs describe `simsh` as an agent sandbox kernel beneath harnesses and AgentOS-style systems
- the high-performance mount contract is enforced strongly enough to avoid silent high-latency fanout regressions
- builtin ACI/query-tooling changes are validated by tests
- release docs and migration docs are aligned with the actual version line
- benchmark evidence is current enough to support the release story

## Decision Rule

If your integration only needs the stable session/result/trace contract line and does not care yet about stronger mount dispatch or agent-efficiency evidence, `v0.2.x` may still be enough.

If your integration depends on `simsh` as a serious harness substrate, especially with memory projections, capability-sensitive mounts, or benchmark-backed agent-execution claims, you should plan for the `v0.3.0` line explicitly.
