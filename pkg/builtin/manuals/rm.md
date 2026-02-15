---
name: rm
synopsis: "rm [-r] ABS_PATH..."
category: file-management
---

# rm -- remove files

## SYNOPSIS

    rm [-r] ABS_PATH...

## DESCRIPTION

Remove one or more files or directories. Use `-r` for recursive removal
of directories and their contents.

## FLAGS

- `-r` -- Recursive removal. Required when removing directories.

## EXAMPLES

Remove a file:

    rm /task_outputs/old_report.md

Remove multiple files:

    rm /task_outputs/temp1.txt /task_outputs/temp2.txt

Remove a directory recursively:

    rm -r /task_outputs/old_reports

## NOTES

- All paths must be absolute.
- Write operations are subject to zone policy checks.
- Without `-r`, removing a directory will fail.
- This operation is not reversible.

## SEE ALSO

mkdir, cp, mv
