---
title: OpenAI - Harness engineering: leveraging Codex in an agent-first world
summary: OpenAI frames harness engineering as moving human leverage away from manual coding and toward repository design, agent-readable knowledge, strict invariants, and feedback loops that let agents work autonomously without losing coherence.
source: https://openai.com/index/harness-engineering/
date: 2026-02-11
tags:
  - openai
  - harness
  - agentic-sandbox
  - repository-knowledge
---

# OpenAI: Harness engineering: leveraging Codex in an agent-first world

## Experience Extracts
- OpenAI describes an internal product built with no manually written code and says the human role shifted from coding to designing the environment, specifying intent, and building feedback loops.
- They argue early agent failures were usually environment failures. The recurring engineering question became: what capability is missing, and how can it be made both legible and enforceable for the agent?
- They explicitly reject one giant `AGENTS.md`. Their pattern is a short `AGENTS.md` as a map, plus a structured repository-local `docs/` tree as the durable system of record.
- They improved agent performance by making the application itself legible: per-worktree bootable instances, browser-driving capabilities, and worktree-local logs, metrics, and traces that agents can query directly.
- They optimize for agent legibility, not only human readability. If knowledge stays in chat threads, docs outside the repo, or tacit team memory, it effectively does not exist for the agent.
- They keep architecture coherent by enforcing invariants mechanically with linters, structural tests, and boundary rules, while leaving local implementation choices flexible.
- They treat throughput as a systems problem. As agent throughput rose, they favored short-lived pull requests, fewer blocking merge gates, and faster follow-up correction loops.
- They treat agent drift as an ongoing garbage-collection problem. Human taste is encoded into golden principles, quality scans, and recurring cleanup tasks that open targeted refactoring pull requests.

## Implication For simsh
- `simsh` should continue treating repository-visible docs, plans, and runtime contracts as first-class agent infrastructure rather than optional process artifacts.
- The most valuable kernel improvements are not more shell breadth, but more legible environments, stronger invariants, and richer machine-visible feedback loops.
- Default workspace design should optimize for agent legibility: explicit zones, predictable path semantics, structured traces, and recoverable failure classes.
- Higher-level harnesses around `simsh` should treat observability, execution plans, and doc-gardening as core control surfaces, not afterthoughts.
