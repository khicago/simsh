---
name: sed
synopsis: "sed -i 's/old/new/[g]' ABS_FILE | sed -n 'Np'|'M,Np' [ABS_FILE]"
category: text-processing
---

# sed -- stream editor

## SYNOPSIS

    sed -i 's/old/new/[g]' ABS_FILE
    sed -n 'Np' [ABS_FILE]
    sed -n 'M,Np' [ABS_FILE]
    COMMAND | sed -n 'Np'
    COMMAND | sed -n 'M,Np'

## DESCRIPTION

A focused subset of sed for deterministic text operations. Supports two modes:

1. In-place substitution (`-i`): Replace occurrences of a string in a file.
2. Line printing (`-n`): Extract specific lines or line ranges.

## FLAGS

- `-i` -- In-place edit mode. Requires a substitution expression and a file path.
- `-n` -- Suppress default output. Used with print expressions (`Np` or `M,Np`).

## EXPRESSIONS

- `'s/old/new/'` -- Replace first occurrence of `old` with `new`.
- `'s/old/new/g'` -- Replace all occurrences of `old` with `new`.
- `'Np'` -- Print line N.
- `'M,Np'` -- Print lines M through N (inclusive).

The delimiter in substitution expressions can be any character (e.g. `s|old|new|`).
Use backslash to escape the delimiter within the pattern.

## EXAMPLES

Replace first occurrence in a file:

    sed -i 's/draft/final/' /task_outputs/report.md

Replace all occurrences:

    sed -i 's/TODO/DONE/g' /task_outputs/notes.md

Print line 5:

    sed -n '5p' /knowledge_base/data.txt

Print lines 10 through 20:

    sed -n '10,20p' /knowledge_base/data.txt

Print lines from stdin:

    cat /task_outputs/log.txt | sed -n '1,5p'

## NOTES

- Only `-i` and `-n` modes are supported.
- In-place edit requires exactly one file path.
- Line numbers are 1-based and must be positive integers.
- All paths must be absolute.

## SEE ALSO

grep, cat, tee
