---
name: ls
synopsis: "ls [-a] [-R] [-l] [ABS_PATH...]"
category: navigation
---

# ls -- list directory contents

## SYNOPSIS

    ls [-a] [-R] [-l] [ABS_PATH...]

## DESCRIPTION

List files and directories at the given absolute path(s). If no path is given,
lists the virtual root directory. Entries are returned one per line.

## FLAGS

- `-a` -- Include entries starting with `.` (dot files), adding `.` and `..` entries.
- `-R` -- Recursive listing. Traverses subdirectories breadth-first, printing each directory header followed by its children.
- `-l` -- Long format. Shows semantic metadata per entry: mode, kind, line count, and path.

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
- Long format columns: `MODE kind=TYPE lines=N PATH`
- Mode is `d` for directories, `-` for files.
- Line count shows `-` when unavailable.

## SEE ALSO

find, cat
