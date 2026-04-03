---
title: Runtime Comparables and Benchmark Fit Study
summary: Decision-oriented study of directly relevant runtime implementations and benchmark families for simsh after the current adapter-seam proof wave, ending with one recommended next feat and explicit rejected alternatives.
date: 2026-04-04
tags:
  - research
  - agent-runtime
  - benchmark
  - planning
---

# Scope

This study is intentionally narrow.

It answers one question:

> Given the current `simsh` stage, what is the next highest-value wave: another runtime feature wave, or an evaluation/benchmark-fit wave?

It is **not** a market survey and **not** a new product roadmap.

# Current Baseline

`simsh` has already closed a dense adapter-side proof wave:

- second adapter seam validation
- shared seam conformance
- shared mount conformance
- direct helper self-coverage
- composition/evolution stress validation

The strategic problem has therefore changed.

The main risk is no longer "can this adapter seam be made coherent?"
The main risk is now:

- whether the next investment targets the right pressure point;
- whether the repo is evaluating itself against the right external task families;
- whether the runtime remains aligned with frontier engineering practice without drifting into unnecessary product scope.

# Inputs

This study builds on existing local materials:

- `task_outputs/research/next-phase-runtime-benchmark-scout-2026-04-04.md`
- `task_outputs/research/simsh-current-phase-optimization-plan-2026-03-31.md`
- `task_outputs/research/frontier-agent-team-blog-podcast-notes-2026-03-22.md`
- `knowledge_base/papers/agent-runtime/swe-agent-2405.15793.summary.md`
- `knowledge_base/papers/agent-runtime/programming-with-pixels-2502.18525.summary.md`
- `knowledge_base/papers/agent-runtime/osworld-2404.07972.summary.md`
- `knowledge_base/papers/agent-runtime/researchenvbench-2603.06739v2.summary.md`

External sources used to close gaps:

- SWE-ReX: https://github.com/SWE-agent/SWE-ReX
- OpenHands runtime docs:
  - https://docs.openhands.dev/openhands/usage/runtimes/overview
  - https://docs.all-hands.dev/openhands/usage/architecture/runtime
  - https://docs.openhands.dev/sdk/guides/agent-server/api-sandbox
- Terminal-Bench:
  - https://www.tbench.ai/
  - https://www.tbench.ai/news/registry-and-adapters
- SWE-bench-Live: https://arxiv.org/abs/2505.23419
- EnvBench: https://arxiv.org/abs/2503.14443
- ResearchEnvBench: https://arxiv.org/abs/2603.06739
- SWE-Skills-Bench: https://arxiv.org/abs/2603.15401

# Comparable Runtime Study

## Comparison Table

| System | What it is | Runtime ownership model | Environment model | Extensibility seam | Observability / evidence model | Main implication for simsh |
| --- | --- | --- | --- | --- | --- | --- |
| `simsh` (current) | Lightweight agent runtime kernel with explicit adapter seam | Runtime owns execution truth; adapters own projection/control-plane truth | VFS-style deterministic workspace with read-only projections and managed `/memory` | `SessionAdapter`, `AdapterProjection`, `VirtualMount`, `ExecutionTrace` | Strong local traces, adapter projections, benchmark-visible evidence | Already strong on seam explicitness and proof layering |
| SWE-ReX | Reusable runtime framework below the agent loop | Runtime infrastructure is reusable; agent logic sits above it | Shell/session-oriented execution backend abstraction | Backend/runtime layer intended to stay generic | Strong emphasis on execution infrastructure reuse and orchestration below agent logic | Reinforces not coupling simsh to one agent loop or one richer adapter |
| OpenHands runtime | Product runtime/sandbox layer for coding agents | Runtime and sandbox are first-class platform services | Local / container / remote runtime split | Clear boundary between runtime, sandbox, and higher orchestration | Operationally stronger on sandbox/runtime API separation | Suggests simsh should stay explicit about environment ownership and lifecycle boundaries |
| SWE-agent | ACI-centric SWE agent | Agent loop tightly shaped by custom ACI | Task environment is still repo/task oriented | ACI is the main seam | Strong action/feedback design emphasis | Still the strongest support for LM-legible action surfaces over shell breadth |

## Strongest Cross-System Takeaways

### 1. Runtime explicitness still matters more than surface breadth

This is consistent across `simsh`, SWE-ReX, OpenHands, and SWE-agent:

- the runtime must own a clear boundary;
- the environment must be predictable;
- the agent loop must not be allowed to smuggle hidden semantics into the runtime.

This remains exactly the right direction for `simsh`.

### 2. The next frontier pressure is orchestration/evaluation, not raw primitive count

None of the strongest comparables suggest that `simsh` should now add a lot more shell primitives.

Instead, they suggest:
- clearer evaluation and benchmark mapping;
- stronger runtime-to-orchestrator boundaries;
- more explicit evidence surfaces for review and asynchronous execution.

### 3. `simsh` is already unusually strong on proof layering

