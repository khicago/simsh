---
name: view
synopsis: "view [--start N] [--lines N] [--fmt jsonl] [--] PATH"
category: navigation
---

# view -- numbered file window

## SYNOPSIS

    view PATH
    view --start N --lines N PATH
    view --fmt jsonl --start N --lines N PATH

## DESCRIPTION

Print a numbered slice of a file. Line numbers are 1-based. The default window
is 80 lines from the start. A `shown N/TOTAL from START` footer says whether
the file continues. Asking for a start past EOF reports `file has N lines`
instead of an empty window. Use `--fmt jsonl`, not `--json`.

Use `--` before a relative path that begins with `-`.

## FLAGS

- `--start N`, `-s N` -- First line to print. Default `1`.
- `--lines N`, `-n N` -- Window size. Default `80`.
- `--fmt jsonl` -- Emit `{line,text}` records.

## EXAMPLES

Read the top of a file:

    view /knowledge_base/readme.md

Read a later window:

    view --start 20 --lines 40 /task_outputs/report.md
