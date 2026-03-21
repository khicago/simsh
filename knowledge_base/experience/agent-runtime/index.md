---
title: Frontier Agent Team Experience Index
summary: Index of practical lessons from frontier agent teams on coding agents, runtimes, traces, review loops, and isolation, with direct implications for simsh kernel design.
updated: 2026-03-22
tags:
  - agent-runtime
  - coding-agents
  - field-notes
---

# Frontier Agent Team Experience Index

## Scope
- Practical lessons shared by frontier agent teams through official blogs, product posts, and transcript-backed interviews.
- Focus areas: coding agents, agent runtimes, supervision, traces, tool design, review loops, and isolation.

## Files
- `knowledge_base/experience/agent-runtime/anthropic-building-effective-agents-2024-12-19.md`
- `knowledge_base/experience/agent-runtime/anthropic-building-agents-with-claude-agent-sdk-2025-09-29.md`
- `knowledge_base/experience/agent-runtime/openai-introducing-codex-2025-05-16.md`
- `knowledge_base/experience/agent-runtime/openai-introducing-the-codex-app-2026-02-02.md`
- `knowledge_base/experience/agent-runtime/cognition-how-cognition-uses-devin-to-build-devin-2026-02-27.md`
- `knowledge_base/experience/agent-runtime/cognition-closing-the-agent-loop-2026-02-10.md`
- `knowledge_base/experience/agent-runtime/cognition-agent-trace-2026-01-29.md`
- `knowledge_base/experience/agent-runtime/lennys-podcast-scott-wu-on-devin-2025-09-08.md`
- `knowledge_base/experience/agent-runtime/latent-space-claude-code-2025.md`

## Cross-Source Themes
- File system plus shell beats full GUI realism for software work.
- ACI quality matters more than shell breadth.
- Context preservation and traceability are becoming first-class.
- Review and correction loops are the next bottleneck after generation.
- Isolation and scoped concurrency matter once agents scale.

## Relevance To simsh
- `simsh` should optimize for trustworthy file editing, search, and shell execution.
- Kernel contracts should be explicit enough that agents do not need to guess at path semantics or side effects.
- `ExecutionTrace`, reviewability, and scoped sessions should be treated as core capabilities rather than accessories.
