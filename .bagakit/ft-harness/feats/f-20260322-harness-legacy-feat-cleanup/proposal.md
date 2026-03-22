# Feat Proposal: f-20260322-harness-legacy-feat-cleanup

## Why
- The remaining unarchived `20260301` feats still use legacy schema assumptions and currently fail modern harness validation.
- This is a process and maintenance problem, not a kernel-runtime problem.
- Leaving it mixed into the kernel mainline would blur product proof work with harness debt repayment.

## Goal
- Migrate legacy feat metadata to the current harness schema and define a safe archive cleanup path for pre-workspace_mode feats without mixing that work into the kernel mainline.

## Scope
- In scope:
- migrating legacy feat metadata to the current harness schema
- defining a safe archive policy for already-done historical feats
- validating and cleaning up old done feat records without ad hoc manual state mutation
- Out of scope:
- kernel runtime behavior
- benchmark or metric-gate work
- entry-adapter feature work

## Impact
- Code paths:
- `.bagakit/ft-harness/` runtime state and any supporting maintenance scripts or docs needed for safe migration
- Tests:
- harness validation and archive-safety checks
- Rollout notes:
- keep this feat isolated from kernel delivery so maintenance risk does not contaminate reference-validation conclusions
