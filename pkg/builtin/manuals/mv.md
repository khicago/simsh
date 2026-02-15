---
name: mv
synopsis: "mv ABS_SRC ABS_DST"
category: file-management
---

# mv -- move or rename files

## SYNOPSIS

    mv ABS_SRC ABS_DST

## DESCRIPTION

Move or rename a file or directory. Both source and destination must be
absolute paths.

## EXAMPLES

Rename a file:

    mv /task_outputs/draft.md /task_outputs/final.md

Move a file to another directory:

    mv /task_outputs/report.md /task_outputs/archive/report.md

Move a directory:

    mv /task_outputs/old_reports /task_outputs/archive

## NOTES

- All paths must be absolute.
- Write operations are subject to zone policy checks.
- Destination parent directory must exist.

## SEE ALSO

cp, rm, mkdir
