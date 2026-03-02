# simsh

> Deterministic command runtime for agent systems.

`simsh` is a lightweight sandbox for agent execution. It combines a constrained shell runtime, an AI-friendly virtual filesystem, and embeddable policy/audit hooks into one reusable core.

It is designed for higher-level agent platforms that need a predictable execution surface. It is not trying to be a full POSIX shell, a container runtime, or a product-specific workflow engine.

## Status

`simsh` is experimental, but the current baseline already includes:

- local CLI and interactive TUI execution
- an HTTP `/v1/execute` runtime service
- policy/profile-gated builtin commands
- AI-friendly virtual filesystem zones
- path metadata via `ls -l` and opt-in API metadata
- a documented extension boundary for adapter-driven mounts
- structured execution result contracts
- execution tracing with side-effect tracking
- adapter lifecycle and memory protocol hooks
- first-class session lifecycle management

Project boundaries and non-goals live in [`docs/notes-project-charter.md`](docs/notes-project-charter.md).

## When to use simsh

Use `simsh` when you need:

- a smaller and more inspectable execution model than a general-purpose shell
- explicit filesystem zones for source material, scratch work, and durable outputs
- policy/profile enforcement that can be surfaced to agents, adapters, and APIs
- a reusable runtime core that stays separate from product-specific orchestration

Choose something else if you need:

- broad POSIX compatibility
- container- or VM-style isolation guarantees
- a workflow engine with built-in domain semantics

## Why simsh

General-purpose shells are powerful, but they are noisy execution environments for agents:

- too much ambient state
- weak path intent signaling
- unclear write boundaries
- hard-to-consume results for planning loops

`simsh` narrows the runtime on purpose:

- **deterministic shell subset** for predictable execution
- **purpose-oriented virtual paths** so agents can infer intent from names
- **explicit policy and profile controls** instead of implicit host behavior
- **filesystem metadata and mount semantics** that make capabilities visible
- **generic core + adapter boundary** so business logic stays out of the runtime kernel

## Architecture At A Glance

```mermaid
flowchart TB
  platform["Agent Platform"] --> runtime["simsh Runtime"]
  runtime --> shell["Deterministic shell subset"]
  runtime --> fs["Virtual filesystem"]
  runtime --> controls["Policy, profile, and audit hooks"]
  fs --> kb["/knowledge_base"]
  fs --> out["/task_outputs"]
  fs --> tmp["/temp_work"]
  fs --> sys["/sys"]
  adapters["Platform adapters"] --> fs
  adapters --> adapter_logic["RPC projections, memory, indexing, product logic"]
```

## Core Ideas

### 1. Purpose-oriented filesystem zones

The virtual root exposes only a small set of high-signal directories:

| Path | Purpose |
| --- | --- |
| `/knowledge_base` | source-oriented reference material and mirrored external artifacts |
| `/task_outputs` | durable deliverables and final agent-authored artifacts |
| `/temp_work` | temporary intermediates, scratch output, and disposable state |
| `/sys` | virtual runtime metadata |

These names are intentionally explicit so an agent can reason about where output belongs before it writes. Writeability is still controlled by the active policy.

### 2. Deterministic execution model

`simsh` supports a focused shell subset rather than emulating all of Bash:

- operators: `;`, `&&`, `||`, `|`
- redirections: `>`, `>>`, `<`, `<<`
- profiles: `core-strict`, `bash-plus`, `zsh-lite`
- policies: `disabled`, `read-only`, `write-limited`, `full`

### 3. Agent-visible path semantics

The runtime exposes path metadata instead of forcing agents to probe by failure:

- `ls -l` shows mode, access, semantic kind, line counts, and path
- `ls -l --fmt json` returns machine-readable entries with capabilities
- `/v1/execute` supports `include_meta=true` for opt-in path metadata

### 4. Generic core, adapter-driven extensions

`simsh` keeps domain logic out of core packages. Memory systems, resource projections, and skill-like trees are expected to live behind adapter-driven `VirtualMount` integrations rather than being hard-coded into the kernel.

See [`docs/architecture-memory-skills-extension.md`](docs/architecture-memory-skills-extension.md) for the boundary.

## Quick Start

### Requirements

- Go `1.22+`

### Build and test

```bash
go test ./...
go build ./cmd/simsh-cli
go build ./cmd/simshd
```

### Run locally

```bash
# one-shot command
go run ./cmd/simsh-cli -profile core-strict -c 'ls -l "/"'

# interactive TUI
go run ./cmd/simsh-cli

# line-based REPL
go run ./cmd/simsh-cli --no-tui

# HTTP runtime service
go run ./cmd/simsh-cli serve -P 18080 -root "$PWD" -profile core-strict

# dedicated HTTP binary
go run ./cmd/simshd -listen ':18080' -root "$PWD" -profile core-strict
```

You can also use the included `Makefile`:

