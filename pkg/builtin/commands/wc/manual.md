---
name: wc
synopsis: "wc [--json] [-l] [-w] [-c] [PATH]"
category: text-processing
---

# wc -- word, line, and byte count

## SYNOPSIS

    wc [--json] [-l] [-w] [-c] [PATH]
    COMMAND | wc [--json] [-l] [-w] [-c]

## DESCRIPTION

Count lines, words, and bytes in a file or stdin. Without flags, displays
all three counts. With flags, displays only the requested counts.

## FLAGS

- `--json` -- Emit a JSON object instead of text output.
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

Structured output:

    wc --json /task_outputs/report.md

## NOTES

- Paths may be absolute or relative to the current virtual working directory.
- Reads from stdin when no file path is provided.
- A word is a sequence of non-whitespace characters.
- Single-metric modes keep bare numeric output for pipeline composability.
- Multi-metric text output uses compact labels such as `lines=2 words=10 bytes=57`.
- `--json` returns only the fields requested by the active metric flags.

## SEE ALSO

cat, grep, head
