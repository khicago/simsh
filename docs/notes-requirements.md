---
title: Requirements Baseline
required: false
sop:
  - Read this doc before changing current kernel-facing product requirements or implementation priorities.
  - Update this doc when new cross-cutting requirements become the source of truth for ongoing workstreams.
  - Regenerate `docs/must-sop.md` after SOP/frontmatter changes.
---

# Requirements Baseline

## Purpose

This document records current cross-cutting requirements that should steer active work.

It is not the architecture doc and not the execution backlog.
- Architecture docs explain stable design boundaries.
- Backlog docs explain what gets implemented next.
- This doc records the requirement-level constraints those decisions must obey.

## Current Requirement: Builtin ACI and Query Tooling

`simsh` is an agent-native runtime kernel. For the builtin command surface, that means the default requirement is not "look like Unix". The default requirement is:
- readable by humans and agents at the same time;
- high signal-to-noise by default;
- token-efficient for common inspect/search/edit loops.

### Requirement 1: Default output must stay dual-readable

For builtin commands, the default output should:
- remain understandable to a human scanning the terminal;
- remain easy for an agent to parse without heavy reconstruction;
- prefer compact fielded text over decorative terminal tradition when the latter wastes tokens.

This is a case-by-case engineering tradeoff, not a universal formatting rule. The bar is signal-to-noise, not nostalgia and not machine-only serialization.

### Requirement 2: Structured output should be explicit and opt-in

When a command naturally exposes structured records or summaries, it should also provide an explicit structured mode.

Preferred shape:
- use `--json` when the command only needs one structured variant;
- keep `--fmt json` where the command already has a broader output-family design (`text|md|json`, etc.).
- use `--fmt jsonl` for commands whose structured output is naturally a stream of flat records rather than one summary object.

The structured mode should:
- expose the same semantics as the default output, not a separate hidden contract;
- be stable enough for agent branching and downstream tooling;
- remain non-invasive to the default shell experience;
- avoid forcing the default output to become machine-only.

Practical interpretation:
- `--json` fits object-style summaries such as counts, lookup summaries, or variable snapshots.
- `--fmt jsonl` fits record streams such as search matches or discovered paths.
- the distinction is intentional; it follows data shape, not naming taste.

### Requirement 3: Pipeline composability remains a first-class constraint

Agents may combine commands just like experienced shell users do. Builtin redesign should therefore preserve efficient pipeline composition where that composition is already natural.

The practical rule is case-by-case:
- if a command is already naturally pipe-friendly, keep the default output easy to compose in pipes and add structured output through an explicit flag;
- if a command is not naturally pipe-friendly and mainly serves as an inspection view, prioritize higher-signal dual-readable default output instead of preserving terminal tradition for its own sake.

Examples:
- `tree` is a good candidate for a better dual-readable default because it is not a strong pipe primitive anyway;
- `grep`, `find`, and similar commands should gain structured modes without losing their default pipe-usable text behavior.

### Requirement 4: Query tools matter more than full-file dumping

The runtime should help agents read only the needed part of data, not repeatedly dump entire files and parse them in-model.

This means the builtin surface should strengthen:
- structure-aware query tools for structured files, especially JSON;
- local search tools that let agents find relevant slices with low token cost;
- targeted extraction flows over generic `cat`-everything workflows.

The `frontmatter` builtin is the current positive reference point for this direction.
The first parallel JSON-focused tool should follow the same philosophy: inspect structure and extract subtrees without turning core builtins into a general-purpose query language.

### Requirement 5: Current wave should be guided by these rules

The current builtin ACI wave should therefore prioritize:
1. explicit command-contract metadata and manual contract sections;
2. dual-readable high-signal defaults for the worst low-signal commands;
3. explicit structured modes such as `--json` or existing `--fmt json` where the data shape naturally supports it;
4. preservation of pipe-friendly default behavior where command composition is already a strength;
5. stronger JSON processing and local search/query tools;
6. low-noise confirmation modes for mutations where silence causes excess verification traffic.

## Decision Rule

When evaluating a builtin change, ask:
- does it improve default signal-to-noise?
- does it preserve dual readability?
- if structured data exists, is there an explicit structured mode?
- does it preserve or improve pipeline composability where that matters?
- does it reduce the need to dump and re-parse whole files?

If the answer is no, it probably does not belong in the current builtin ACI wave.