```bash
make test
make check
make cli
make cli-c CMD='ls -l /'
make cli-serve PORT=18080
make simshd
```

## Examples

### Inspect the virtual root

```bash
go run ./cmd/simsh-cli -profile core-strict -c 'ls -l "/"'
```

Example output:

```text
d ro knowledge_dir - /knowledge_base
d ro virtual_dir - /sys
d ro task_output_dir - /task_outputs
d ro temp_work_dir - /temp_work
# columns: mode access kind lines path
```

### Call the HTTP API with metadata

```bash
curl -sS http://127.0.0.1:18080/v1/execute \
  -H 'Content-Type: application/json' \
  -d '{
    "command": "ls -l /knowledge_base",
    "profile": "core-strict",
    "policy": "read-only",
    "include_meta": true
  }'
```

Example response:

```json
{
  "output": "# columns: mode access kind lines path",
  "exit_code": 0,
  "meta": {
    "paths": [
      {
        "mode": "d",
        "access": "ro",
        "kind": "knowledge_dir",
        "lines": -1,
        "path": "/knowledge_base",
        "capabilities": ["describe", "list", "search"]
      }
    ]
  }
}
```

## Builtin Commands

Current builtins are grouped around inspection, text processing, and safe file mutation:

- inspection: `ls`, `tree`, `pwd`, `env`, `which`, `type`, `man`, `frontmatter`, `date`
- text processing: `cat`, `head`, `tail`, `grep`, `find`, `diff`, `sort`, `uniq`, `wc`, `sed`
- file operations: `mkdir`, `cp`, `mv`, `rm`, `rmdir`, `touch`, `tee`

The generated runtime profile in [`simsh.md`](simsh.md) contains the fuller command reference and examples.

## Package Layout

- `pkg/sh`: shell parsing and execution semantics
- `pkg/fs`: AI-friendly filesystem composition and adapter glue
- `pkg/engine/runtime`: runtime assembly (`sh + fs + policy/profile`)
- `pkg/contract`: stable contracts and shared types
- `pkg/service/httpapi`: HTTP execute endpoint
- `pkg/cmd`: CLI/TUI-facing runtime entry helpers
- `cmd/simsh-cli`: local runtime (`CLI + TUI + serve`)
- `cmd/simshd`: dedicated HTTP service
- `cmd/simsh-doc`: generator for `simsh.md`

## What simsh is not

`simsh` intentionally does not try to solve everything inside the runtime kernel.

Non-goals include:

- full POSIX shell compatibility
- container or VM-style isolation
- hard-coded product workflows
- domain-specific memory or orchestration semantics in core packages
- treating one reference workload as the source of truth for all abstractions

## Documentation

Start here:

- [`simsh.md`](simsh.md): generated runtime profile
- [`docs/notes-project-charter.md`](docs/notes-project-charter.md): project goals, scope, and non-goals
- [`docs/architecture.md`](docs/architecture.md): current architecture overview
- [`docs/architecture-path-access-metadata.md`](docs/architecture-path-access-metadata.md): path metadata and listing/API formats
- [`docs/architecture-memory-skills-extension.md`](docs/architecture-memory-skills-extension.md): extension boundary for mounts and business-layer systems

Next-stage design docs:

- [`docs/architecture-session-trace-model.md`](docs/architecture-session-trace-model.md): planned session and execution trace contracts
- [`docs/architecture-platform-adapter-contract.md`](docs/architecture-platform-adapter-contract.md): platform adapter, memory lifecycle, and projection seams
- [`docs/notes-v0-1-0-to-v0-2-migration.md`](docs/notes-v0-1-0-to-v0-2-migration.md): migration plan from the `v0.1.0` baseline to the planned `v0.2` contract set

Historical context:

- [`docs/first_version_plan.md`](docs/first_version_plan.md): v1 implementation history and completed scope

## Development

Common commands:

```bash
make test
make test-race
make lint
make check
make doc
```

`make doc` regenerates [`simsh.md`](simsh.md) from the current runtime description.

## Contributing

The runtime is still experimental, so boundary discipline matters more than feature count.

Before changing core behavior:

- read [`docs/notes-project-charter.md`](docs/notes-project-charter.md) and the relevant architecture docs
- keep core contracts generic; push workload semantics into adapters unless a contract has proven reusable
- run `make check` and `go test ./...`

If a change updates SOP/frontmatter-driven docs under `docs/`, regenerate [`docs/must-sop.md`](docs/must-sop.md).

## Roadmap

The current baseline focuses on deterministic execution, virtual filesystem semantics, and policy/profile controls.

Planned next-stage work is documented and split into implementation feats around:

- first-class session lifecycle
- structured execution results
- execution traces with side-effect tracking
- adapter lifecycle and optional memory protocol hooks
- adapter-backed end-to-end validation

Those contracts are described in the architecture docs above and tracked in the feat harness under `.bagakit/ft-harness/`.
