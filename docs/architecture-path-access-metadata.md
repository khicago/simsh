---
title: Path Access Metadata and Listing/API Formats
required: false
sop:
  - Read this doc before changing PathMeta access/capabilities, ls -l output formats, or /v1/execute metadata.
  - Update this doc when schema/flags/output formats change.
  - Regenerate `docs/must-sop.md` after SOP/frontmatter changes.
---

# Path Access Metadata and Listing/API Formats

## Context
simsh is designed for agentic execution. Agents need to know which paths are writable vs immutable *without* trial-and-error.

We already enforce immutability for mount-backed paths and synthetic mount-parent directories. This doc defines the follow-up design that makes those rules explicit via **SSOT path access metadata** and consistent consumption in:
- `ls -l` (human + agent friendly, token-efficient)
- `man ls` (docs)
- `/v1/execute` (API response metadata, opt-in)

## Goals
- Provide a **single source of truth** (SSOT) for per-path access semantics.
- Keep default `ls -l` output **short** (works for directories with hundreds of entries).
- Provide **high-signal alternate formats**: Markdown table and JSON.
- Make CLI/man/API adapt to the SSOT schema, not re-encode rules.

## Non-Goals
- Full ACL system (users/groups/masks).
- Making mounts writable.
- Guaranteeing backwards compatibility for long listing formatting.

## SSOT Schema: `access` + `capabilities`

### Fields
Extend `contract.PathMeta` and `contract.LSLongRow` with:

- `access`: `"ro" | "rw"`
  - `"ro"` means the path is treated as immutable by the virtual runtime.
  - `"rw"` means write operations *may* be allowed (still bounded by policy and adapter rules).
- `capabilities`: `[]string`
  - A compact, explicit set of allowed operations.
  - Used by agents (planning) and by formatters (ls/api).

### Canonical capability set (initial)
Capabilities are strings from this set:

- `read`: can read file content (`ReadRawContent`)
- `list`: can list directory children (`ListChildren`)
- `describe`: can return metadata (`DescribePath`)
- `search`: can enumerate files / resolve search (`CollectFilesUnder`, `ResolveSearchPaths`)
- `write`: can create/overwrite file (`WriteFile`, `>` redirection)
- `append`: can append (`AppendFile`, `>>` redirection)
- `edit`: can in-place edit (`EditFile`, `sed -i`)
- `mkdir`: can create directory (`MakeDir`)
- `remove`: can delete file (`RemoveFile`)

Notes:
- `cp`/`mv` are composite commands; they should derive behavior from the above capabilities via preflight checks.
- `access` is redundant with capabilities for `write/mkdir/remove`, but kept for human readability and quick branching.

### Derivation rules (initial)
These rules define what `DescribePath` SHOULD emit:

- Mount-backed paths (under an active `VirtualMount.MountPoint()`):
  - `access=ro`
  - `capabilities` subset: `read`, `list`, `describe`, `search` (depending on mount support)
- Synthetic mount-parent directories (e.g. `/sys` that exists only to parent `/sys/bin`):
  - `access=ro`
  - `capabilities` subset: `list`, `describe`, `search`
- Zone-backed virtual filesystem paths (e.g. `/task_outputs`, `/temp_work`, `/knowledge_base`):
  - `access` and `capabilities` derived from:
    - execution policy (`read-only` vs `full|write-limited`)
    - zone writability (agentfs zones)
    - adapter restrictions (localfs root constraints, symlink rules, etc.)

## `ls -l` Output Formats

### Default long format: short columns + legend
Default `ls -l` should output one entry per line with **fixed columns**:

`MODE ACCESS KIND LINES PATH`

- `MODE`: `d` for directory, `-` for file
- `ACCESS`: `ro|rw`
- `KIND`: existing semantic kind (e.g. `sys_bin_dir`, `sys_binary`, `virtual_dir`, `file`, ...)
- `LINES`: `-` when unavailable
- `PATH`: display path

At the end of the output, print a single legend line (to preserve human readability):

`# columns: mode access kind lines path`

### `--fmt md`
`ls -l --fmt md` outputs a Markdown table:

| mode | access | kind | lines | path |
|---|---|---|---:|---|
| d | ro | virtual_dir | - | /sys |
| d | ro | sys_bin_dir | - | /sys/bin |
| - | ro | sys_binary | - | /sys/bin/ls |

### `--fmt json`
`ls -l --fmt json` outputs JSON for machine consumption.

Recommended shape:
```json
{
  "columns": ["mode", "access", "kind", "lines", "path"],
  "entries": [
    {"mode":"d","access":"ro","kind":"virtual_dir","lines":-1,"path":"/sys","capabilities":["list","describe","search"]},
    {"mode":"d","access":"ro","kind":"sys_bin_dir","lines":-1,"path":"/sys/bin","capabilities":["list","describe","search"]},
    {"mode":"-","access":"ro","kind":"sys_binary","lines":-1,"path":"/sys/bin/ls","capabilities":["read","describe"]}
  ]
}
```

Notes:
- JSON SHOULD include `capabilities` even if text/md formats hide it by default.
- `lines=-1` is allowed to mean "unknown/unavailable".

## API: `/v1/execute` Metadata (opt-in)

### Request
Add:
- `include_meta: boolean` (default: false)

### Response
Extend response with:
- `meta` (only when `include_meta=true`)

`meta` SHOULD include:
- `paths`: list of paths involved in execution with `access` + `capabilities` + `kind`

The API meta is derived from the same SSOT schema as `ls -l --fmt json`.

Current implementation note:
- `meta.paths` is implemented.
- `meta.denials` is deferred and can be added in a follow-up without changing the SSOT schema.

## Test Matrix (when implementing)
- `ls -l` output (default) uses short columns and includes the legend line once.
- `ls -l --fmt md` renders a valid markdown table.
- `ls -l --fmt json` is valid JSON and includes capabilities.
- Synthetic mount parents (`/sys`) show `access=ro`.
- Mount-backed entries show `access=ro`.
- Zone-backed writable paths show `access=rw` only when policy + adapter allow it.
- `/v1/execute include_meta=true` returns `meta.paths` with SSOT access/capabilities.
