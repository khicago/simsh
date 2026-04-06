---
title: v0.3.0 Release Readiness
required: false
sop:
  - Read this doc before cutting the v0.3.0 line or claiming the repository is release-ready.
  - Update this doc when the v0.3.0 closeout checklist, evidence set, or cut criteria change.
  - Regenerate `docs/must-sop.md` after SOP/frontmatter changes.
---

# v0.3.0 Release Readiness

## Why This Doc Exists

`docs/notes-v0-2-x-to-v0-3-0-migration.md` explains how the `v0.3.0` line differs from `v0.2.x`.

This document is narrower.
It answers one operational question:

> is the repository actually ready to justify a deliberate `v0.3.0` release cut right now?

It is a closeout artifact, not a new roadmap.

## Scope

This doc tracks:

- release-facing documentation alignment
- benchmark and uplift evidence freshness
- remaining release risks
- the explicit cut criteria for the `v0.3.0` line

This doc does not introduce:

- new runtime features
- new benchmark families
- publish automation

## Current Checklist

- [x] `simsh` is described as an agent sandbox kernel beneath harnesses and AgentOS-style systems.
  - Primary refs:
    - `README.md`
    - `docs/architecture-overview.md`
    - `docs/notes-project-charter.md`
- [x] High-performance mount behavior is documented and backed by direct proof instead of intent-only prose.
  - Primary refs:
    - `docs/architecture-high-performance-mount-system.md`
    - `docs/notes-v0-2-x-to-v0-3-0-migration.md`
    - `pkg/contract/mount_dispatch_test.go`
    - `pkg/engine/remote_high_latency_mount_test.go`
    - `pkg/builtin/op_remote_high_latency_test.go`
- [x] Builtin ACI and query-tooling changes are part of the release story and remain test-backed.
  - Primary refs:
    - `docs/notes-builtin-aci-review.md`
    - `pkg/builtin/op_json_test.go`
    - `pkg/builtin/op_search_runtime_test.go`
- [x] External-comparison and paired-uplift layers remain downstream from the native benchmark SSOT.
  - Primary refs:
    - `benchmarks/external_mapping/README.md`
    - `benchmarks/terminal_bench_compare/README.md`
    - `benchmarks/paired_uplift/README.md`
- [x] Current benchmark evidence has been refreshed through the canonical repository entrypoints.
  - Commands:
    - `make benchmark-refresh`
    - `make benchmark-uplift`
    - `make check`

## Current Evidence Snapshot

### Native Reference Suite

Source:
- `benchmarks/simsh_native_reference/reports/baseline-20260404.json`

Current headline metrics:
- `trace_completeness = 1.0`
- `session_success = 1.0`
- `reviewable_patch_latency_ms = 25`
- `async_completion_success = 1.0`

Interpretation:
- the native suite is not only green; it is comfortably above configured gates.

### Terminal-Bench Comparison Prototype

Sources:
- `benchmarks/terminal_bench_compare/reports/prototype-baseline-20260404.json`
- `benchmarks/terminal_bench_compare/reports/prototype-baseline-20260404.md`

Current scope:
- `1` direct-fit slice
- `1` translated proof slice
- all compared native scenarios successful

Interpretation:
- the external-comparison layer remains intentionally narrow and still behaves like a downstream proof artifact rather than a second benchmark suite.

### Paired A/B Uplift

Sources:
- `benchmarks/paired_uplift/reports/raw-baseline-20260406.json`
- `benchmarks/paired_uplift/reports/paired-baseline-20260406.json`
- `benchmarks/paired_uplift/reports/paired-baseline-20260406.md`
- `benchmarks/paired_uplift/reports/paired-baseline-20260406.failures.json`

Current headline deltas:
- `simsh success = 3/3`
- `baseline success = 2/3`
- `simsh retries = 0`
- `baseline retries = 3`
- `simsh misunderstandings = 0`
- `baseline misunderstandings = 3`
- `simsh observation tokens = 149`
- `baseline observation tokens = 3826`

Interpretation:
- this is not cross-project leaderboard evidence
- it is repo-controlled proof that the current kernel reduces wasted model work under a fixed task set

## Remaining Risks

- The repository is release-ready in the engineering sense, but it is not yet a published `v0.3.0` release line.
- Benchmark freshness snapshots intentionally include volatile timing fields; release discussion should focus on gate pass/fail and directional deltas, not byte-stable timing numbers.
- Long-run should remain explicitly idle unless a real post-closeout row is created; release execution should not quietly masquerade as another feature wave.

## Cut Criteria

The repository is ready for a deliberate `v0.3.0` cut when all of the following are true:

1. `make check` is green.
2. `make benchmark-refresh` has been run and the checked-in native + Terminal-Bench comparison evidence is current.
3. `make benchmark-uplift` has been run and the checked-in paired A/B evidence is current.
4. README, migration docs, backlog, and long-run handoff all describe the same current release line and evidence stack.
5. No new bounded engineering feat is still being treated as part of the `v0.3.0` scope.

## Out Of Scope After This Closeout

Once this closeout is complete, the next step is a deliberate release action or a new bounded engineering wave.

It should not be:

- a stealth benchmark-family expansion
- a stealth mount-feature expansion
- a silent rewrite of migration/versioning language

