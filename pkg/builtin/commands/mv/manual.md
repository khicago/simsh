---
name: mv
synopsis: "mv [--confirm] [--json] SRC_PATH DST_PATH"
category: file-management
---

# mv -- move or rename files

## SYNOPSIS

    mv [--confirm] [--json] SRC_PATH DEST_PATH

## DESCRIPTION

Move or rename a file or directory. Paths may be absolute or relative to the
current virtual working directory.

Default success output stays silent. Use `--confirm` for a low-noise text
acknowledgement or `--json` for a machine-readable move summary.

## FLAGS

- `--confirm` -- Emit one confirmation line when the move succeeds.
- `--json` -- Emit a machine-readable move summary object.

## EXAMPLES

Rename a file:

    mv /task_outputs/draft.md /task_outputs/final.md

Move a file to another directory:

    mv /task_outputs/report.md /task_outputs/archive/report.md

Move a directory:

    mv /task_outputs/old_reports /task_outputs/archive

Emit confirmation:

    mv --confirm /task_outputs/draft.md /task_outputs/final.md

Emit JSON summary:

    mv --json /task_outputs/draft.md /task_outputs/final.md

## NOTES

- Paths may be absolute or relative to the current virtual working directory.
- Write operations are subject to zone policy checks.
- Destination parent directory must exist.
- `--json` returns `src`, `dest`, and moved `bytes`.

## SEE ALSO

cp, rm, mkdir
