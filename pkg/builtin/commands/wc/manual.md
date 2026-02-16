---
name: wc
synopsis: "wc [-l] [-w] [-c] [ABS_FILE]"
category: text-processing
---

# wc -- word, line, and byte count

## SYNOPSIS

    wc [-l] [-w] [-c] [ABS_FILE]
    COMMAND | wc [-l] [-w] [-c]

## DESCRIPTION

Count lines, words, and bytes in a file or stdin. Without flags, displays
all three counts. With flags, displays only the requested counts.

## FLAGS

- `-l` -- Count lines only.
- `-w` -- Count words only.
- `-c` -- Count bytes only.

## EXAMPLES

Count all metrics for a file:

    wc /task_outputs/report.md

Count lines only:

    wc -l /knowledge_base/data.txt

Count words from stdin:

    cat /task_outputs/notes.md | wc -w

Combine flags:

    wc -lw /task_outputs/report.md

## NOTES

- All paths must be absolute.
- Reads from stdin when no file path is provided.
- A word is a sequence of non-whitespace characters.

## SEE ALSO

cat, grep, head
