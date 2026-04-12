# Feature Proposal: f-8mkjav3p

## Why
- `f-223crkgwk` improved trace truth and external outcome truth, but it deliberately stopped before a larger external seam redesign.
- The remaining gap is no longer a single correctness bug; it is a bounded contract-quality problem at the external seam and platform adapter boundary.
- This needs its own feature so it can evolve after mount refresh semantics are settled, not in parallel chaos.

## Goal
- Strengthen the next general external seam and adapter-boundary structured contracts while preserving kernel/adapter separation.

## Scope
- In scope:
  - next external command/result/trace boundary upgrades
  - selected adapter-boundary truth surfaces that materially depend on those upgrades
  - docs/tests for the new seam semantics
- Out of scope:
  - mount refresh control
  - proof-layer benchmark work
  - broad new builtin capability expansion
