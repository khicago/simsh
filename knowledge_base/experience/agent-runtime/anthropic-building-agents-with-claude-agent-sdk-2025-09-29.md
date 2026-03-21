---
title: Anthropic - Building Agents with the Claude Agent SDK
summary: Anthropic frames the core design as giving Claude a computer, with bash, file creation/editing, and search forming a reusable substrate for coding and broader digital work.
source: https://claude.com/blog/building-agents-with-the-claude-agent-sdk
date: 2025-09-29
tags:
  - anthropic
  - claude
  - runtime-primitives
---

# Anthropic: Building Agents with the Claude Agent SDK

## Experience Extracts
- Anthropic frames the core design principle as "giving Claude a computer."
- They argue that bash commands, file create/edit, and file search generalize beyond coding and become a substrate for many kinds of agents.
- The same harness that powers their coding agent is described as powering research, note-taking, and other internal agent loops.

## Implication For simsh
- The kernel should treat `file editing + search + shell execution` as the highest-value primitives.
- A strong coding kernel can become a general digital-work kernel if these primitives are trustworthy.
