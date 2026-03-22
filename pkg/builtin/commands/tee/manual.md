---
name: tee
synopsis: "tee [--confirm] [--json] [-a] PATH"
category: file-management
---

# tee -- write stdin to file and stdout

## SYNOPSIS

    COMMAND | tee [--confirm] [--json] [-a] PATH

## DESCRIPTION

Read from standard input and write to both a file and standard output.
Requires stdin input, typically from a pipeline or heredoc.

Default success output keeps stdin passthrough so `tee` remains a strong
pipeline primitive. `--confirm` and `--json` replace that passthrough with an
explicit success summary.

## FLAGS

- `--confirm` -- Emit a low-noise success summary instead of passthrough stdout.
- `--json` -- Emit a machine-readable write summary instead of passthrough stdout.
- `-a` -- Append to the file instead of overwriting.

## EXAMPLES

Write pipeline output to a file:

    echo "report content" | tee /task_outputs/report.md

Append to an existing file:

    echo "additional line" | tee -a /task_outputs/report.md

Chain with other commands:

    cat /knowledge_base/data.txt | grep "important" | tee /task_outputs/filtered.txt

Emit confirmation instead of passthrough:

    echo "report content" | tee --confirm /task_outputs/report.md

Emit JSON summary instead of passthrough:

    echo "report content" | tee --json /task_outputs/report.md

## NOTES

- Requires stdin input; cannot be used standalone.
- Writes to exactly one file.
- Paths may be absolute or relative to the current virtual working directory.
- Write operations are subject to zone policy checks.
- Default mode keeps `stdout == stdin`.
- `--confirm` and `--json` are terminal-sink modes; they replace stdout passthrough.

## SEE ALSO

echo, cat
