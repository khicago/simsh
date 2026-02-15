# simsh

`simsh` is an agent-focused command runtime composed from three core packages:
- `sh`: shell execution semantics
- `fs`: AI-friendly filesystem
- `cmd`: runnable runtime environment

## Why
- deterministic shell subset for agents
- production integration hooks (policy + audit + external command bridge)
- purpose-oriented filesystem zones that reduce path hallucinations

## Package Layout
- `pkg/sh`: shell parser/executor environment
- `pkg/fs`: AI-friendly filesystem composition
- `pkg/engine/runtime`: runtime composition (`sh + fs + policy/profile`)
- `pkg/cmd`: runtime entry adapters (TUI + CLI helpers)
- `pkg/contract`: runtime contracts
- `pkg/service/httpapi`: HTTP execute endpoint
- `cmd/simsh-cli`: local command runtime (`TUI + CLI + serve -P`)
- `cmd/simshd`: HTTP runtime service
- `cmd/simsh-doc`: generate root `simsh.md`

## AI-Friendly Filesystem Zones
- `/task_outputs`: durable deliverables and final artifacts
- `/temp_work`: temporary intermediates
- `/knowledge_base`: read-only references

## Quick Start
```bash
# generate root runtime description
go run ./cmd/simsh-doc

# CLI one-shot
go run ./cmd/simsh-cli -profile core-strict -c 'ls -l "/"'

# CLI interactive TUI
go run ./cmd/simsh-cli

# CLI serve command
go run ./cmd/simsh-cli serve -P 18080 -root "$PWD" -profile core-strict

# Dedicated service binary
go run ./cmd/simshd -listen ':18080' -root "$PWD" -profile core-strict

curl -sS http://127.0.0.1:18080/v1/execute \
  -H 'Content-Type: application/json' \
  -d '{"command":"env PATH","profile":"core-strict","policy":"read-only"}'
```

## Runtime Features
- builtins: `ls`, `env`, `cat`, `head`, `tail`, `grep`, `find`, `echo`, `tee`, `sed`, `man`, `date`
- script operators: `;`, `&&`, `||`, `|`
- redirections: `>`, `>>`, `<`, `<<`
- profiles: `core-strict`, `bash-plus`, `zsh-lite`
- policies: `disabled`, `read-only`, `write-limited`, `full`
- mounts: `/sys/bin`, `/bin`, optional `/test`
- interactive runtime: full-screen TUI by default, fallback line REPL via `--repl`/`--no-tui`
