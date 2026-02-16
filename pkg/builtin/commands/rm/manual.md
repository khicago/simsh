---
name: rm
synopsis: "rm ABS_PATH..."
category: file-management
---

# rm -- remove files

## SYNOPSIS

    rm ABS_PATH...

## DESCRIPTION

Remove one or more files.

## EXAMPLES

Remove a file:

    rm /task_outputs/old_report.md

Remove multiple files:

    rm /task_outputs/temp1.txt /task_outputs/temp2.txt

## NOTES

- All paths must be absolute.
- Write operations are subject to zone policy checks.
- Directory removal is not supported.
- This operation is not reversible.

## SEE ALSO

mkdir, cp, mv
