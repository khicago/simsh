---
name: touch
synopsis: "touch ABS_FILE..."
category: file-management
---

# touch -- create empty files

## SYNOPSIS

    touch ABS_FILE...

## DESCRIPTION

Create one or more empty files. If the file already exists, it is not
modified.

## EXAMPLES

Create a single file:

    touch /task_outputs/notes.md

Create multiple files:

    touch /task_outputs/a.txt /task_outputs/b.txt

## NOTES

- All paths must be absolute.
- Write operations are subject to zone policy checks.
- Does not modify existing files.

## SEE ALSO

mkdir, tee, rm
