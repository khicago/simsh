---
title: Anthropic - Building Effective Agents
summary: Anthropic argues that simple, composable agent patterns and carefully designed tool interfaces outperform overcomplicated orchestration; they highlight absolute file paths and explicit feedback as practical reliability wins.
source: https://www.anthropic.com/engineering/building-effective-agents
date: 2024-12-19
tags:
  - anthropic
  - agent-design
  - aci
---

# Anthropic: Building Effective Agents

## Experience Extracts
- Successful teams tend to use simple, composable patterns instead of complex agent frameworks.
- Anthropic explicitly separates workflows from agents and argues against overcomplicating orchestration too early.
- They emphasize transparency, explicit planning, and careful ACI/tool design as core reliability levers.
- Their SWE-bench tool-design example is especially relevant: they found relative file paths caused mistakes after directory changes, and requiring absolute paths fixed that class of failure.

## Implication For simsh
- `simsh` should optimize for a small number of clear runtime invariants before adding more orchestration features.
- File-path semantics are part of the kernel contract, not just a convenience detail.
