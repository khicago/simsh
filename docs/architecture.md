# simsh Architecture Plan

## Goal
Build a production-ready command runtime for agentic services with explicit safety controls and AI-friendly filesystem semantics.

## Core Package Model
- `pkg/contract`: stable interfaces and shared types
- `pkg/sh`: shell runtime (`parser + planner + executor + builtin dispatch`)
- `pkg/fs`: filesystem runtime (virtual zones + metadata + safety boundaries)
- `pkg/engine/runtime`: runtime composition (`sh + fs + policy/profile`)

## Entry Adapters
- `pkg/cmd`: runtime entry adapters (`TUI + CLI helpers`).
- `pkg/service/httpapi`: HTTP execute endpoint.
- `cmd/simsh-cli`: local runner (TUI + one-shot + `serve -P`).
- `cmd/simshd`: server runner.

## Supporting Layers
- `cmd/simsh-doc`: root document generator.

## Entry Adapter Prioritization
- `CLI`/`TUI` and HTTP are important integration surfaces, but they are not the kernel SSOT for architecture decisions.
- Near-term review and refactor work should prioritize the runtime core: `pkg/contract`, `pkg/sh`, `pkg/fs`, and `pkg/engine/runtime`.
- Entry adapters should stay thin wrappers over the unified runtime stack and should not become the place where product semantics or trust-boundary rules are invented first.
- Later optimization work for entry adapters should focus on:
  - keeping CLI and HTTP behavior aligned with core `ExecutionResult` / `ExecutionTrace` contracts;
  - reducing adapter-local policy/root override semantics that can drift away from the kernel model;
  - improving boundary/integration regression coverage without letting adapter concerns dominate kernel design.

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
  - `/sys/bin`: system builtins
  - `/bin`: custom external commands (injected bridge)
  - `/test`: optional regression corpus mount
  - synthetic parent mount directories are exposed (e.g. `/sys` for `/sys/bin`)
  - mount-backed paths are immutable for write/mkdir/remove and move/copy flows
- Command introspection:
  - `tree`: renders ASCII directory hierarchy (supports hidden/depth filters)
  - `pwd`: prints current runtime root
  - `which` / `type`: resolve command path and source (`alias` vs `builtin` vs `external`)
  - `frontmatter`: stat/get/print markdown frontmatter across multiple files
  - `rmdir`: removes empty directories only
  - generic command aliases are supported (`ll` -> `ls -l`, `fm` -> `frontmatter`)
  - runtime rc bootstrap supports read-only mounted config files with `export` and `alias`
- Manual system:
  - SSOT path: `pkg/builtin/commands/*/manual.md` (single source; remove duplicated `pkg/builtin/manuals/*`)
  - `man <cmd>` returns concise summary + `Use-When` / `Avoid-When`
  - `man -v <cmd>` renders full manual with frontmatter stripped
- Path access SSOT + `ls -l`/API formats: `docs/architecture-path-access-metadata.md`
- Memory/skills business-layer extension plan: `docs/architecture-memory-skills-extension.md`

## Current Status
- [x] Core package split (`sh/fs/cmd`).
- [x] Runtime composition moved to `pkg/engine/runtime`; `cmd` keeps entry-facing concerns.
- [x] AI-friendly filesystem naming and zone policy.
- [x] CLI and HTTP switched to unified engine runtime stack.
- [x] Root `simsh.md` generated from package-level descriptors.
- [x] `cmd/simsh-cli` upgraded with high-quality TUI default mode.
- [x] `cmd/simsh-cli serve -P` provides web runtime access.
