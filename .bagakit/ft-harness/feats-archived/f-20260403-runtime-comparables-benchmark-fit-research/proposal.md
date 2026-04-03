# Feat Proposal: f-20260403-runtime-comparables-benchmark-fit-research

## Why
- The current adapter-seam proof wave is complete, but the next investment choice is still open.
- The highest-value next move is no longer obvious from local implementation debt alone; it now depends on how `simsh` compares to directly relevant runtime designs and benchmark families.

## Goal
- Compare directly relevant runtime implementations and benchmark families, then recommend the next simsh feat based on fit rather than intuition.

## Scope
- In scope:
  - A narrow comparison against directly relevant runtime implementations such as SWE-ReX and OpenHands runtime/sandbox design.
  - A narrow benchmark fit study across families such as Terminal-Bench, SWE-bench-Live, EnvBench, ResearchEnvBench, and SWE-Skills-Bench.
  - One explicit recommendation for the next simsh feat, plus explicit rejected alternatives.
- Out of scope:
  - Open-ended literature review.
  - New product features.
  - Full benchmark adoption or integration work.

## Impact
- Code paths:
  - research outputs under `task_outputs/research/`
  - planning docs if the recommendation becomes canonical
- Tests:
  - none directly; this is a research/decision slice
- Rollout notes:
  - Keep the outcome decision-oriented. The deliverable is a recommendation, not just a survey.
