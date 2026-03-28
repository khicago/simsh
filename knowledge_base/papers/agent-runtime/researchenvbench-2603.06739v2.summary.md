---
title: ResearchEnvBench summary
paper: ResearchEnvBench
arxiv: 2603.06739v2
pdf: knowledge_base/papers/agent-runtime/researchenvbench-2603.06739v2.pdf
source: https://arxiv.org/abs/2603.06739v2
---

# ResearchEnvBench

## Citation
- Title: ResearchEnvBench: Benchmarking Agents on Environment Synthesis for Research Code Execution
- Version used: arXiv v2, March 11, 2026

## What It Is
- A benchmark for whether agents can synthesize a runnable environment for real research repositories.
- Inputs include a repository, documentation, and a target execution setting.
- The task is not just code repair; the agent must resolve dependencies and make the environment executable.

## Core Thesis
- Existing agent benchmarks usually assume the execution environment already exists.
- That misses a major source of real-world failure: dependency resolution, version coupling, and environment drift.

## Main Findings
- Current agents still struggle substantially when environment construction becomes part of the task.
- Failures are dominated by incomplete dependency resolution and brittle version coupling.

## Why It Matters For simsh
- `simsh` should not try to become a full environment synthesizer, but it should expose environment state and projection freshness clearly enough that higher-level harnesses can reason about drift.
- Adapter-side projection freshness, invalidation, and refresh semantics matter because environment truth is part of agent trust, not just a backend implementation detail.
- A realistic reference adapter should model staleness and refresh explicitly rather than treating all projected files as timeless snapshots.
