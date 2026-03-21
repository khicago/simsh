---
title: OpenAI - Introducing the Codex App
summary: OpenAI frames the next scaling problem as human supervision and parallel collaboration, emphasizing project isolation, worktrees, and multiple agents operating safely on the same codebase.
source: https://openai.com/index/introducing-the-codex-app/
date: 2026-02-02
tags:
  - openai
  - codex
  - isolation
  - multi-agent
---

# OpenAI: Introducing the Codex App

## Experience Extracts
- OpenAI says the hard problem has shifted from "what agents can do" to "how humans direct, supervise, and collaborate with them at scale."
- Their answer is project-based isolation, parallel agents, and built-in worktrees so multiple agents can operate on one repo without conflicts.
- The product framing treats coding as the strongest base domain, then extends outward into broader knowledge work.

## Implication For simsh
- Multi-agent support should be built on isolation and state management, not on looser shared-state concurrency.
- The kernel does not need to become a GUI app, but it does need good primitives for isolated long-running sessions.
