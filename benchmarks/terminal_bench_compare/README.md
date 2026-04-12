# Terminal-Bench Comparison Prototype

This directory is the K-028 lightweight comparison layer for `simsh`.

It answers a narrower question than the mapping layer:

- given the current native benchmark suite and Terminal-Bench mapping, what is the smallest checked-in comparison artifact worth maintaining right now?

This layer stays strictly downstream from:

- `benchmarks/internal/scenarios/`
- `benchmarks/simsh_native_reference/`
- `benchmarks/external_mapping/`

## Scope Contract

`prototype_scope.json` is the checked-in scope SSOT for the prototype.

Current design rules:
- exactly one direct-fit native scenario
- exactly one translated proof slice
- all scenario ids must come from the native benchmark catalog
- all expected statuses must match `benchmarks/external_mapping/terminal_bench_mapping.json`

## Non-Goals

- full Terminal-Bench harness adoption
- benchmark-only scenario ids
- changes to native benchmark semantics or runtime behavior
- broad translated coverage beyond one proof slice

## Expected Outputs

The completed comparison layer emits:
- one compact machine-readable comparison artifact
- one adjacent human-readable summary/report

Both outputs should be derived from the native benchmark report plus the K-027 mapping artifacts rather than hand-maintained in parallel.

Current checked-in outputs:
- `reports/prototype-baseline-20260404.json`
- `reports/prototype-baseline-20260404.md`

Repo-level evidence breadcrumb:
- `../evidence_manifest.json`
  - the checked-in proof-surface SSOT that indexes this prototype together with
    the native baseline and paired uplift artifacts

Regenerate with:

```bash
make benchmark-refresh
```

The refresh target is orchestration-only:
- it regenerates the checked-in native baseline
- it regenerates the checked-in comparison JSON/MD pair
- it does not change prototype scope or mapping files

The native baseline is a freshness snapshot, so fields like `generated_at` and duration-based latency numbers are expected to move when refreshed.
The comparison JSON/MD pair remains the byte-guarded downstream artifact pair.
