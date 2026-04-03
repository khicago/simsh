# Terminal-Bench Comparison Prototype

This directory is the lightweight K-028 comparison layer for `simsh`.

It exists to answer one narrow question:

- what does the current native benchmark suite look like when projected through the smallest useful Terminal-Bench comparison slice?

It is downstream from two existing truth sources:

1. the native benchmark report emitted by `benchmarks/simsh_native_reference`
2. the checked-in evaluation-feasibility layer under `benchmarks/external_mapping/`

It does **not** adopt Terminal-Bench wholesale.

## Design Contract

- Native scenario ids and categories remain canonical.
- Mapping statuses (`as_is`, `translated`, `excluded`) remain interpretation metadata from `benchmarks/external_mapping/terminal_bench_mapping.json`.
- The comparison prototype consumes those inputs and emits compact artifacts; it does not create a second benchmark suite.
- The prototype should center the strongest direct fit (`inspect_edit_write_loop`) and may include at most one translated proof slice.
- The comparison layer must stay report-driven and read-only with respect to native benchmark semantics.

## Non-Goals

- no full Terminal-Bench harness
- no benchmark-only scenario ids
- no runtime or adapter expansion to improve external coverage
- no pressure to rename native scenarios to look more benchmark-native

## Expected Outputs

The implementation in this directory should emit:

- one machine-readable comparison artifact
- one adjacent human-readable report

Those outputs should make provenance explicit:

- which native report was consumed
- which native scenarios were selected
- whether each selected slice is `as_is` or `translated`
- what translation notes or exclusions still bound the result
