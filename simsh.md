# simsh Runtime Profile

## Shell Runtime
- deterministic script operators: `;`, `&&`, `||`, `|`
- deterministic redirections: `>`, `>>`, `<`, `<<`
- command mounts: `/sys/bin` (builtin), `/bin` (injected external)
- parent directories of mount points are exposed as synthetic virtual directories (e.g. `/sys`)
- mount-backed virtual paths are immutable (no write/edit/remove/mkdir/cp/mv on those paths)
- builtin commands: `cat`, `cp`, `date`, `diff`, `echo`, `env`, `find`, `grep`, `head`, `ls`, `man`, `mkdir`, `mv`, `rm`, `sed`, `sort`, `tail`, `tee`, `touch`, `uniq`, `wc`
- profile gates: `core-strict`, `bash-plus`, `zsh-lite`

### Builtin Command Reference

#### cat
    cat [-n] ABS_FILE | cat (stdin passthrough)
- Use cat with no args to pass stdin through a pipeline.
- Use -n to display line numbers.

Examples:
    cat /knowledge_base/readme.md
    cat -n /knowledge_base/readme.md
    echo hello | cat

#### cp
    cp SRC_ABS DEST_ABS
- Copies a file from source to destination. Both paths must be absolute.
- Mount-backed virtual paths are immutable and not valid copy operands.

Examples:
    cp /knowledge_base/template.md /task_outputs/report.md
    cp /task_outputs/input.txt /task_outputs/output.txt

#### date
    date [-u] [+%Y-%m-%d|+%F|+%T|+%s|...]
- Use -u for UTC time output.
- Format specifiers follow the supported simsh subset only.

Examples:
    date
    date -u
    date +%F
    date +%s

#### diff
    diff ABS_FILE1 ABS_FILE2
- Compares two files line by line.
- Exit code 0 if identical, 1 if different.

Examples:
    diff /knowledge_base/v1.md /knowledge_base/v2.md

#### echo
    echo [ARGS...]
- Use echo for deterministic plain text output.

Examples:
    echo hello world
    echo "report data" | tee /task_outputs/report.txt

#### env
    env [KEY]
- Use env PATH to inspect command search order.

Examples:
    env
    env PATH

#### find
    find [ABS_DIR] -name PATTERN [-o -name PATTERN ...] [-exec CMD {} ';'|+]
- Use -o to combine multiple -name patterns.
- -exec ... + batches matched paths in one invocation.

Examples:
    find / -name "*.md"
    find /task_outputs -name "*.json"
    find /knowledge_base -name "*.md" -exec cat {} ;

#### grep
    grep [-E|-F] [-r] [-l] [-A N] [-B N] [-C N] PATTERN [ABS_PATH]
- Use -r for directory search and -l to list matched files only.
- Context flags -A/-B/-C include neighboring lines around each match.

Examples:
    grep TODO /task_outputs/notes.md
    grep -E "func\s+" -r /knowledge_base
    grep -l error -r /task_outputs

#### head
    head [-n N|-N] [ABS_FILE]
- Use stdin input when no file path is provided.
- -n accepts non-negative integers only.

Examples:
    head /task_outputs/report.md
    head -n 5 /task_outputs/report.md

#### ls
    ls [-a] [-R] [-l] [ABS_PATH...]
- Use -l to include semantic metadata.
- Use -R for recursive traversal.

Examples:
    ls /
    ls -l /task_outputs
    ls -R /knowledge_base

#### man
    man [-v] [-l|--list] <command>
- man CMD shows summary with tips and examples.
- man -v CMD shows full documentation.
- man --list shows all available commands.

Examples:
    man ls
    man -v grep
    man --list

#### mkdir
    mkdir [-p] ABS_PATH...
- Creates directories. -p creates parent directories as needed.
- Mount-backed virtual paths are immutable and cannot be created.

Examples:
    mkdir /task_outputs/reports
    mkdir -p /task_outputs/a/b/c

#### mv
    mv SRC_ABS DEST_ABS
- Moves a file from source to destination. Both paths must be absolute.
- Mount-backed virtual paths are immutable and cannot be moved.

Examples:
    mv /task_outputs/draft.md /task_outputs/final.md

#### rm
    rm ABS_PATH...
- Removes files. Does not support directory removal.
- Mount-backed virtual paths are immutable and cannot be removed.

Examples:
    rm /task_outputs/old.md
    rm /task_outputs/temp1.txt /task_outputs/temp2.txt

#### sed
    sed -i 's/old/new/[g]' ABS_FILE | sed -n 'Np'|'M,Np' [ABS_FILE]
- Only a focused subset of sed is supported for deterministic behavior.
- Use -n with Np or M,Np to print line ranges.

Examples:
    sed -i 's/draft/final/' /task_outputs/report.md
    sed -n '10,20p' /knowledge_base/data.txt

#### sort
    sort [-r] [-n] [-u] [ABS_FILE]
- Sorts lines. Use stdin when no file is given.
- -r reverses order, -n sorts numerically, -u removes duplicates.

Examples:
    sort /task_outputs/names.txt
    sort -rn /task_outputs/scores.txt

#### tail
    tail [-n N|-N] [ABS_FILE]
- Use stdin input when no file path is provided.
- -n accepts non-negative integers only.

Examples:
    tail /task_outputs/report.md
    tail -n 5 /task_outputs/report.md

#### tee
    echo data | tee [-a] ABS_FILE
- Use -a to append instead of replacing file content.
- tee requires stdin, usually from a pipeline or heredoc.

Examples:
    echo content | tee /task_outputs/file.md
    echo more | tee -a /task_outputs/file.md

#### touch
    touch ABS_PATH...
- Creates empty files if they do not exist.

Examples:
    touch /task_outputs/notes.md

#### uniq
    uniq [-c] [-d] [ABS_FILE]
- Removes adjacent duplicate lines. Use stdin when no file is given.
- -c prefixes lines with occurrence count, -d only prints duplicates.

Examples:
    sort /task_outputs/log.txt | uniq -c
    uniq -d /task_outputs/sorted.txt

#### wc
    wc [-l] [-w] [-c] [ABS_FILE]
- Counts lines, words, and bytes. Use stdin when no file is given.

Examples:
    wc /task_outputs/report.md
    wc -l /knowledge_base/data.txt

## Filesystem Model
- virtual root `/` exposes purpose-oriented directories only:
- `/task_outputs`: durable deliverables and final outputs (writable)
- `/temp_work`: temporary intermediates and disposable artifacts (writable)
- `/knowledge_base`: read-only references and knowledge files
- path policy is zone-scoped and blocks path escape by host-root relative checks
- metadata is AI-oriented: kind, line count, frontmatter rows, speaker-like rows, relevance

## Runtime Composition
- package `engine`: runtime composition layer (`sh + fs + policy/profile`)
- package `sh`: command language and execution semantics
- package `fs`: AI-oriented virtual filesystem contract and adapters
- package `cmd`: runtime entrypoints (CLI/TUI/serve) calling `engine`
- engine runtime wires `sh + fs` into a request-safe runtime instance
- CLI default interactive mode uses TUI and also exposes `serve -P` for HTTP integration
