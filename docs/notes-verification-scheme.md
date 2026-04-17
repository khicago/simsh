---
title: Verification Scheme
required: false
sop:
  - Read this doc before adding a new test layer, benchmark family, or CI gate.
  - Keep the five verification layers distinct: unit/contract, release kernel gate, native reference, external comparison, paired uplift.
  - Do not adopt an external benchmark wholesale, and do not put LLM-in-the-loop work on the default CI path.
  - Update this doc when Makefile verify targets, CI jobs, or layer ownership change.
  - Regenerate `docs/must-sop.md` after SOP/frontmatter changes.
---

# Verification Scheme

## Why This Doc Exists

`simsh` already has several proof surfaces. The problem is not "too few tests".
The problem is that they answer different questions and were easy to treat as one pile.

This document is the SSOT for what each layer proves, which command runs it, and what must not leak into another layer.

It is not a new benchmark family.

## Layer Map

| Layer | Question it answers | Command | Default CI | Must stay |
| --- | --- | --- | --- | --- |
| L0 unit/contract | Did this package's public behavior break? | `make test-unit` | yes | Fast, local, no live LLM |
| L1 release kernel gate | Is the kernel still safe to cut? | `make release-check` | yes | Includes full `go test ./...`, lint, prepared-exec allocation gate, focused race |
| L2 native reference | Do realistic file/session/mount workflows still work on `simsh` itself? | `make test-native` | yes, as package tests | Deterministic scenarios; native scenario ids are SSOT |
| L3 external comparison | What is the smallest honest Terminal-Bench-shaped comparison worth keeping? | `make test-compare` | yes, as package tests | Downstream mapping + 1 direct + 1 translated slice only |
| L4 paired uplift | Holding agent/tasks/budgets fixed, does full `simsh` waste less work than a thin substrate? | `make test-uplift` | yes, as package tests | Repo-controlled baseline; do not regenerate reports in CI |

`make verify` is the operator-facing alias for `release-check`. Its full `go test ./...`
step already executes the package tests behind the named layer targets above.

`make test` (`go test ./...`) already covers L0 through L4 package tests. The named targets exist so a failure can be isolated to one question.

## What Each Layer Must Not Do

- L0 must not grow into a second benchmark suite.
- L1 must not depend on wall-clock microbenchmarks as hard gates.
- L2 must not mutate native scenarios to look more like Terminal-Bench or SWE-bench.
- L3 must not adopt Terminal-Bench wholesale or add SWE-bench-Live as a live runner.
- L4 must not compare against an ambient host shell, widen the 3-task first cut without an explicit scope change, or put an LLM agent on the default CI path.

Refreshing checked-in reports (`make benchmark-refresh`, `make benchmark-uplift`) is a maintainer action, not a CI action.

## How To Choose A Layer

- Bug in a builtin, path guard, or session rule: add or tighten an L0 test next to the code.
- Change that could regress prepared execution or racey session/HTTP paths: keep L1 green; do not weaken `perf-check` into a timing assertion.
- Change to default workspace loops, mounts, traces, or adapter-backed workflows: add or update an L2 native scenario only if the behavior is a reusable workflow, not a one-off unit case.
- Change to mapping/export/freshness of external comparison artifacts: stay in L3.
- Change to retries, observation tokens, or substrate misunderstandings: stay in L4.

If a change needs an LLM to judge success, it does not belong on `make verify`.

## Current Proof Owners

- L0/L1: `pkg/`, `cmd/`, `Makefile`
- L2: `benchmarks/simsh_native_reference/`, `benchmarks/internal/scenarios/`
- L3: `benchmarks/external_mapping/`, `benchmarks/terminal_bench_compare/`
- L4: `benchmarks/paired_uplift/`, `docs/architecture-paired-ab-uplift-proof-harness.md`
- Evidence index: `benchmarks/evidence_manifest.json`

## Next Improvements Worth Doing

These are in-scope later, not part of the current scheme:

- split the largest `*_test.go` files so L0 stays readable;
- grow L2/L4 only with tasks that pressure kernel truth (cwd, search/edit, mount refusal, traces);
- add one real embedder dogfood path before adding another synthetic benchmark family.
