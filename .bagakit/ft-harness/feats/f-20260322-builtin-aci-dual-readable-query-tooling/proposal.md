# Feat Proposal: f-20260322-builtin-aci-dual-readable-query-tooling

## Why
- The current builtin surface is still uneven as an agent-computer interface: some commands already expose high-signal or structured formats, but many still optimize for terminal tradition rather than dual readability, token efficiency, and low parse cost.
- The new requirements baseline now makes the tradeoff explicit: default output should stay readable by humans and agents, structured output should be explicit and opt-in, and pipe-friendly commands should keep their composition value.
- This feat is the implementation wave that turns those requirements into command-level contracts, tool behavior, and documentation.

## Goal
- Turn the default builtin surface into a dual-readable, high-signal, pipe-aware ACI with explicit structured modes and stronger JSON/local query tools.

## Scope
- In scope:
- feat/task plan and per-tool optimization inventory
- builtin command contract metadata and `man`/manual contract rendering
- tool-by-tool optimization for commands currently judged in need of improvement:
  - `tree`
  - `grep`
  - `find`
  - `wc`
  - `env`
  - `type`
  - `which`
  - `mkdir`
  - `touch`
  - `cp`
  - `mv`
  - `rm`
  - `rmdir`
  - `tee`
  - `sed`
- stronger structure-aware query tooling, with JSON processing as the first default structured-data target
- README/doc updates that explain default text behavior, structured flags, and pipeline tradeoffs
- Out of scope:
- turning all builtin defaults into JSON
- regressing pipe-friendly text defaults for commands that are already strong composition primitives
- broad shell-surface expansion unrelated to ACI leverage
- forcing changes into low-priority tools that the review explicitly judged acceptable as-is (`cat`, `head`, `tail`, `echo`, `date`, `sort`, `uniq`, `diff`, `cd`, `pwd`, `frontmatter`)

## Impact
- Code paths:
- `pkg/engine/builtin_catalog.go`
- `pkg/contract/runtime_types.go`
- `pkg/builtin/op_help_manual.go`
- selected `pkg/builtin/op_*.go` implementations per tool
- selected `pkg/builtin/commands/*/manual.md` manuals per tool
- `pkg/builtin/coverage_test.go` and new focused builtin/engine tests as needed
- `README.md`
- requirements/review/backlog docs under `docs/`
- Tests:
- `go test ./pkg/builtin ./pkg/engine`
- targeted regression coverage per optimized tool
- whole-tree `go test ./...` before final close-out
- Rollout notes:
- start with feat/docs baseline and task inventory
- land one cross-cutting metadata/manual infrastructure task before tool-specific tasks
- then optimize tools one by one with isolated commits and explicit review
- finish with aggregate docs/README sync after all tool commits land

## Tool Inventory

### Cross-cutting infrastructure
- command metadata contract fields
- `man` summary/list rendering
- manual contract sections

### Inspection / query tools
- `tree`
  - replace ASCII-first default bias with a higher-signal dual-readable default
  - add explicit structured mode
- `grep`
  - keep pipe-friendly text default
  - add explicit structured record mode
- `find`
  - keep path-per-line default
  - add structured record mode for planning/filtering
- `wc`
  - replace unlabeled positional default with a more readable compact form
  - add explicit structured mode
- `env`
  - improve list-like variable readability
  - add explicit structured mode
- `type`
  - replace sentence-style output with more regular fielded output
  - add explicit structured mode
- `which`
  - keep simple path default
  - add explicit structured mode
- JSON query tooling
  - add a stronger structure-aware inspector so agents can query structured files without whole-file dumping

### Mutation / confirmation tools
- `mkdir`
- `touch`
- `cp`
- `mv`
- `rm`
- `rmdir`
- `tee`
- `sed`

Each of these should preserve low-noise default behavior when appropriate, but gain an explicit confirmation or structured summary mode so text-only harnesses do not need redundant verification reads.
