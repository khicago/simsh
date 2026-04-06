---
title: Reusable Items - Coding (Catalog)
required: false
sop:
  - Update this list when you introduce or adopt a new reusable component/library/mechanism.
  - When you remove or deprecate something, update this list and point to the replacement or migration.
  - Regenerate `docs/must-sop.md` after SOP/frontmatter changes.
---

# Reusable Items - Coding (Catalog)

This is a project-local catalog of reusable engineering assets. The goal is discoverability and convergence.

## Reusable Components
| Item | MUST/SHOULD/NICE | When to Use | Source of Truth |
| --- | --- | --- | --- |
| Builtin command lookup resolver (`which`/`type`) | SHOULD | Need stable command resolution order (alias first, then builtin, then external) without duplicating lookup logic in each command | `pkg/builtin/op_command_lookup.go` |
| Empty-directory removal command (`rmdir`) | SHOULD | Need explicit directory-only removal semantics that differ from `rm` | `pkg/builtin/op_rmdir.go` |
| ASCII tree renderer (`tree`) | SHOULD | Need compact directory hierarchy visualization with depth/hidden controls for agent-readable context | `pkg/builtin/op_tree.go` |
| Frontmatter multi-file inspector (`frontmatter`) | SHOULD | Need batch stat/get/print for markdown frontmatter with compact/json/md output modes and context slicing | `pkg/builtin/op_frontmatter.go`, `pkg/builtin/frontmatter_helpers.go` |

## Reusable Libraries / Packages
| Item | MUST/SHOULD/NICE | When to Use | Source of Truth |
| --- | --- | --- | --- |
| `contract.Ops.RemoveDir` callback | SHOULD | Adapter must expose directory removal as first-class capability instead of overloading file remove behavior | `pkg/contract/integration_contract.go` |
| Runtime bootstrap channels (`CommandAliases`, `EnvVars`, `RCFiles`) | SHOULD | Need pluggable alias/env behavior from static config without adding write surface | `pkg/contract/integration_contract.go`, `pkg/engine/orchestrator.go` |

## Reusable Mechanisms
Examples: error handling patterns, feature flag patterns, logging/metrics conventions, migration playbooks.
| Item | MUST/SHOULD/NICE | When to Use | Source of Truth |
| --- | --- | --- | --- |
| Manual SSOT under `commands/*/manual.md` | MUST | Add/update command manuals; avoid dual-source drift between runtime embed files and command-local docs | `pkg/builtin/manuals.go`, `pkg/builtin/commands/*/manual.md` |
| `man` progressive disclosure guard | SHOULD | Need concise default manual output with escalation path to full details | `pkg/builtin/op_help_manual.go` |
| Shared text-scan core (`grep` / `rg`) | SHOULD | Need line-oriented text search with shared matcher/context/jsonl rendering while keeping command-specific parsers separate and preferring runtime `SearchContent` fast paths over enumerate-then-read loops when available | `pkg/builtin/search_scan.go`, `pkg/builtin/op_pattern_scan.go`, `pkg/builtin/op_rg.go` |
| Narrow JSON query inspector (`json`) | SHOULD | Need structure-aware JSON inspection that covers shape, subtree extraction, key inspection, and length inspection without drifting into jq-style language semantics; batch reads should stay cheap; repeated `--path` must have one explicit result shape; non-object/non-countable queries must stay explicit errors instead of coercions | `pkg/builtin/op_json.go`, `pkg/builtin/json_path.go`, `pkg/builtin/commands/json/manual.md` |
| Terminal-Bench prototype scope + comparison artifact | SHOULD | Need one narrow external-comparison layer that stays downstream from native scenario ids, checked-in baseline reports, and Terminal-Bench mapping truth without growing into a second benchmark suite | `benchmarks/external_mapping/*.go`, `benchmarks/terminal_bench_compare/prototype_scope.json`, `benchmarks/terminal_bench_compare/reports/*` |
| High-performance mount semantic axes + CLI pressure matrix | SHOULD | Need one explicit mount-system contract that separates truth model, materialization, write semantics, consistency, and latency so tool authors and adapter authors can reason about fanout-heavy CLI workloads without hiding behind ad hoc fallback loops | `docs/architecture-high-performance-mount-system.md` |
| RC parser subset (`export` + `alias`) | SHOULD | Need deterministic startup customization from read-only mounted rc files while rejecting ambiguous shell constructs | `pkg/engine/orchestrator.go` |
| VirtualMount driver + mount-router composition | SHOULD | Need to project business context trees (memory/resources/skills) without coupling core to backend, while keeping capability dispatch and high-fanout mount pressure aligned with the governing mount architecture doc and with builtin fast paths that consume `ListEntries`, `SearchContent`, and `ApplyMutations` | `pkg/contract/mount_contract.go`, `pkg/engine/virtualfs_bridge.go`, `pkg/builtin/op_listing.go`, `pkg/builtin/op_pattern_scan.go`, `pkg/builtin/op_rg.go`, `pkg/builtin/op_move.go`, `docs/architecture-high-performance-mount-system.md` |
| PreparedOps snapshot (`PrepareOps` + `ExecutePrepared`) | SHOULD | Need high-frequency command execution with stable callbacks; avoid repeating per-exec normalize/wrap/bootstrap work | `pkg/engine/orchestrator.go`, `pkg/engine/runtime/runtime_stack.go` |
| Adapter seam conformance harness | SHOULD | Need reusable lifecycle/projection/managed-memory seam assertions across multiple adapter shapes without copying benchmark smoke logic into each adapter test; prefer an `error`-returning core plus thin `testing.T` wrappers so helper failure semantics stay directly testable | `pkg/adapter/internal/contracttest/lifecycle.go`, `pkg/adapter/internal/contracttest/lifecycle_test.go`, `pkg/adapter/reference/adapter_conformance_test.go`, `pkg/adapter/resourceset/adapter_test.go` |
| Adapter mount conformance harness | SHOULD | Need reusable `VirtualMount` list/search/describe/read-only metadata assertions across multiple adapter shapes without turning benchmark or adapter-local tests into duplicated mount smoke checks; keep it mount-only and directly self-tested | `pkg/adapter/internal/contracttest/mount.go`, `pkg/adapter/internal/contracttest/mount_test.go`, `pkg/adapter/reference/adapter_conformance_test.go`, `pkg/adapter/resourceset/adapter_test.go` |
| Adapter composition/evolution stress proof | SHOULD | Need one harder multi-step workload that proves existing truth surfaces stay aligned together under mutation, invalidation, resume, audit, metrics, and denials without inventing new product nouns | `pkg/adapter/reference/adapter_test.go`, `benchmarks/simsh_native_reference/suite.go` |
| Native benchmark scenario catalog + external mapping guardrail | SHOULD | Need stable native benchmark scenario ids/categories plus a downstream machine-readable mapping layer for external benchmark families; keep evaluation-feasibility artifacts aligned without mutating the native suite | `benchmarks/internal/scenarios/catalog.go`, `benchmarks/external_mapping/scenario_inventory.json`, `benchmarks/external_mapping/*_mapping.json`, `benchmarks/external_mapping/mapping_guard_test.go` |

## Deprecations
- `pkg/builtin/manuals/*.md` -> `pkg/builtin/commands/*/manual.md` (remove duplicated manual sources and drift risk)
