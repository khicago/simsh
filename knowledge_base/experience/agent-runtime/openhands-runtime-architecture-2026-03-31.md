---
title: OpenHands - Runtime Architecture
summary: OpenHands treats runtime as a separable sandbox layer with explicit provider, image, mount, and plugin policies. The main lesson for simsh is to keep runtime/provider seams explicit and make environment freshness, rebuild, and mount behavior legible instead of implicit.
source: https://docs.openhands.dev/openhands/usage/architecture/runtime
date: 2026-03-31
tags:
  - openhands
  - runtime
  - sandbox
  - adapters
---

# OpenHands: Runtime Architecture

## Experience Extracts
- OpenHands treats runtime as a dedicated layer beneath the agent rather than as incidental CLI glue.
- The same agent logic can target different sandbox providers, which keeps environment concerns separate from agent behavior.
- Their runtime docs make image build and rebuild policy explicit instead of burying it in implementation details.
- Volume and overlay behavior are documented as first-class runtime choices.
- Capability expansion is pushed into runtime plugins rather than mixed into the agent loop itself.

## Implication For simsh
- `simsh` should keep the reference adapter and future harness-facing providers explicit at the adapter boundary rather than letting product semantics leak into core packages.
- Projection freshness and refresh behavior should be visible in projected state, just as OpenHands makes image and mount policy visible at the runtime layer.
- If `/memory`, `/resources`, or future `/skills` projections become more dynamic, the lifecycle needs to be legible to the agent and to validation code, not hidden behind side effects.
