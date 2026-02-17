---
title: Doc Co-Authoring Workflow (Guidelines)
required: false
sop:
  - Use this workflow when drafting substantial docs (proposals, specs, decision docs, RFCs).
  - Update this doc when your team changes its doc-writing workflow or quality bar.
  - Regenerate `docs/must-sop.md` after SOP/frontmatter changes.
---

# Doc Co-Authoring Workflow (Guidelines)

This is an optional workflow for collaboratively writing docs that work for readers (humans and agents).

Use it when the doc is non-trivial (new design/spec/decision) and you want to reduce ambiguity and context gaps.

## Stage 1: Context Gathering
Goal: close the gap between what authors know and what readers/agents know.

Recommended steps:
1) Clarify meta-context:
   - Doc type (spec/decision/proposal), audience, desired impact, constraints.
2) Dump context first (unstructured is OK):
   - Background, alternatives considered, constraints, stakeholders, timelines, dependencies.
3) Ask and answer 5-10 gap questions:
   - Focus on edge cases, tradeoffs, and what could go wrong.

## Stage 2: Refinement & Structure
Goal: build the doc section-by-section with deliberate iteration.

Recommended steps:
1) Agree on a section outline (or pick a template).
2) Start with the section with the most unknowns (often the core decision / technical approach).
3) For each section:
   - Clarify what must be included.
   - Draft the section.
   - Iterate with surgical edits until it reads cleanly.
4) Near the end: reread for flow, consistency, and redundancy.

## Stage 3: Reader Testing
Goal: verify the doc works without author context.

Recommended steps:
1) Generate 5-10 realistic reader questions (what would someone ask an agent after reading this?).
2) Test answers with a "fresh reader" (someone not involved, or a new agent session).
3) Fix blind spots:
   - Ambiguity, hidden assumptions, missing definitions, contradictions.

## Suggested Quality Bar
- The doc is understandable by a reader with only the repo context.
- It names the decision, rationale, scope, and exceptions explicitly.
- It links sources-of-truth (code, configs, PRs, issues) where relevant.
