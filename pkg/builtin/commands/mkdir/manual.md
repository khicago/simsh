---
name: mkdir
synopsis: "mkdir [-p] PATH..."
category: file-management
---

# mkdir -- create directories

## SYNOPSIS

    mkdir [-p] PATH...

## DESCRIPTION

Create one or more directories. By default, parent directories must already
exist. Use `-p` to create intermediate directories as needed.

## FLAGS

- `-p` -- Create parent directories as needed. No error if the directory already exists.

## EXAMPLES

Create a single directory:

    mkdir /task_outputs/reports

Create nested directories:

    mkdir -p /task_outputs/2026/01/data

Create multiple directories:

    mkdir /task_outputs/logs /task_outputs/cache

## NOTES

- Paths may be absolute or relative to the current virtual working directory.
- Write operations are subject to zone policy checks.
- Without `-p`, fails if the parent directory does not exist.

## SEE ALSO

ls, cp, rm
