---
name: cp
synopsis: "cp SRC_PATH DST_PATH"
category: file-management
---

# cp -- copy files

## SYNOPSIS

    cp SRC_PATH DST_PATH

## DESCRIPTION

Copy a file from source to destination. Paths may be absolute or relative to
the current virtual working directory.

## EXAMPLES

Copy a file:

    cp /knowledge_base/template.md /task_outputs/report.md

## NOTES

- Paths may be absolute or relative to the current virtual working directory.
- Write operations are subject to zone policy checks.
- Directory copy is not supported.

## SEE ALSO

mv, rm, mkdir
