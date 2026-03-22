# Feat Proposal: f-20260322-simsh-native-reference-validation-metric-gates

## Why
- The kernel mainline now has hardened boundaries, explicit path semantics, better trace fidelity, and observable cancellation behavior.
- The next unanswered question is not "can simsh do it" but "does simsh measurably help agent file-workflows better than heavier environments or raw interfaces".
- Without a small reference benchmark and explicit metric gates, P4 stays rhetorical and future kernel work loses a hard usefulness feedback loop.

## Goal
- Build a small simsh-native benchmark and threshold-based metric gates that prove the current kernel improves realistic agent file workflows.

## Scope
- In scope:
- a small `simsh`-native benchmark pack covering relative navigation, inspect/edit/write loops, mount or synthetic boundaries, trace-consumable planning, and cancel or timeout scenarios
- metric definitions and threshold gates for trace completeness, session success, reviewable patch latency, and async completion success
- baseline reporting that marks metrics pass or fail instead of emitting raw numbers only
- Out of scope:
- adding new kernel primitives that are unrelated to validation
- product-layer leaderboard or large-scale orchestration infrastructure
- legacy harness schema cleanup

## Impact
- Code paths:
- benchmark or validation artifacts and any lightweight runner or scoring code needed to execute them repeatably
- Tests:
- repeatable benchmark command plus `go test ./...`
- Rollout notes:
- keep this feat proposal-only until the benchmark pack shape is reviewed; it is a proof feat, not a speculative primitive-expansion feat
