# Paired A/B Uplift Proof

## Scope

- Comparison rule: `same_agent_same_budget_only_substrate_changes`
- Agent: `paired_probe_agent_v1`
- simsh substrate: `simsh_full_sessioned`
- baseline substrate: `thin_core_stateless`
- Task manifest: `benchmarks/paired_uplift/task_set.json`

## Summary

- Total tasks: 3
- simsh success count: 3
- baseline success count: 2
- simsh retries: 0
- baseline retries: 3
- simsh wasted observation tokens: 0
- baseline wasted observation tokens: 11

## Task Results

| scenario | winner | simsh success | baseline success | retry delta | wasted-token delta | misunderstanding delta |
| --- | --- | --- | --- | --- | --- | --- |
| `relative_navigation_session` | `simsh` | `true` | `true` | `1` | `1` | `1` |

Why `relative_navigation_session` is included: Smallest paired task that directly pressures session-local cwd continuity and relative-path feedback.
Truth surfaces for `relative_navigation_session`: `virtual_cwd`, `relative_path_resolution`, `execution_trace`
| `inspect_edit_write_loop` | `simsh` | `true` | `true` | `1` | `5` | `1` |

Why `inspect_edit_write_loop` is included: Strongest current inspect/edit/write slice for measuring whether the rg-style search front door removes fallback retries.
Truth surfaces for `inspect_edit_write_loop`: `file_mutation`, `execution_trace`, `session_continuity`
| `trace_consumable_planning` | `simsh` | `true` | `false` | `1` | `5` | `1` |

Why `trace_consumable_planning` is included: Pressures narrow JSON query tooling under a fixed observation budget so the harness can measure fallback cost rather than only final correctness.
Truth surfaces for `trace_consumable_planning`: `execution_trace`, `structured_result_feedback`

## Failure Taxonomy

| bucket | runtime | kind | count | scenarios |
| --- | --- | --- | --- | --- |
| `environment_misunderstanding` | `thin_core_stateless` | `missing_json_query_surface` | 1 | `trace_consumable_planning` |
| `environment_misunderstanding` | `thin_core_stateless` | `missing_rg_front_door` | 1 | `inspect_edit_write_loop` |
| `environment_misunderstanding` | `thin_core_stateless` | `session_cwd_not_persistent` | 1 | `relative_navigation_session` |
| `failure` | `thin_core_stateless` | `budget_exhausted_after_fallback` | 1 | `trace_consumable_planning` |

## Notes

- This harness stays downstream from the native benchmark identity contract.
- The baseline is repo-controlled and intentionally thinner; it is not an ambient host-shell bakeoff.
- Raw paired runs are freshness snapshots; this markdown is a deterministic downstream rendering of the checked-in snapshot.
