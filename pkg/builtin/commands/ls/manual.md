---
name: ls
synopsis: "ls [-a] [-R] [-l] [--fmt text|md|json] [ABS_PATH...]"
category: navigation
---

# ls -- list directory contents

## SYNOPSIS

    ls [-a] [-R] [-l] [--fmt text|md|json] [ABS_PATH...]

## DESCRIPTION

List files and directories at the given absolute path(s). If no path is given,
lists the virtual root directory. Entries are returned one per line.

## FLAGS

- `-a` -- Include entries starting with `.` (dot files), adding `.` and `..` entries.
- `-R` -- Recursive listing. Traverses subdirectories breadth-first, printing each directory header followed by its children.
- `-l` -- Long format. Shows semantic metadata per entry: mode, access, kind, line count, and path.
- `--fmt text|md|json` -- Output format for long listings (requires `-l`). `md` and `json` currently only support a single non-recursive target.

Flags may be combined (e.g. `ls -alR`).

## EXAMPLES

List the root directory:

    ls /

Long format listing of task outputs:

    ls -l /task_outputs

Recursive listing of knowledge base:

    ls -R /knowledge_base

Show hidden entries in long format:

    ls -al /

List a specific file (returns the path):

    ls /task_outputs/report.md

## NOTES

- All paths must be absolute (start with `/`).
- Long format columns: `MODE ACCESS KIND LINES PATH`
- Mode is `d` for directories, `-` for files.
- Access is `ro|rw` derived from SSOT path metadata (mount-backed + synthetic mount parents are `ro`).
- Line count shows `-` when unavailable.
- `ls -l` prints a legend line once at the end: `# columns: mode access kind lines path`

## SEE ALSO

find, cat
