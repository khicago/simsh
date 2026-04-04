# Terminal-Bench Comparison Freshness Automation

## Goal

`K-029` makes one thing explicit:

- how to deterministically refresh the checked-in native benchmark baseline and the checked-in Terminal-Bench comparison artifact/report pair.

## Refresh Contract

The canonical refresh path is:

```bash
make benchmark-refresh
```

That runs two steps, in order:

1. regenerate the checked-in native baseline
2. regenerate the checked-in Terminal-Bench comparison JSON/Markdown pair

## Important Boundary

The native baseline is a **freshness snapshot**, not a byte-stable golden file.

Expected changes after refresh include:
- `generated_at`
- duration-derived latency fields
- scenario `duration_ms`

The comparison JSON/Markdown pair is still expected to stay byte-aligned with the current generator and current checked-in inputs, and that pair is what the narrow drift guard protects.

## Why This Shape

- It keeps refresh automation orchestration-only.
- It avoids turning the comparison layer into a second benchmark harness.
- It preserves the native benchmark and prototype scope as the only inputs that define what is being compared.