Relative to many open systems, `simsh` now has a rarer property:
- seam proof
- mount proof
- helper proof
- composition proof

That is a real differentiator.

It means the next move should exploit this strength, not reset back to feature expansion.

# Benchmark Fit Study

## Fit Matrix

| Benchmark | Best use for simsh | What it validates well | What it distorts if adopted too literally | Fit now |
| --- | --- | --- | --- | --- |
| Terminal-Bench | Primary external comparison target | Terminal-oriented agent behavior, shell/file task realism, CLI-centric workflows | Assumes a fuller terminal environment than simsh should necessarily emulate | High |
| SWE-bench-Live | Dynamic workload reference | Drift, changing tasks, non-static SWE evaluation | Centers issue-resolution agent performance more than runtime substrate quality | Medium-high |
| EnvBench | Boundary comparator | Environment setup and environment drift/setup exposure | Pushes toward environment synthesis, which is not simsh's current core scope | Medium |
| ResearchEnvBench | Long-horizon comparator | Environment synthesis for research execution | Strongly pulls toward environment construction rather than runtime truth surfaces | Medium-low now |
| SWE-Skills-Bench | Targeted secondary comparator | Whether skill-related capability really changes SWE outcomes | Narrower than simsh's full runtime scope; not a full-system benchmark | Medium |
| OSWorld | Context reference only | Why full environment realism matters | Too GUI-heavy and too broad for simsh's current shape | Low |
| Programming with Pixels | Design-justification reference | Reinforces file editing + shell as high-value surfaces | Better as thesis support than direct benchmark target | Low as benchmark, high as design evidence |

## Main Benchmark Judgment

The strongest current fit is:

### 1. Terminal-Bench as the closest external benchmark family

Why:
- `simsh` is closest to a terminal/runtime substrate.
- It can borrow task shapes and evaluation pressure from terminal-oriented benchmarks without pretending to be a full desktop/GUI system.

### 2. SWE-bench-Live as the dynamic-workload north star

Why:
- It pressures drift and changing conditions, which is strategically relevant to the next simsh wave.
- It is less a direct runtime benchmark and more a reality check for whether a runtime helps with evolving tasks.

### 3. EnvBench / ResearchEnvBench as boundary references, not immediate goals

Why:
- They help define what `simsh` should explicitly **not** try to solve yet.
- They are useful mainly to keep scope disciplined.

# Synthesis

## What simsh should not do next

The study does **not** support these directions as the next wave:

1. Add many more shell or product nouns.
2. Add a third adapter just to prove genericity again.
3. Adopt a full benchmark suite immediately.
4. Expand toward environment synthesis.
5. Start building registry / marketplace / remote sync semantics.

## Why those are wrong now

- They would spend the newly earned design slack on breadth instead of leverage.
- They would dilute the strongest current asset: explicit, layered proof around runtime truth.
- They would likely increase maintenance faster than signal.

# Recommended Next Feat

## Recommendation

**K-027: external benchmark mapping and evaluation-feasibility wave**

### Goal

Map current `simsh` proof layers onto one or two external benchmark families, starting with Terminal-Bench as the closest fit and SWE-bench-Live as the dynamic-workload reference.

### What this wave should do

- Define which existing `simsh` scenarios correspond to which external task families.
- Identify what can be evaluated **as-is**, what requires translation, and what should be explicitly excluded.
- Add one or two translation/evaluation artifacts, not a full benchmark adoption.
- Produce a decision on whether simsh should:
  - stay on its own native benchmark suite only,
  - add a lightweight external comparison layer,
  - or prepare a later, larger evaluation wave.

### Why this is the best next move

- It is the natural next step after the current proof wave.
- It keeps the system frontier-aligned without driving feature sprawl.
- It transforms the current implementation maturity into external evaluation clarity.
- It lets future investment be governed by evidence rather than taste.

## Rejected Alternatives

### Rejected 1: A third adapter wave

Reason:
- Current genericity has already been proven enough for now.
- Another adapter would likely teach less than better evaluation mapping.

### Rejected 2: Immediate concurrent-session/orchestration feature wave

Reason:
- Important, but still one level too implementation-oriented.
- Better to first understand what evaluation pressure actually justifies it.

### Rejected 3: Environment-synthesis expansion

Reason:
- Too far from current `simsh` scope.
- Would blur the runtime-truth thesis that currently makes the system distinctive.

# Non-Goals

- Do not adopt a large benchmark suite wholesale in the next wave.
- Do not add new agent product semantics.
- Do not widen `simsh` into a full environment-builder.
- Do not weaken existing native benchmark gates in order to appear benchmark-compatible.

# Bottom Line

The current `simsh` design is still aligned with frontier practice.

The strongest next move is **not** more primitives.
It is **not** more product semantics.
It is **not** more adapters.

It is:

**use the current proof maturity to decide and shape the right external evaluation relationship.**

That means the next wave should be:

**benchmark mapping and evaluation feasibility**, led by Terminal-Bench fit and informed by SWE-bench-Live as the dynamic-workload reference.
