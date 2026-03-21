---
title: OSWorld summary
paper: OSWorld
arxiv: 2404.07972
pdf: knowledge_base/papers/agent-runtime/osworld-2404.07972.pdf
source: https://arxiv.org/abs/2404.07972
---

# OSWorld

## Citation
- Title: OSWorld: Benchmarking Multimodal Agents for Open-Ended Tasks in Real Computer Environments
- Version used: arXiv v2, May 30, 2024

## What It Is
- A benchmark and real computer environment for multimodal agents.
- Covers Ubuntu, Windows, and macOS.
- Includes 369 open-ended tasks across web apps, desktop apps, OS file I/O, and cross-application workflows.

## Core Thesis
- Existing benchmarks are too narrow because they either lack executable environments or only test within a single app/domain.
- Real computer use requires agents to operate across arbitrary applications, interfaces, and OS-level file workflows.

## Main Findings
- Humans solve more than 72% of tasks.
- The best evaluated model in the paper solves only about 12%.
- The main failure modes are GUI grounding and weak operational knowledge.

## Why It Matters For simsh
- It validates that file I/O and multi-step computer workflows are core agent problems, not edge cases.
- It also shows that a fully realistic computer environment is still too hard and noisy for current agents.
- For `simsh`, this supports a middle path: keep real file workflow semantics, but make the runtime more explicit and deterministic than a full OS GUI environment.

## Design Takeaways
- A benchmark should include real side effects, not only text completion.
- Execution-based evaluation matters because many tasks have multiple valid action sequences.
- If `simsh` wants to claim agent usefulness, it should eventually validate against tasks that look like real computer work, not only isolated command tests.
