---
title: v0.3.1 Patch Release Readiness
required: false
sop:
  - Read this doc before cutting the v0.3.1 patch line or claiming the current post-v0.3.0 tree is ready to tag.
  - Update this doc when the v0.3.1 patch scope, evidence set, or cut criteria change.
  - Regenerate `docs/must-sop.md` after SOP/frontmatter changes.
---

# v0.3.1 Patch Release Readiness

## Why This Doc Exists

`v0.3.0` is already tagged.

This document exists so the repository does not keep describing that released line as “upcoming” while `main` has already moved on with post-`v0.3.0` hardening and release-facing cleanup work.

It answers one narrower question:

> is the current post-`v0.3.0` tree coherent enough to cut a deliberate `v0.3.1` patch release?

## Patch Scope

This patch line is intentionally narrow.

It includes:

- `K-031` remote high-latency mount fail-closed proof
- `K-032` v0.3.0 release-readiness closeout and evidence refresh
- release-facing truth cleanup so docs and process state reflect the actual tagged baseline

It does not include:

- new runtime nouns
- new benchmark families
- new adapter waves
- publish automation

## Current Patch Summary

The current `main` line after `v0.3.0` adds two material things:

1. stronger mount-contract proof
   - `remote_high_latency` mounts now have direct contract, engine, and builtin proof that missing critical capabilities fail closed instead of silently fanning out
2. stronger release/evidence alignment
   - benchmark evidence has been refreshed
   - closeout docs and handoff have been brought into one explicit SSOT

## Evidence Snapshot

### Native Reference Suite

Source:
- `benchmarks/simsh_native_reference/reports/baseline-20260404.json`

Current headline metrics:
- `trace_completeness = 1.0`
- `session_success = 1.0`
- `reviewable_patch_latency_ms = 25`
- `async_completion_success = 1.0`

### Terminal-Bench Comparison Prototype

Sources:
- `benchmarks/terminal_bench_compare/reports/prototype-baseline-20260404.json`
- `benchmarks/terminal_bench_compare/reports/prototype-baseline-20260404.md`

Current scope remains:
- `1` direct-fit slice
- `1` translated proof slice

### Paired A/B Uplift

Sources:
- `benchmarks/paired_uplift/reports/raw-baseline-20260406.json`
- `benchmarks/paired_uplift/reports/paired-baseline-20260406.json`
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

## Cut Criteria

The repository is ready for a deliberate `v0.3.1` patch cut when all of the following are true:

1. `v0.3.0` is treated as the historical released baseline everywhere.
2. current `main` is framed consistently as the post-`v0.3.0` patch candidate line.
3. `go test ./...` is green.
4. `make check` is green.
5. the checked-in benchmark evidence referenced above is current.

## Out Of Scope After This Patch

After `v0.3.1`, the next step should be either:

- a deliberate new engineering wave
- or a later patch/feature release with a separately defined scope

It should not be a silent continuation of release-truth cleanup.

