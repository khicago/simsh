---
title: OpenAI - Introducing Codex
summary: OpenAI describes Codex as running each task in an isolated repo-backed environment with strong observability through logs, tests, and citations, and stresses that agents work best with reliable tooling and explicit project conventions.
source: https://openai.com/index/introducing-codex/
date: 2025-05-16
tags:
  - openai
  - codex
  - observability
---

# OpenAI: Introducing Codex

## Experience Extracts
- Each task runs in its own isolated environment preloaded with the repository.
- Codex exposes verifiable evidence through terminal logs, test outputs, and citations.
- OpenAI explicitly says agents perform best with configured dev environments, reliable tests, and clear project documentation such as `AGENTS.md`.
- They describe two interaction modes that should converge over time: real-time pairing and longer task delegation.

## Implication For simsh
- Kernel design should assume that observability is a first-class feature, not an afterthought.
- `ExecutionResult`, `ExecutionTrace`, and project conventions are all part of agent effectiveness.
