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
| RC parser subset (`export` + `alias`) | SHOULD | Need deterministic startup customization from read-only mounted rc files while rejecting ambiguous shell constructs | `pkg/engine/orchestrator.go` |
| VirtualMount driver + mount-router composition | SHOULD | Need to project business context trees (memory/resources/skills) without coupling core to backend | `pkg/contract/mount_contract.go`, `pkg/engine/virtualfs_bridge.go` |

## Deprecations
- `pkg/builtin/manuals/*.md` -> `pkg/builtin/commands/*/manual.md` (remove duplicated manual sources and drift risk)
