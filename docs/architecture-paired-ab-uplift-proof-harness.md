---
title: Paired A/B Uplift Proof Harness
required: false
sop:
  - Read this doc before changing the paired uplift benchmark, baseline substrate, or per-run report schema.
  - Keep the experiment controlled: hold the agent, paired task set, and budgets fixed while changing only the runtime substrate.
  - Keep the paired uplift layer downstream from the native benchmark suite and external comparison artifacts; do not mutate those layers to make uplift look better.
  - Regenerate `docs/must-sop.md` after SOP/frontmatter changes.
---

# Paired A/B Uplift Proof Harness

## Why This Doc Exists

`simsh` now has three benchmark/evidence layers:

- a native reference suite
- an external benchmark mapping layer
- a lightweight Terminal-Bench comparison layer

Those layers prove truth-surface quality and external-comparison fit.
They do **not** yet answer the next higher-value question:

> with the agent, tasks, and budgets held fixed, does `simsh` actually improve outcomes relative to a thinner runtime substrate?

This document defines the contract for answering that question without turning the repository into a generic benchmark zoo.

## Design Position

The paired uplift harness is:

- a controlled proof layer
- downstream from existing benchmark truth surfaces
- intentionally smaller than a general benchmark suite

It is **not**:

- a leaderboard
- a host-shell bakeoff
- a hidden external benchmark adoption layer
- a reason to mutate native scenarios until they flatter `simsh`

## Relationship to Existing Benchmark Layers

The benchmark stack should now be read in this order:

1. `benchmarks/simsh_native_reference/`
   - proves native truth surfaces directly
2. `benchmarks/external_mapping/`
   - maps native scenarios to external benchmark families
3. `benchmarks/terminal_bench_compare/`
   - emits a narrow downstream comparison artifact
4. `benchmarks/paired_uplift/`
   - asks whether the current runtime substrate creates measurable agent-task uplift under controlled paired tasks

The paired uplift layer must stay downstream from the first three.
It may reuse their identities, categories, summaries, and truth-surface language.
It must not rewrite their semantics.
The repo-level checked-in breadcrumb for those proof artifacts is
`benchmarks/evidence_manifest.json`; it indexes the current evidence set and
canonical refresh entrypoints without creating a second benchmark harness.
Its byte fingerprints audit the currently checked-in artifact bytes, while the
volatile-field declarations mark values expected to move when reports are
regenerated.

## Core Invariant

Every paired run must hold these constant:

- the deterministic probe agent
- the paired task set
- the task success criteria
- the step budget
- the observation budget

The only variable should be the runtime substrate.

That invariant is stricter than “run similar tasks twice”.
It is what makes uplift evidence explainable instead of anecdotal.

## Why the Baseline Must Stay Repo-Controlled

The baseline substrate should be a repo-controlled thin runtime, not an ambient host shell.

Reason:

- host shells inherit uncontrolled command availability and environment drift
- host PATH differences would contaminate the comparison with machine-local state
- the harness is trying to compare runtime substrate quality, not workstation setup luck

So the baseline should be:

- deliberately thinner than full `simsh`
- deterministic inside the repo
- explicit about which command and result surfaces are missing

## Baseline Design Rule

The baseline substrate should remove or narrow agent-oriented leverage surfaces while preserving enough shell/file behavior to execute the same paired tasks.

Examples of acceptable baseline narrowing:

- no `json` inspector
- no `rg` search front door
- no session-scoped cwd continuity if that is one of the runtime surfaces under test
- no other specialized query conveniences that the paired task explicitly pressures

Examples of unacceptable baseline drift:

- changing the task filesystem layout
- changing success semantics
- changing budgets
- relying on host-only binaries

## Task-Set Design Rule

The paired task set should emphasize current `simsh` strengths that matter to real agent loops:

- targeted structure-aware querying
- lower-fanout local search
- lower-noise inspect/edit/review loops
- explicit environment feedback when a substrate lacks a capability

The task set should **not**:

- invent impossible tasks just to make the baseline fail
- require product-specific business logic
- duplicate the full native benchmark suite

Each paired task should declare:

