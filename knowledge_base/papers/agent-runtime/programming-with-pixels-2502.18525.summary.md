---
title: Programming with Pixels summary
paper: Programming with Pixels
arxiv: 2502.18525
pdf: knowledge_base/papers/agent-runtime/programming-with-pixels-2502.18525.pdf
source: https://arxiv.org/abs/2502.18525
---

# Programming with Pixels

## Citation
- Title: Programming with Pixels: Can Computer-Use Agents Do Software Engineering?
- Version used: arXiv v2, October 3, 2025

## What It Is
- A VSCode-based environment and benchmark for evaluating computer-use agents on software engineering tasks.
- The environment supports purely visual interaction, and the study also tests what happens when text APIs are added.

## Core Thesis
- Pure computer-use agents are too weak for serious software engineering if they only click, type, and observe pixels.
- Giving direct access to a very small number of high-value APIs changes the picture dramatically.

## Main Findings
- Purely visual agents perform much worse than specialized coding agents.
- When the same agents get direct access to only two APIs, file editing and bash operations, performance jumps sharply.
- Extra IDE text APIs bring further gains.
- The paper identifies visual grounding and poor use of available environment tools as key bottlenecks.

## Why It Matters For simsh
- It strongly supports the idea that `simsh` should not try to mimic a full GUI computer.
- The highest-leverage kernel capabilities are likely a trustworthy file editing surface and a trustworthy shell surface.
- This paper is a strong argument that a compact, explicit runtime can be more useful than a visually realistic one.

## Design Takeaways
- The best kernel is not the most realistic interface; it is the interface that removes the most agent friction.
- File editing and shell execution are high-value primitives and deserve especially strong contracts.
- If `simsh` makes these two surfaces reliable, it can likely capture a large fraction of the practical value of heavier computer-use systems.
