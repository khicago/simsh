---
name: man
synopsis: "man [-v] [-l|--list] <command>"
category: system
---

# man -- display command manual

## SYNOPSIS

    man <command>
    man -v <command>
    man -l
    man --list

## DESCRIPTION

Display documentation for builtin and external commands. Supports progressive
disclosure: the default mode shows a summary with tips and examples, while
verbose mode shows the full embedded documentation.

## FLAGS

- `-v` -- Verbose mode. Show the full detailed manual instead of the summary.
- `-l`, `--list` -- List all available commands with one-line descriptions.

## EXAMPLES

Show summary for ls:

    man ls

Show full documentation for grep:

    man -v grep

List all available commands:

    man --list

Show manual for an external command:

    man my_custom_tool

## NOTES

- Builtin commands are looked up first, then external commands.
- The `--list` mode shows both builtin and external commands.
- Command names can be given with or without path prefix.

## SEE ALSO

env, ls
