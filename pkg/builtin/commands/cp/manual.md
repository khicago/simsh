---
name: cp
synopsis: "cp [--confirm] [--json] SRC_PATH DST_PATH"
category: file-management
---

# cp -- copy files

## SYNOPSIS

    cp [--confirm] [--json] SRC_PATH DST_PATH

## DESCRIPTION

Copy a file from source to destination. Paths may be absolute or relative to
the current virtual working directory.

Default success output stays silent. Use `--confirm` for a low-noise text
acknowledgement or `--json` for a machine-readable copy summary.

## FLAGS

- `--confirm` -- Emit one confirmation line when the copy succeeds.
- `--json` -- Emit a machine-readable copy summary object.

## EXAMPLES

Copy a file:

    cp /knowledge_base/template.md /task_outputs/report.md

Emit confirmation:

    cp --confirm /knowledge_base/template.md /task_outputs/report.md

Emit JSON summary:

    cp --json /knowledge_base/template.md /task_outputs/report.md

## NOTES

- Paths may be absolute or relative to the current virtual working directory.
- Write operations are subject to zone policy checks.
- Directory copy is not supported.
- `--json` returns `src`, `dest`, and copied `bytes`.

## SEE ALSO

mv, rm, mkdir
