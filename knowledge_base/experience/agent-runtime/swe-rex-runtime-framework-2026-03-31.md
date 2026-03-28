---
title: SWE-ReX - Remote Execution Framework
summary: SWE-ReX separates agent logic from execution infrastructure, supports multiple shell sessions and platforms, and treats runtime interaction itself as a reusable framework. The direct lesson for simsh is to keep execution semantics and infrastructure seams generic while keeping evidence and validation strong.
source: https://github.com/SWE-agent/SWE-ReX
date: 2026-03-31
tags:
  - swe-rex
  - runtime
  - shell-sessions
  - infrastructure
---

# SWE-ReX: Remote Execution Framework

## Experience Extracts
- SWE-ReX is framed as a runtime interface, not as an agent implementation.
- It supports interacting with running shell sessions, including interactive tools, while keeping the agent-side code stable across local and remote backends.
- It emphasizes parallel session execution and broad platform support as infrastructure concerns, not as agent logic.
- The stated design goal is to disentangle agent logic from infrastructure concerns so the agent stack is easier to maintain and evaluate.

## Implication For simsh
- `simsh` should keep strengthening the runtime and adapter seam rather than turning the core into a product-specific orchestration layer.
- When richer adapter-side non-core behavior is added, the right shape is still generic runtime contract plus explicit control plane, not hidden coupling between one harness and one adapter.
- Reference validation should remain strong enough that infrastructure realism does not quietly outrun evidence quality.
