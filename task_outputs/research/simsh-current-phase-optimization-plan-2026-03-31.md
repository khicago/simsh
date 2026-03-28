---
title: simsh Current-Phase Optimization Plan
summary: A current-phase synthesis of simsh against recent agent-runtime papers and similar systems, with a concrete next-wave plan centered on adapter-side freshness, refresh, workflow truthfulness, and alternating implement-review-refine loops.
updated: 2026-03-31
tags:
  - research
  - simsh
  - agent-runtime
  - planning
---

# simsh Current-Phase Optimization Plan

Date: 2026-03-31

## Primary References

Papers already stored locally:
- `knowledge_base/papers/agent-runtime/osworld-2404.07972.pdf`
- `knowledge_base/papers/agent-runtime/swe-agent-2405.15793.pdf`
- `knowledge_base/papers/agent-runtime/programming-with-pixels-2502.18525.pdf`
- `knowledge_base/papers/agent-runtime/researchenvbench-2603.06739v2.pdf`

Experience notes already stored locally:
- `knowledge_base/experience/agent-runtime/openai-harness-engineering-2026-02-11.md`
- `knowledge_base/experience/agent-runtime/openhands-runtime-architecture-2026-03-31.md`
- `knowledge_base/experience/agent-runtime/swe-rex-runtime-framework-2026-03-31.md`
- `knowledge_base/experience/agent-runtime/index.md`

## First-Principles Read On simsh

`simsh` is not trying to win by being the smallest Linux clone.

It is trying to win by giving an agent:
- a more legible environment than a host shell;
- a lighter environment than a VM or containerized devbox;
- more trustworthy side-effect and path semantics than an ad hoc CLI wrapper;
- a reusable runtime kernel beneath harnesses, memory systems, and product UIs.

That means the next phase should not optimize for more shell breadth.

It should optimize for:
- environment truth;
- adapter-side realism without core pollution;
- explicit refresh and workflow semantics;
- continuous review and design-refinement loops.

## What Similar Systems Suggest

### OpenAI harness engineering
- The highest-leverage work is environment design, not prompt cleverness.
- Repository-local docs, plans, and evidence are part of runtime quality.
- Agent legibility is a better target than raw realism.

### OpenHands runtime architecture
- Runtime/provider seams should stay explicit.
- Mount, rebuild, and environment policies should be visible and documented.
- Dynamic runtime behavior becomes manageable only when lifecycle policy is explicit.

### SWE-ReX
- Execution infrastructure should stay reusable and generic.
- Agent logic should not be tightly coupled to one runtime backend.
- Parallelism and shell-session realism matter, but they belong below the agent loop.

### SWE-agent and Programming with Pixels
- The winning substrate for software work remains file editing plus shell plus high-quality ACI.
- Tool/interface design matters more than imitating a full computer.

### OSWorld and ResearchEnvBench
- Real environment and file workflow issues are central, not edge cases.
- But full environment complexity is still too noisy.
- The right move for `simsh` is not to synthesize whole environments; it is to make environment freshness and drift visible enough that a harness can reason about them.

## Current Gap

The reference adapter is already much better than a toy projection layer. It now has:
- `/knowledge_base/reference`
- `/resources`
- managed `/memory`
- projection metadata
- minimal control-plane upserts
- workflow summaries

But it still lacks two kinds of truth that matter for a realistic harness:

1. `projection freshness lifecycle truth`
- Today freshness is mostly a label, not a lifecycle.
- There is no explicit invalidation or refresh model.
- A projected item can look present without the runtime exposing whether it is fresh, stale, refreshed, or failed.

2. `workflow transition truth`
- Current workflow status is mostly derived heuristically from read/write/deny evidence.
- That is useful, but it is not yet an explicit transition model with a clear adapter-side override/control-plane seam.

## Current-Phase Plan

### Wave 1
Implement realistic projection freshness and refresh semantics.

Target outcomes:
- projected docs/resources expose explicit lifecycle state, not only a static freshness label;
- control plane can invalidate and refresh projections explicitly;
- `/memory/status.json`, `/memory/projections.json`, and projection indexes stay aligned.

### Wave 2
Review the new freshness and refresh model through adapter tests and benchmark gates.

Target outcomes:
- reference validation treats stale or invalid projections as first-class evidence;
- benchmark success does not ignore freshness regressions;
- review findings are concrete enough to feed back into docs and rows.

### Wave 3
Refine the docs, backlog, and execution rows based on review findings.

Target outcomes:
- architecture and backlog language reflect the real contract;
- the next execution rows are updated from review findings instead of drifting away from code.

### Wave 4
Implement explicit workflow transition and managed-memory curation boundaries.

Target outcomes:
- workflow state is not only path-derived but also explicitly representable at the adapter control plane;
- managed `/memory` makes the boundary between raw observation, derived workflow state, and curated state clearer;
- the reference adapter stays realistic without becoming a product-specific memory engine.

## Recommended Team Shape

Use a three-lane alternating loop:

1. `Implementer`
- lands the smallest adapter-side slice that changes runtime truth

2. `Reviewer`
- replays tests and benchmark scenarios
- hunts for contract drift and evidence gaps

3. `Refiner`
- writes findings back into backlog, docs, and execution rows before the next implementation wave

This is the right shape because the current risks are not just code defects. They are truth drift between:
- adapter behavior
- benchmark evidence
- docs and backlog contract
- long-run execution planning

## Bottom Line

The strongest next move is:

- do not broaden the shell;
- do not push product memory semantics into core;
- do make adapter-side freshness and workflow semantics explicit, reviewable, and benchmark-visible;
- do force the next wave to alternate implement, review, and refine instead of stacking implementation-only passes.
