---
title: Next-Phase Runtime and Benchmark Scout
summary: Narrow research brief for the next simsh wave, focused on comparable runtime implementations and benchmark candidates that match the current adapter-seam and composition-proof stage.
date: 2026-04-04
tags:
  - research
  - agent-runtime
  - benchmark
  - planning
---

# Why Now

`simsh` has already closed the current adapter-seam proof wave:

- shared seam conformance
- shared mount conformance
- helper self-coverage
- composition/evolution stress validation

That means the next risk is no longer "can we build the seam?" but:

- are we still aligned with frontier runtime design?
- are we evaluating the right things?
- which benchmark family should shape the next investment?

This is the right point to run a **narrow experimental research team**, not a broad literature review.

# Main Judgment

Yes: a short, problem-driven research team is warranted now.

But it should be scoped to one question:

> What is the next highest-value runtime pressure point for simsh, given the current seam-proof stage and the benchmark landscape?

The wrong move would be to launch a general "AI runtime research" effort.
The right move is a bounded team that compares a few directly relevant systems and benchmarks, then produces a concrete next-wave recommendation.

# Comparable Implementations To Study

## 1. SWE-ReX

Why it matters:
- It is very close to `simsh` in spirit: reusable runtime infrastructure below the agent loop.
- It emphasizes sandboxed shell execution, parallelism, and backend-agnostic runtime composition.

What to study:
- how much state is held in the runtime vs above it;
- how they treat shell-session realism;
- how they keep the runtime reusable across agent loops.

Source:
- https://github.com/SWE-agent/SWE-ReX

## 2. OpenHands Runtime/Sandbox

Why it matters:
- It is one of the clearest open implementations of a runtime/sandbox layer used by a coding agent platform.
- It shows how a larger system separates sandbox/runtime concerns from agent orchestration.

What to study:
- runtime API surface;
- local vs docker vs remote runtime split;
- where environment ownership and lifecycle boundaries sit.

Sources:
- https://docs.openhands.dev/openhands/usage/runtimes/overview
- https://docs.all-hands.dev/openhands/usage/architecture/runtime
- https://docs.openhands.dev/sdk/guides/agent-server/api-sandbox

## 3. SWE-agent

Why it matters:
- The most direct research support for `simsh` remains its ACI thesis: interface quality beats raw shell breadth.
- It is still the cleanest "agent-computer interface for SWE" reference.

What to study:
- how benchmark assumptions shape interface design;
- which actions are treated as first-class;
- where their setup differs from `simsh`'s lighter runtime philosophy.

Sources:
- https://arxiv.org/abs/2405.15793
- local summary: `knowledge_base/papers/agent-runtime/swe-agent-2405.15793.summary.md`

# Benchmark Shortlist

## Primary Candidates

### 1. Terminal-Bench

Why it is a strong fit:
- It directly targets terminal-oriented agent behavior.
- `simsh` is much closer to a terminal/runtime substrate than to a full GUI system.
- It is the most obvious external benchmark family for "terminal mastery under real task structure".

Why it is not a perfect fit:
- `simsh` is not a full real terminal VM substitute.
- Some Terminal-Bench tasks may assume heavier environment semantics than `simsh` should natively emulate.

How to use it:
- as an external comparison target;
- as inspiration for task shapes;
- not as a drop-in product metric.

Sources:
- https://www.tbench.ai/
- https://www.tbench.ai/news/registry-and-adapters
- arXiv pointer surfaced in search: `TERMINAL-BENCH` / arXiv:2601.11868

### 2. SWE-bench-Live

Why it matters:
- It pressures dynamic, changing, contamination-resistant software tasks.
- It is much better than static SWE-bench variants for asking whether an agent/runtime stack survives drift.

Why it is only partial fit:
- It evaluates repository issue-resolution agents more than runtime substrates.
- It is useful as a north-star workload family, not as a pure runtime benchmark.

How to use it:
- as evidence for why dynamic evaluation matters;
- as a source of task realism patterns;
- potentially as a future mapping target if simsh grows a stronger repo-task loop.

Sources:
- https://arxiv.org/abs/2505.23419
- https://openai.com/index/introducing-swe-bench-verified/
- https://openai.com/index/why-we-no-longer-evaluate-swe-bench-verified/

### 3. EnvBench

Why it matters:
- It specifically targets automated environment setup.
- This is relevant if the next simsh question becomes "how much environment drift/setup should be exposed or synthesized?"

Why it is not the immediate best fit:
- `simsh` currently tries to make drift visible and manageable, not synthesize full environments.

How to use it:
- as a boundary benchmark;
- to define what simsh explicitly does **not** aim to solve.

Source:
- https://arxiv.org/abs/2503.14443

### 4. ResearchEnvBench

Why it matters:
- It explicitly targets environment synthesis for research code execution.
- It is highly relevant if simsh eventually wants stronger scientific/research runtime positioning.

Why it is not immediate core fit:
- It sits closer to environment construction than current simsh scope.

How to use it:
- as a longer-horizon comparator;
- to stress-test whether simsh should stay "runtime truth + drift visibility" rather than expand toward synthesis.

Source:
- https://arxiv.org/abs/2603.06739

## Secondary Candidates

### 5. SWE-Skills-Bench

Why it matters:
- The repo now has an explicit `/skills` story and real skill-selection/control-plane semantics.
- This benchmark is unusually aligned with the question "do skills actually matter?"

How to use it:
- not as a whole-system benchmark;
- as a targeted external reference for the skills layer.

Source:
- https://arxiv.org/abs/2603.15401

## Context-Only References

### OSWorld
- Useful for understanding why full realism matters.
- Not a primary fit for current simsh because it is much broader and more GUI-centric.
- Source: https://arxiv.org/abs/2404.07972

### Programming with Pixels
- Useful because it reinforces the exact simsh thesis: file editing + shell beats full computer-use realism for SWE.
- Better as a design justification reference than as a benchmark target.
- Source: https://arxiv.org/abs/2502.18525

# Recommended Experimental Team Shape

## Lane 1: Comparable Runtime Study

Question:
- What boundaries do SWE-ReX and OpenHands keep between runtime, sandbox, and agent loop?

Deliverable:
- one comparison table:
  - runtime ownership
  - session model
  - environment model
  - extensibility seam
  - observability surface

## Lane 2: Benchmark Mapping

Question:
- Which benchmark family best matches simsh's current stage?

Deliverable:
- one shortlist with:
  - fit to current simsh scope
  - what it would validate
  - what it would distort if adopted too literally

## Lane 3: Synthesis

Question:
- Given the above, what should the next simsh feat actually be?

Deliverable:
- one recommended next-wave item
- one rejected alternative list
- one explicit non-goal list

# Most Likely Next-Wave Direction

Current best guess:

**Do not** start with full benchmark adoption.

Instead:
- compare `simsh` to runtime comparables;
- map benchmark families;
- then decide whether the next wave is:
  - better environment-drift visibility;
  - stronger review-loop evidence;
  - scoped concurrent session/isolation work;
  - or benchmark-mapping work itself.

The strongest current candidate is still:

**a benchmark-mapping / evaluation-feasibility wave**, not another raw feature wave.

# Bottom Line

Yes, open an experimental team now.

But keep it narrow:
- not "survey the field"
- not "add more features"
- instead:
  - compare 2-3 directly relevant runtimes
  - shortlist benchmark families
  - synthesize one concrete next feat

If this turns into open-ended research, it will slow the repo down.
If it stays decision-oriented, it will probably be the highest-leverage next move.
