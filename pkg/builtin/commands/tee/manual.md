---
name: tee
synopsis: "tee [-a] ABS_FILE"
category: file-management
---

# tee -- write stdin to file and stdout

## SYNOPSIS

    COMMAND | tee [-a] ABS_FILE

## DESCRIPTION

Read from standard input and write to both a file and standard output.
Requires stdin input, typically from a pipeline or heredoc.

## FLAGS

- `-a` -- Append to the file instead of overwriting.

## EXAMPLES

Write pipeline output to a file:

    echo "report content" | tee /task_outputs/report.md

Append to an existing file:

    echo "additional line" | tee -a /task_outputs/report.md

Chain with other commands:

    cat /knowledge_base/data.txt | grep "important" | tee /task_outputs/filtered.txt

## NOTES

- Requires stdin input; cannot be used standalone.
- Writes to exactly one file.
- All paths must be absolute.
- Write operations are subject to zone policy checks.

## SEE ALSO

echo, cat
