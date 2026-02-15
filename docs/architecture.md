# simsh Architecture Plan

## Goal
Build a production-ready command runtime for agentic services with explicit safety controls and AI-friendly filesystem semantics.

## Core Package Model
- `pkg/sh`: shell runtime (`parser + planner + executor + builtin dispatch`)
- `pkg/fs`: filesystem runtime (virtual zones + metadata + safety boundaries)
- `pkg/engine/runtime`: runtime composition (`sh + fs + policy/profile`)
- `pkg/cmd`: runtime entry adapters (`TUI + CLI helpers`)

## Supporting Layers
- `pkg/contract`: stable interfaces and shared types.
- `pkg/service/httpapi`: HTTP execute endpoint.
- `cmd/simsh-cli`: local runner (TUI + one-shot + `serve -P`).
- `cmd/simshd`: server runner.
- `cmd/simsh-doc`: root document generator.

## Filesystem Semantics
- `/task_outputs`: persistent outputs.
- `/temp_work`: temporary artifacts.
- `/knowledge_base`: read-only references.

These names are intentionally explicit so an agent can infer write intent directly from path names.

## Runtime Capability Model
- Profiles:
  - `core-strict`
  - `bash-plus`
  - `zsh-lite`
- Policies:
  - `disabled`
  - `read-only`
  - `write-limited`
  - `full`
- Mounts:
  - `/sys/bin`: builtin commands
  - `/bin`: injected external commands
  - `/test`: optional regression corpus mount

## Current Status
- [x] Core package split (`sh/fs/cmd`).
- [x] Runtime composition moved to `pkg/engine/runtime`; `cmd` keeps entry-facing concerns.
- [x] AI-friendly filesystem naming and zone policy.
- [x] CLI and HTTP switched to unified engine runtime stack.
- [x] Root `simsh.md` generated from package-level descriptors.
- [x] `cmd/simsh-cli` upgraded with high-quality TUI default mode.
- [x] `cmd/simsh-cli serve -P` provides web runtime access.
