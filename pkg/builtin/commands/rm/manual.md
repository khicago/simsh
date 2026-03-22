---
name: rm
synopsis: "rm PATH..."
category: file-management
---

# rm -- remove files

## SYNOPSIS

    rm PATH...

## DESCRIPTION

Remove one or more files.

## EXAMPLES

Remove a file:

    rm /task_outputs/old_report.md

Remove multiple files:

    rm /task_outputs/temp1.txt /task_outputs/temp2.txt

## NOTES

- Paths may be absolute or relative to the current virtual working directory.
- Write operations are subject to zone policy checks.
- Directory removal is not supported.
- This operation is not reversible.

## SEE ALSO

mkdir, cp, mv
