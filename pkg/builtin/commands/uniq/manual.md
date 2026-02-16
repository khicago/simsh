---
name: uniq
synopsis: "uniq [-c] [-d] [ABS_FILE]"
category: text-processing
---

# uniq -- filter adjacent duplicate lines

## SYNOPSIS

    uniq [-c] [-d] [ABS_FILE]
    COMMAND | uniq [-c] [-d]

## DESCRIPTION

Filter out adjacent duplicate lines from a file or stdin. Only consecutive
identical lines are considered duplicates. Typically used after `sort`.

## FLAGS

- `-c` -- Prefix each line with the count of consecutive occurrences.
- `-d` -- Only output lines that are repeated (duplicates only).

## EXAMPLES

Remove adjacent duplicates:

    uniq /task_outputs/sorted.txt

Count occurrences:

    sort /task_outputs/log.txt | uniq -c

Show only duplicated lines:

    sort /task_outputs/data.txt | uniq -d

Pipeline usage:

    cat /task_outputs/words.txt | sort | uniq -c | sort -rn

## NOTES

- All paths must be absolute.
- Reads from stdin when no file path is provided.
- Only filters adjacent duplicates. Use `sort` first for global deduplication.

## SEE ALSO

sort, wc, grep
