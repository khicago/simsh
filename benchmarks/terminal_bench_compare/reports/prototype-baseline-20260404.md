# Terminal-Bench Comparison Prototype

## Scope

- External family: `Terminal-Bench`
- Source benchmark: `simsh_native_reference`
- Source report: `benchmarks/simsh_native_reference/reports/baseline-20260404.json`

## Summary

- Compared scenarios: 2
- Direct-fit slices: 1
- Translated proof slices: 1
- All native scenarios successful: `true`

## Compared Scenarios

| scenario | role | external task | status | success | patch workflow | trace completeness |
| --- | --- | --- | --- | --- | --- | --- |
| `inspect_edit_write_loop` | `direct_fit` | `terminal_file_inspect_edit_loop` | `as_is` | `true` | `true` | `1.00` |

Why `inspect_edit_write_loop` is here: Closest current native fit to terminal-oriented read-edit-write patch workflows.
Comparable dimensions for `inspect_edit_write_loop`: `read_edit_write_patch_loop`, `trace_visible_task_result`, `reviewable_patch_workflow`
Excluded dimensions for `inspect_edit_write_loop`: `full_terminal_bench_task_suite`
| `relative_navigation_session` | `translated_proof` | `terminal_relative_navigation_subflow` | `translated` | `true` | `false` | `1.00` |

Why `relative_navigation_session` is here: Smallest translated slice that exercises terminal-task navigation pressure without dragging in broader runtime or adapter semantics.
Comparable dimensions for `relative_navigation_session`: `relative_navigation`, `session_cwd_continuity`, `path_resolution_feedback`
Excluded dimensions for `relative_navigation_session`: `full_repo_task_context`, `broader_terminal_bench_environment_shape`
Translation notes for `relative_navigation_session`: Use as a sub-behavior inside larger terminal tasks rather than as a direct external scenario analogue.

## Notes

- This artifact stays downstream from the native benchmark SSOT and the checked-in Terminal-Bench mapping layer.
- It is a comparison prototype, not a benchmark adoption layer and not a second scenario catalog.
