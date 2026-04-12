# Feature Proposal: f-8mkj92dt

## Why
- `f-223crkgwk` intentionally stopped short of generic refresh control because the scope and budget contract was still too weak.
- `f-8mkhf382` confirmed this should be the first independent execution lane after the umbrella planning pass.
- Mount refresh now needs its own focused feature boundary so contract semantics, builtin behavior, and fail-closed rules can converge without mixing in external seam or benchmark work.

## Goal
- Make mount refresh explicit, scope-aware, budgeted, and fail-closed across contract, builtin, and documentation surfaces.

## Scope
- In scope:
  - refresh request/result contract semantics
  - refresh narrowness and refusal semantics
  - `mounts refresh` builtin behavior and tests
  - mount architecture doc updates
- Out of scope:
  - external seam upgrades
  - proof-layer or benchmark work
  - product-specific adapter refresh workflows
