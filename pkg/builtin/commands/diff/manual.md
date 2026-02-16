---
name: diff
synopsis: "diff ABS_FILE1 ABS_FILE2"
category: text-processing
---

# diff -- compare two files

## SYNOPSIS

    diff ABS_FILE1 ABS_FILE2

## DESCRIPTION

Compare two files line by line and output the differences in unified diff
format. Shows lines that are added, removed, or changed between the two files.

## EXAMPLES

Compare two files:

    diff /knowledge_base/v1.md /knowledge_base/v2.md

Compare original and modified:

    diff /task_outputs/draft.md /task_outputs/final.md

## NOTES

- Both paths must be absolute.
- Output uses unified diff format.
- Exit code 0 if files are identical, 1 if they differ.

## SEE ALSO

cat, grep, sed
