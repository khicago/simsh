---
name: sort
synopsis: "sort [-r] [-n] [-u] [ABS_FILE]"
category: text-processing
---

# sort -- sort lines

## SYNOPSIS

    sort [-r] [-n] [-u] [ABS_FILE]
    COMMAND | sort [-r] [-n] [-u]

## DESCRIPTION

Sort lines of text from a file or stdin. By default, sorts lines
lexicographically in ascending order.

## FLAGS

- `-r` -- Reverse the sort order (descending).
- `-n` -- Numeric sort. Compare lines by their leading numeric value.
- `-u` -- Unique. Output only the first of equal lines.

## EXAMPLES

Sort a file alphabetically:

    sort /task_outputs/names.txt

Reverse sort:

    sort -r /task_outputs/names.txt

Numeric sort:

    sort -n /task_outputs/scores.txt

Unique sorted lines from stdin:

    cat /task_outputs/log.txt | sort -u

## NOTES

- All paths must be absolute.
- Reads from stdin when no file path is provided.
- Numeric sort treats non-numeric lines as having value 0.

## SEE ALSO

uniq, wc, grep
