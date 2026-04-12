# Feature Proposal: f-8mkhf382

## Why
- `f-223crkgwk` closed the first major runtime-contract wave: session active execution, trace truth, external outcome truth, operational mount budgets/introspection, and benchmark evidence-manifest guardrails.
- The strongest remaining opportunities are now split across three adjacent but still coherent lanes:
  - explicit mount refresh scope/budget contracts
  - the next external seam and adapter-boundary contract upgrades
  - proof-layer phase 2 work that strengthens controlled evidence without widening into a second benchmark system
- These lanes should live under one feature umbrella so the next phase remains strategically unified, but they still need separate task lanes to preserve auditability and avoid design spillover.

## Goal
- Advance `simsh` beyond the first runtime/proof wave by adding explicit mount refresh control contracts, maturing the next external seam boundary where it remains too weakly structured, and strengthening downstream proof layers while preserving kernel/adapter separation.

## Principle Layer
- What:
  - Run one umbrella feature with disjoint execution lanes for mount refresh control, external seam evolution, and proof-layer phase 2 upgrades.
- Why:
  - The remaining value is real, but it no longer sits in one file cluster or one isolated correctness gap. It sits in adjacent contract layers that should evolve together without being implemented as one blended rewrite.
- Intended generalization:
  - Leave `simsh` with clearer control-plane semantics for mounts, stronger structured seams for external capability injection, and more auditable downstream proof artifacts.
- Failure boundary:
  - Do not merge adapter-specific semantics into kernel contracts.
  - Do not let proof work turn into benchmark sprawl.
  - Do not add refresh-on-read or hidden control-plane behavior.
  - Do not broaden shell scope casually just because external seam work is active.
- Behavior examples:
  - mount refresh remains explicit and budgeted instead of piggybacking on ordinary read paths.
  - external command seams preserve structured truth without requiring benchmark-driven API distortion.
  - proof layers remain downstream from native benchmark truth and continue to justify checked-in evidence artifacts.
- Evidence refs:
  - `.bagakit/feature-tracker/features/f-223crkgwk/proposal.md`
  - `.bagakit/feature-tracker/features/f-223crkgwk/state.json`
  - `.bagakit/researcher/topics/frontier/simsh-top-tier-evolution/passes/pass-001.md`

## Scope
- In scope:
  - explicit mount refresh scope/budget/refusal contracts and their first execution surface
  - the next bounded external seam / adapter-boundary contract upgrades
  - proof-layer phase 2 improvements that strengthen auditable downstream evidence and controlled regeneration semantics
- Out of scope:
  - generic product workflow logic in kernel
  - a second benchmark harness or broad benchmark-family adoption
  - hidden refresh or hidden control-plane side effects on read paths
  - broad new builtin surface expansion unrelated to these lanes

## Acceptance Criteria
- A staged task graph exists with separate lanes for mount refresh, external seam, and proof-layer follow-up work.
- At least one lane lands with green `go test ./...` without blurring kernel/adapter boundaries.
- Mount refresh semantics become more explicit without introducing hidden fanout or hidden mutation.
- Proof-layer changes improve auditable reproducibility or evidence integrity in a reviewable way.

## Transfer Checks
- A future maintainer can answer:
  - what refresh contract is explicit now vs still adapter-local
  - which external seam truths are now structured vs still compatibility-only
  - what the proof layers claim and how the checked-in evidence set is regenerated
  - why these lanes were grouped into one feature but kept as separate tasks

## Impact
- Code paths:
  - `pkg/contract/**`
  - selected `pkg/engine/**` and `pkg/service/**`
  - selected adapter seam packages
  - `benchmarks/**`
  - `docs/**`
- Tests:
  - focused lane-specific tests per task
  - `go test ./benchmarks/...`
  - `go test ./...`
- Rollout notes:
  - Lane 1: mount refresh contract + explicit control surface
  - Lane 2: external seam / adapter-boundary phase 2
  - Lane 3: proof-layer phase 2 and evidence strengthening