- its stable task id
- its summary
- its source truth surfaces or scenario anchors
- its native scenario id and category join keys when applicable
- its budgets
- its success criteria
- the deterministic agent behavior used for both substrates

The first checked-in task-set manifest should be a repository artifact rather than ad hoc code-only state.

## Task-Manifest Contract

The paired uplift layer should keep one checked-in task manifest as its scope SSOT.

That manifest should declare:

- stable paired task ids
- native scenario join keys where applicable
- task summary and rationale
- per-task budgets
- expected outputs
- evidence references

The manifest should stay narrow in the first cut.
Expanding it should be treated as a new bounded feat, not as a quiet benchmark-scope drift.

## Budget Contract

Budgets are first-class and must be explicit per task.

The minimum budget surface is:

- `max_steps`
- `max_observation_tokens`

`max_observation_tokens` may be heuristic if no live model is in the loop, but the heuristic must be:

- deterministic
- documented
- used consistently across both substrates

If approximation is used, the report field must say so explicitly.

## Order and Seed Contract

The paired harness should make order and seed explicit.

Minimum fields:

- pair seed
- run order (`AB` or `BA`)
- deterministic stop conditions

This prevents “same task, same budget” from quietly turning into different run conditions.

## Step Classification Contract

Each executed step should classify what happened, not only whether it exited with code `0`.

Minimum classifications:

- `progress`
- `retry`
- `wasted`
- `environment_misunderstanding`

`environment_misunderstanding` should be reserved for substrate-driven confusion such as:

- missing command
- missing structured output mode
- substrate-specific capability mismatch that forces a fallback branch

This keeps the report focused on runtime leverage rather than generic task failure.

## Per-Run Report Contract

Each substrate run should emit a machine-readable report with at least:

- task id
- native scenario id and category when applicable
- substrate id
- seed
- run order
- success
- duration
- step count
- retry count
- wasted-step count
- environment-misunderstanding count
- observation bytes
- approximate observation tokens
- per-step records

Per-step records should capture:

- command
- exit code
- observation size
- classification
- classification source when the step used a structured seam such as
  `ExecutionTrace.ExternalOutcomes`
- compact external outcome breadcrumbs when the step depends on external
  command availability or capability mismatch
- external outcome breadcrumbs should be a safe artifact projection, not a raw
  provider trace dump: preserve command identity, outcome kind, compatibility
  exit code, and virtual command-path identities, but avoid serializing host-local
  absolute paths into checked-in reports
- short note when a fallback branch or misunderstanding was triggered

If a task fails because it exhausted the fixed budget, the report should preserve both:

- the budget-exhaustion fact
- the last misunderstanding or fallback cause that led there

## Aggregate Report Contract

The aggregate paired uplift report should summarize:

- total tasks
- per-substrate success counts/rates
- per-substrate retry and wasted-step totals
- per-substrate approximate observation-token totals
- task-level deltas and winners

The purpose is not only “did `simsh` win”.
The purpose is:

- where did it save retries
- where did it save observation budget
- where did it avoid environment confusion

These uplift metrics are downstream proof metrics.
They do not replace the native benchmark gate set; they complement it.

## Failure Taxonomy Contract

The harness should also emit a separate failure-taxonomy artifact.

That artifact should group repeated failure causes into stable categories, for example:

- missing command
- unsupported structured query surface
- budget exhausted after fallback
- wrong-path or capability misunderstanding

The taxonomy should stay small and diagnosis-oriented.
It should not become a dump of one-off strings.
When an entry is derived from external command availability, the taxonomy should
preserve the classification source and observed external outcome kind so the
proof artifact remains auditable without parsing stdout or stderr.

## Non-Goals

- claiming cross-project benchmark parity
- widening into full Terminal-Bench or SWE-bench adoption
- introducing ambient host-shell variability into the comparison
- changing native benchmark semantics to improve paired uplift numbers
- building a generic agent-evaluation platform inside this repo

## Initial Implementation Shape

The first cut should stay narrow:

- one repo-controlled thin baseline substrate
- one deterministic probe agent
- one small paired task set
- one machine-readable aggregate artifact
- one adjacent human-readable summary
- one separate failure-taxonomy artifact

If a future wave needs more breadth, it should add a new bounded feat rather than silently widening this harness.
