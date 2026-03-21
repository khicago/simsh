---
title: Latent Space - Claude Code
summary: Claude Code discussion emphasizes cycle-time reduction, backlog pull-in, and agent loops that generate follow-up fixes, suggesting kernel evaluation should include outcome metrics and iterative improvement signals.
source: https://www.latent.space/p/claude-code
date: 2025
tags:
  - claude-code
  - metrics
  - iteration
---

# Latent Space Transcript: Claude Code

## Experience Extracts
- Anthropic team members discuss two internal measures they care about: cycle-time reduction and the number of features that would not have been built otherwise.
- They also describe internal loops where feedback channels trigger agent-generated fixes and PRs.

## Implication For simsh
- Kernel evaluation should include business-style outcome metrics, not only benchmark task success.
- Candidate metrics include time from task start to reviewable patch, trace completeness, review-loop closure rate, and backlog pull-in rate.
