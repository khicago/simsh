---
title: Reference adapter `/skills` projection should stay read-only and keep ineligible skills visible
kind: howto
confidence: high
tags:
  - adapter
  - skills
  - projection
  - benchmark
sources:
  - pkg/adapter/reference/adapter.go
  - pkg/adapter/reference/adapter_test.go
  - benchmarks/simsh_native_reference/suite.go
  - docs/architecture-memory-skills-extension.md
  - docs/architecture-platform-adapter-contract.md
created: 2026-04-01T00:00:00Z
updated: 2026-04-01T00:00:00Z
---

## Context

The first `/skills` projection slice for the reference adapter needed to stay adapter-local and evidence-friendly without turning into a mutable skill registry or hiding ineligible skills from the caller.

## Guidance

- Prefer canonical skill paths under `/skills/.../SKILL.md` when the caller provides a logical skill name instead of a file name.
- Keep `/skills` read-only; use metadata rather than mount writeability to express selection, eligibility, or precedence.
- Keep ineligible skills visible in `_index.json` and `/memory/projections.json`; callers should not have to infer ineligibility from disappearance.
- Surface at least `source`, canonical `freshness`, `eligibility`, `precedence`, and optional `selected` state in machine-readable records.
- Benchmark one selected eligible skill and one visible-but-ineligible fallback skill so selection and non-selection are both proven.

## Scope

- Applies to the reference adapter and future adapter-side `/skills` work.
- Does not require a skill execution engine, install/update registry, or core-runtime changes.
