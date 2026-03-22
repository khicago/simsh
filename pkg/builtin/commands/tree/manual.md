---
name: tree
synopsis: "tree [-a] [-L N] [PATH...]"
category: navigation
---

# tree -- display directory structure

## SYNOPSIS

    tree [-a] [-L N] [PATH...]

## DESCRIPTION

Render directory contents as an ASCII tree. Paths may be absolute or relative
to the current virtual working directory. If no path is provided, it starts
from the current virtual working directory.

## FLAGS

- `-a` -- Include hidden entries (names starting with `.`).
- `-L N` -- Limit recursion depth to a non-negative integer.

## EXAMPLES

Show root tree:

    tree /

Limit depth to 2:

    tree -L 2 /task_outputs

Include hidden files:

    tree -a /knowledge_base

## NOTES

- Paths may be absolute or relative to the current virtual working directory.
- File targets are printed as a single line path.
- `-L 0` prints only the root line for directory targets.

## SEE ALSO

ls, find
