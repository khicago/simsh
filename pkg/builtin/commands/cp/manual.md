---
name: cp
synopsis: "cp ABS_SRC ABS_DST"
category: file-management
---

# cp -- copy files

## SYNOPSIS

    cp ABS_SRC ABS_DST

## DESCRIPTION

Copy a file from source to destination. Both paths must be absolute.

## EXAMPLES

Copy a file:

    cp /knowledge_base/template.md /task_outputs/report.md

## NOTES

- All paths must be absolute.
- Write operations are subject to zone policy checks.
- Directory copy is not supported.

## SEE ALSO

mv, rm, mkdir
