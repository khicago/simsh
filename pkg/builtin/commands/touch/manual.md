---
name: touch
synopsis: "touch PATH..."
category: file-management
---

# touch -- create empty files

## SYNOPSIS

    touch PATH...

## DESCRIPTION

Create one or more empty files. If the file already exists, it is not
modified.

## EXAMPLES

Create a single file:

    touch /task_outputs/notes.md

Create multiple files:

    touch /task_outputs/a.txt /task_outputs/b.txt

## NOTES

- Paths may be absolute or relative to the current virtual working directory.
- Write operations are subject to zone policy checks.
- Does not modify existing files.

## SEE ALSO

mkdir, tee, rm
