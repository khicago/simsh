---
name: mv
synopsis: "mv SRC_PATH DST_PATH"
category: file-management
---

# mv -- move or rename files

## SYNOPSIS

    mv SRC_PATH DST_PATH

## DESCRIPTION

Move or rename a file or directory. Paths may be absolute or relative to the
current virtual working directory.

## EXAMPLES

Rename a file:

    mv /task_outputs/draft.md /task_outputs/final.md

Move a file to another directory:

    mv /task_outputs/report.md /task_outputs/archive/report.md

Move a directory:

    mv /task_outputs/old_reports /task_outputs/archive

## NOTES

- Paths may be absolute or relative to the current virtual working directory.
- Write operations are subject to zone policy checks.
- Destination parent directory must exist.

## SEE ALSO

cp, rm, mkdir
