---
name: pwd
synopsis: "pwd"
category: navigation
---

# pwd -- print current working directory

## SYNOPSIS

    pwd

## DESCRIPTION

Print the current session-local virtual working directory used by the runtime.

## EXAMPLES

Print the current working directory:

    pwd

## NOTES

- Does not accept any flags or positional arguments.
- Output is always an absolute path.
- `pwd` reflects the current `cd` state inside the runtime or session.

## SEE ALSO

cd, env, ls
