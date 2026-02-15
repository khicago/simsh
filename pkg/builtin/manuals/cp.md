---
name: cp
synopsis: "cp [-r] ABS_SRC ABS_DST"
category: file-management
---

# cp -- copy files

## SYNOPSIS

    cp [-r] ABS_SRC ABS_DST

## DESCRIPTION

Copy a file or directory from source to destination. Both paths must be
absolute. Use `-r` for recursive directory copy.

## FLAGS

- `-r` -- Recursive copy. Required when copying directories.

## EXAMPLES

Copy a file:

    cp /knowledge_base/template.md /task_outputs/report.md

Recursive directory copy:

    cp -r /knowledge_base/templates /task_outputs/templates

## NOTES

- All paths must be absolute.
- Write operations are subject to zone policy checks.
- Without `-r`, copying a directory will fail.

## SEE ALSO

mv, rm, mkdir
