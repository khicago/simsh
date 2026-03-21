---
title: SWE-agent summary
paper: SWE-agent
arxiv: 2405.15793
pdf: knowledge_base/papers/agent-runtime/swe-agent-2405.15793.pdf
source: https://arxiv.org/abs/2405.15793
---

# SWE-agent

## Citation
- Title: SWE-agent: Agent-Computer Interfaces Enable Automated Software Engineering
- Version used: arXiv v3, November 11, 2024

## What It Is
- A software engineering agent built from an LM plus a custom agent-computer interface, or ACI.
- Evaluated on SWE-bench and HumanEvalFix.

## Core Thesis
- Agents should not be forced to use interfaces designed for humans if a better LM-facing interface can be designed.
- Interface design alone can produce large gains without changing model weights.

## Main Findings
- A small, LM-oriented action set beats direct use of a raw Linux shell.
- The custom ACI improves repository navigation, file viewing/editing, test execution, and feedback quality.
- The paper reports large gains over prior non-interactive baselines on software engineering tasks.

## Why It Matters For simsh
- This is the most direct research support for `simsh`.
- The main lesson is not "add more commands"; it is "design the environment around agent cognition".
- A good agent runtime should expose actions that are simple, guarded, and produce concise, reliable feedback.

## Design Takeaways
- Prefer LM-friendly commands over a huge human-oriented shell surface.
- Give explicit feedback for invalid edits and failed actions.
- Repository navigation and file editing should be first-class, not accidental byproducts of generic shell behavior.
- Kernel quality should be judged by whether the action surface improves agent behavior, not by shell completeness.
