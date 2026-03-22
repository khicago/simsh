---
name: rm
synopsis: "rm [--confirm] [--json] PATH..."
category: file-management
---

# rm -- remove files

## SYNOPSIS

    rm [--confirm] [--json] PATH...

## DESCRIPTION

Remove one or more files.

Default success output stays silent. Use `--confirm` for a low-noise text
acknowledgement or `--json` for a machine-readable removal summary.

## FLAGS

- `--confirm` -- Emit one confirmation line per removed path.
- `--json` -- Emit machine-readable removal status entries.

## EXAMPLES

Remove a file:

    rm /task_outputs/old_report.md

Remove multiple files:

    rm /task_outputs/temp1.txt /task_outputs/temp2.txt

Emit confirmation:

    rm --confirm /task_outputs/temp1.txt

Emit JSON status:

    rm --json /task_outputs/temp1.txt /task_outputs/temp2.txt

## NOTES

- Paths may be absolute or relative to the current virtual working directory.
- Write operations are subject to zone policy checks.
- Directory removal is not supported.
- This operation is not reversible.
- `--confirm` and `--json` only report successful removals; failures still follow the normal non-zero exit code path.

## SEE ALSO

mkdir, cp, mv
