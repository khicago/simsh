---
title: Project Charter
required: false
sop:
  - Read this doc before major architecture or product-boundary changes.
  - Update this doc when project goals, scope, or non-goals change.
  - Regenerate `docs/must-sop.md` after SOP/frontmatter changes.
---

# Project Charter

## Why This Exists
This document defines what `simsh` is, what it is not, and the design principles that should stay stable across iterations.

## Goals
- Provide a deterministic command runtime for agent systems.
- Expose an AI-friendly filesystem model with explicit path semantics and safety boundaries.
- Offer embeddable policy, audit, and extension hooks without coupling core runtime behavior to a single product domain.
- Keep the runtime small enough to serve as a reusable lightweight sandbox for higher-level agent platforms.

## Non-Goals
- Becoming a full POSIX shell or a general-purpose container runtime.
- Embedding domain-specific workflows, memory models, or product semantics directly into core packages.
- Treating any single reference workload as the source of truth for all runtime abstractions.
- Solving platform orchestration concerns such as multi-agent scheduling, business task planning, or product UI flows inside core runtime code.

## Scope and Boundaries
- Core runtime owns command execution semantics, filesystem projection boundaries, policy/profile enforcement, path metadata, structured execution contracts, and session primitives.
- Business/platform layers own domain adapters, RPC-to-file projections, memory curation logic, indexing/retrieval systems, and product-facing workflows.
- Reference workloads may pressure-test abstractions, but they do not justify hard-coding domain-specific namespaces or behavior into core runtime contracts.

## Principles
- Determinism over shell completeness.
- Explicit contracts over implicit magic.
- Generic core, opinionated adapters.
- Auditability and side-effect clarity by default.
- Filesystem projection is an integration surface, not a hidden implementation detail.
- Promote abstractions only after they survive at least one real adapter-backed workload.

## Success Criteria
- An embedder can project external systems into `simsh` with stable and explainable path semantics.
- An agent can reason about allowed operations and execution side effects without trial-and-error.
- Core contracts remain reusable across multiple workloads even when one workload is used as an early pressure test.
- Architecture changes can be evaluated against clear boundaries instead of product-specific intuition alone.

## Change Process
- Record durable scope or boundary changes here before spreading them across architecture docs.
- Put stable contract details in focused architecture docs, not in this charter.
- Validate major contract changes with tests and at least one adapter-backed end-to-end workload before declaring them stable.
