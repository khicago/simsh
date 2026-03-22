---
name: tree
synopsis: "tree [-a] [-L N] [--fmt outline|ascii|json] [PATH...]"
category: navigation
---

# tree -- display directory structure

## SYNOPSIS

    tree [-a] [-L N] [--fmt outline|ascii|json] [PATH...]

## DESCRIPTION

Render directory contents in a dual-readable outline by default. Paths may be
absolute or relative to the current virtual working directory. If no path is
provided, it starts from the current virtual working directory.

`tree` supports three renderers:
- `outline` -- the default high-signal, low-noise text format
- `ascii` -- classic branch-art compatibility mode
- `json` -- machine-readable flat entries with `path`, `depth`, and `kind`

## FLAGS

- `-a` -- Include hidden entries (names starting with `.`).
- `-L N` -- Limit recursion depth to a non-negative integer.
- `--fmt outline|ascii|json` -- Select output format. Default is `outline`.

## EXAMPLES

Show root tree:

    tree /

Use classic ASCII compatibility mode:

    tree --fmt ascii /

Get machine-readable entries:

    tree --fmt json /task_outputs

Limit depth to 2:

    tree -L 2 /task_outputs

Include hidden files:

    tree -a /knowledge_base

## NOTES

- Paths may be absolute or relative to the current virtual working directory.
- File targets are printed as a single line path.
- `-L 0` prints only the root line for directory targets.
- `outline` is the default because `tree` is mainly an inspection view, not a strong pipeline primitive.
- `json` uses flat entries instead of nested trees so downstream filtering and diffing stay simple.

## SEE ALSO

ls, find
