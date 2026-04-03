# External Benchmark Mapping

This directory captures the checked-in evaluation-feasibility layer for `simsh`.

It answers a narrow question:

- how current `simsh` native benchmark scenarios relate to external benchmark families.

It does **not** mean `simsh` adopts those external benchmarks wholesale.

## Artifacts

- `scenario_inventory.json`
  - curated evaluation inventory of native benchmark scenarios, keyed by stable native scenario ids
- `terminal_bench_mapping.json`
  - mapping from native scenarios to `Terminal-Bench`
- `swe_bench_live_mapping.json`
  - mapping from native scenarios to `SWE-bench-Live`

The same evaluation layer may also include a lightweight comparison prototype:
- it consumes the checked-in native benchmark report plus the checked-in inventory/mapping layer
- it emits one compact comparison artifact for a narrowly chosen external family slice
- it must not create a second scenario catalog or rename native scenarios
- it should treat live benchmark re-execution as optional regeneration work, not as a hidden prerequisite for reading or validating the comparison artifact

## Status Values

- `as_is`
  - native scenario already matches the external family closely enough to compare without reshaping the task
- `translated`
  - the scenario is relevant, but only after a narrow translation layer
- `excluded`
  - the scenario should remain outside that external family because forcing a fit would distort `simsh` scope

## Design Rule

The native benchmark remains the primary SSOT.

This folder is a downstream evaluation layer:
- stable native scenario ids and categories stay canonical; task-shape summaries and truth-surface lists are curated evaluation metadata layered on top
- it must not rename or mutate native scenarios to look more benchmark-compatible
- it must not introduce environment synthesis or new runtime nouns
- a comparison prototype should prefer one direct-fit slice plus at most one translated proof slice instead of broad external-family coverage
