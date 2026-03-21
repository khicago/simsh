---
title: Cognition - How Cognition Uses Devin to Build Devin
summary: Cognition describes a workflow that separates codebase Q&A and scoping from execution, stresses well-scoped tasks and reusable playbooks, and says review has become a bigger bottleneck than generation.
source: https://cognition.ai/blog/how-cognition-uses-devin-to-build-devin
date: 2026-02-27
tags:
  - cognition
  - devin
  - scoping
  - review
---

# Cognition: How Cognition Uses Devin to Build Devin

## Experience Extracts
- Cognition reports using Devin across web, Slack, Linear, CLI, and API, depending on where work starts.
- They describe a recurring pattern: use codebase Q&A first for scoping, then launch execution with a better tailored prompt.
- They say the bottleneck shifted from writing code to reviewing it.
- They emphasize well-scoped tasks, playbooks for repeated work, MCP connections to external tools, and verification mechanisms for tasks that need strong validation.

## Implication For simsh
- The kernel should support a clean split between scoping/search and action/execution.
- Repeated successful task patterns should be codified above the kernel, but the kernel must expose enough structure to make that possible.
