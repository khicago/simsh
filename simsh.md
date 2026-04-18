# simsh Runtime Profile

## Shell Runtime
- deterministic script operators: `;`, `&&`, `||`, `|`
- deterministic redirections: `>`, `>>`, `<`, `<<`
- command mounts: `/sys/bin` (builtin), `/bin` (injected external)
- parent directories of mount points are exposed as synthetic virtual directories (e.g. `/sys`)
- mount-backed virtual paths are immutable (no write/edit/remove/mkdir/cp/mv on those paths)
- default aliases: `ll` -> `ls -l`, `fm` -> `frontmatter`
- optional rc bootstrap supports read-only `export` + `alias` statements
- builtin commands: `basename`, `cat`, `cd`, `cp`, `date`, `diff`, `dirname`, `echo`, `edit`, `env`, `find`, `frontmatter`, `glob`, `grep`, `head`, `json`, `ls`, `man`, `mkdir`, `mounts`, `mv`, `pwd`, `rg`, `rm`, `rmdir`, `sed`, `sort`, `tail`, `tee`, `touch`, `tree`, `type`, `uniq`, `view`, `wc`, `which`
- profile gates: `core-strict`, `bash-plus`, `zsh-lite`

### Builtin Command Reference

#### basename
    basename [--] PATH
- Prints the final component of a resolved virtual path.
- Relative paths resolve against the session working directory before basename is applied.

Examples:
    basename /task_outputs/report.md

#### cat
    cat [-n] PATH | cat (stdin passthrough)
- Use cat with no args to pass stdin through a pipeline.
- Use -n to display line numbers.

Examples:
    cat /knowledge_base/readme.md
    cat -n /knowledge_base/readme.md
    echo hello | cat

#### cd
    cd [PATH]
- Changes the session-local virtual working directory.
- With no argument, cd returns to the virtual root.

Examples:
    cd /task_outputs
    cd ../knowledge_base
    cd

#### cp
    cp [--confirm] [--json] SRC_PATH DEST_PATH
- Copies a file from source to destination.
- Use --confirm or --json when you want explicit success feedback without changing the default silent behavior.
- Projection-backed virtual paths are not valid copy operands; factual mounts may opt into transfer support.

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
    diff PATH1 PATH2
- Compares two files line by line.
- Exit code 0 if identical, 1 if different.

Examples:
    diff /knowledge_base/v1.md /knowledge_base/v2.md

#### dirname
    dirname [--] PATH
- Prints the parent of a resolved virtual path.
- Relative paths resolve against the session working directory before dirname is applied.

Examples:
    dirname /task_outputs/report.md

#### echo
    echo [ARGS...]
- Use echo for deterministic plain text output.

Examples:
    echo hello world
    echo "report data" | tee /task_outputs/report.txt

#### edit
    edit [--json] [--confirm] [--all] [--count] --old OLD [--new NEW] [--] PATH
- Default replace requires a unique OLD snippet so edits cannot silently hit the wrong copy.
- Use --all only when every match should change, and --count to inspect matches without writing.
- Use --confirm or --json for an explicit summary; default success is silent like other mutations.

Examples:
    edit --old TODO --new DONE /task_outputs/notes.md
    edit --json --old unique --new patched /task_outputs/report.md
    edit --count --old foo /task_outputs/dup.txt
    edit --all --old foo --new bar /task_outputs/dup.txt

#### env
    env [--json] [--split] [KEY]
- Use env PATH to inspect command search order.
- Use --split with one key such as PATH when you want list-like values one item per line.
- Use --json for machine-readable environment output.

Examples:
    env
    env PATH

#### find
    find [DIR] -name PATTERN [-o -name PATTERN ...] [--fmt jsonl] [-exec CMD {} ';'|+]
- Use -o to combine multiple -name patterns.
- Use --fmt jsonl for explicit machine-readable path records without changing the default text stream.
- -exec ... + batches matched paths in one invocation.

Examples:
    find / -name "*.md"
    find /task_outputs -name "*.json"
    find /knowledge_base -name "*.md" -exec cat {} ;

#### frontmatter
    frontmatter <stat|get|print> ...
- Use stat to inspect frontmatter presence across many files.
- Use get --key KEY to read a specific top-level key value.
- Use print -C N to print key/frontmatter with line context.

Examples:
    frontmatter stat docs -r
    frontmatter get --key title /knowledge_base/notes.md
    frontmatter print --key sop -C 2 docs/must-sop.md

#### glob
    glob [--fmt jsonl] [--] PATTERN [PATH ...]
- A pattern without a slash matches basenames recursively, so glob '*.go' finds every Go file under the target.
- Use ** in a path pattern for explicit directory wildcards.
- Use --fmt jsonl for path records without changing the default path-per-line stream.

Examples:
    glob '*.go'
    glob '**/*.md' /knowledge_base
    glob --fmt jsonl 'pkg/*.go' /task_outputs/project

#### grep
    grep [-E|-F] [-r] [-l] [-A N] [-B N] [-C N] [--fmt jsonl] PATTERN [PATH]
- Use -r for directory search and -l to list matched files only.
- Use --fmt jsonl when you want flat machine-readable records without changing the default text output.
- Context flags -A/-B/-C include neighboring lines around each match.

Examples:
    grep TODO /task_outputs/notes.md
    grep -E "func\s+" -r /knowledge_base
    grep -l error -r /task_outputs

#### head
    head [-n N|-N] [PATH]
- Use stdin input when no file path is provided.
- -n accepts non-negative integers only.

Examples:
    head /task_outputs/report.md
    head -n 5 /task_outputs/report.md

#### json
    json <stat|get|keys|len> ...
- Use json stat to inspect JSON shape across files and directories.
- Use json get --path QUERY to extract one or a small set of JSON subtrees without dumping the whole file.
- Use json keys and json len for narrow structure-aware queries instead of re-reading full JSON into the model.

Examples:
    json stat /task_outputs/data.json
    json stat -r --fmt json /workspace
    json get --path items[0].name /task_outputs/data.json
    json get --path meta.author --path items[1].name --fmt json /task_outputs/data.json
    json keys --path meta /task_outputs/data.json
    json len -r --path items /workspace

#### ls
    ls [-a] [-R] [-l] [--fmt text|md|json] [PATH...]
- Use -l to include semantic metadata.
- Use --fmt md|json only with -l and a single non-recursive target.
- Use -R for recursive traversal.

Examples:
    ls /
    ls -l /task_outputs
    ls -R /knowledge_base

#### man
    man [-v] [-l|--list] [--fmt text|json] <command>
- man CMD shows summary with tips and examples.
- man -v CMD shows full documentation with command-specific details.
- man --list shows the builtin and external command catalog; add --fmt json for a machine-readable view.

Examples:
    man ls
    man -v grep
    man --list

#### mkdir
    mkdir [--confirm] [--json] [-p] PATH...
- Creates directories. -p creates parent directories as needed.
- Use --confirm or --json when you want explicit success feedback without changing the default silent behavior.
- Projection-backed virtual paths are immutable; only mounts that declare mutate support accept directory creation.

Examples:
    mkdir /task_outputs/reports
    mkdir -p /task_outputs/a/b/c

#### mounts
    mounts [--fmt text|json]
mounts refresh [--require-narrow] MOUNT_POINT...
- mounts shows the active virtual mount contract surface visible to the runtime.
- mounts refresh is explicit and target-scoped; ordinary reads still do not trigger hidden refresh.
- Use --fmt json for machine-readable mount point/profile/status or refresh data.

Examples:
    mounts
    mounts --fmt json
    mounts refresh /knowledge_base
    mounts refresh --require-narrow /knowledge_base/guide.md

#### mv
    mv [--confirm] [--json] SRC_PATH DEST_PATH
- Moves a file from source to destination.
- Use --confirm or --json when you want explicit success feedback without changing the default silent behavior.
- Projection-backed virtual paths are not valid move operands; factual mounts may opt into transfer support.

Examples:
    mv /task_outputs/draft.md /task_outputs/final.md

#### pwd
    pwd
- Prints the current session-local virtual working directory.

Examples:
    pwd

#### rg
    rg [-F] [-i|-S] [-l] [-g GLOB]... [-A N] [-B N] [-C N] [--fmt jsonl] PATTERN [PATH ...] | rg --files [-g GLOB]... [PATH ...]
- rg searches recursively by default and falls back to the current working directory when no path is given.
- Use --files to list searchable files, optionally narrowed with one or more -g globs.
- Use --fmt jsonl as the canonical structured mode; --json is accepted only as a compatibility alias.

Examples:
    rg "TODO" /task_outputs
    rg -g "*.md" "hello" /knowledge_base
    rg --files -g "*.json" /task_outputs

#### rm
    rm [--confirm] [--json] PATH...
- Removes files. Does not support directory removal.
- Use --confirm or --json when you want explicit success feedback without changing the default silent behavior.
- Projection-backed virtual paths are immutable; only mounts that declare mutate support accept removals.

Examples:
    rm /task_outputs/old.md
    rm /task_outputs/temp1.txt /task_outputs/temp2.txt

#### rmdir
    rmdir [--confirm] [--json] PATH...
- Removes empty directories only.
- Use --confirm or --json when you want explicit success feedback without changing the default silent behavior.
- Use rm for files; rmdir rejects non-empty directories.

Examples:
    rmdir /task_outputs/empty_dir
    rmdir /task_outputs/cache/a /task_outputs/cache/b

#### sed
    sed -i [--json] 's/old/new/[g]' PATH | sed -n 'Np'|'M,Np' [PATH]
- Only a focused subset of sed is supported for deterministic behavior.
- Print mode stays text-first; use --json only with -i when you want a machine-readable mutation summary.
- Use -n with Np or M,Np to print line ranges.

Examples:
    sed -i 's/draft/final/' /task_outputs/report.md
    sed -n '10,20p' /knowledge_base/data.txt

#### sort
    sort [-r] [-n] [-u] [PATH]
- Sorts lines. Use stdin when no file is given.
- -r reverses order, -n sorts numerically, -u removes duplicates.

Examples:
    sort /task_outputs/names.txt
    sort -rn /task_outputs/scores.txt

#### tail
    tail [-n N|-N] [PATH]
- Use stdin input when no file path is provided.
- -n accepts non-negative integers only.

Examples:
    tail /task_outputs/report.md
    tail -n 5 /task_outputs/report.md

#### tee
    echo data | tee [--confirm] [--json] [-a] PATH
- Use -a to append instead of replacing file content.
- Default output preserves stdin passthrough; --confirm and --json replace stdout with an explicit success summary.
- tee requires stdin, usually from a pipeline or heredoc.

Examples:
    echo content | tee /task_outputs/file.md
    echo more | tee -a /task_outputs/file.md

#### touch
    touch [--json] PATH...
- Creates empty files if they do not exist.
- Use --json when you want explicit created/already_exists feedback without changing the default silent behavior.

Examples:
    touch /task_outputs/notes.md

#### tree
    tree [-a] [-L N] [--fmt outline|ascii|json] [PATH...]
- Default output is an outline optimized for dual readability and low token noise.
- Use --fmt ascii for classic branch rendering or --fmt json for machine-readable entries.
- Use -L to limit output depth for large directories.
- Use -a to include hidden entries.

Examples:
    tree /
    tree -L 2 /task_outputs
    tree -a /knowledge_base

#### type
    type [--json] COMMAND...
- Reports whether a command resolves to alias, builtin, or external.
- Lookup order matches command execution: alias -> /sys/bin builtin -> /bin custom external.
- Use --json when you want machine-readable command-resolution records.

Examples:
    type ls
    type report_tool
    type ll

#### uniq
    uniq [-c] [-d] [PATH]
- Removes adjacent duplicate lines. Use stdin when no file is given.
- -c prefixes lines with occurrence count, -d only prints duplicates.

Examples:
    sort /task_outputs/log.txt | uniq -c
    uniq -d /task_outputs/sorted.txt

#### view
    view [--start N] [--lines N] [--fmt jsonl] [--] PATH
- view prints a numbered window so agents can inspect a slice instead of dumping the whole file.
- Line numbers are 1-based. Default window is 80 lines from the start of the file.
- Use --fmt jsonl for {line,text} records, not --json.

Examples:
    view /knowledge_base/readme.md
    view --start 20 --lines 40 /task_outputs/report.md
    view --fmt jsonl --start 1 --lines 10 /task_outputs/data.json

#### wc
    wc [--json] [-l] [-w] [-c] [PATH]
- Single-metric modes keep bare numeric output for pipeline composability.
- Default multi-metric output uses compact labels, and --json provides an explicit structured mode.

Examples:
    wc /task_outputs/report.md
    wc -l /knowledge_base/data.txt

#### which
    which [--fmt json] COMMAND...
- Search order is alias -> /sys/bin (system builtin) -> /bin (custom external).
- Use --fmt json when you want machine-readable lookup results without changing the default text output.

Examples:
    which ls
    which report_tool
    which ll

## Filesystem Model
- virtual root `/` exposes purpose-oriented directories only:
- `/task_outputs`: durable deliverables and final outputs (writable)
- `/temp_work`: temporary intermediates and disposable artifacts (writable)
- `/knowledge_base`: read-only references and knowledge files
- path policy is zone-scoped and blocks path escape by host-root relative checks
- metadata is AI-oriented: kind, line count, frontmatter rows, speaker-like rows, relevance

## Runtime Composition
- package `engine`: command language, parser, dispatch, traces, and prepared execution
- package `builtin`: default ACI command surface
- package `fs`: AI-oriented virtual filesystem contract and adapters
- package `sh`: compatibility wrapper plus generated runtime-profile text
- package `cmd`: runtime entrypoints (CLI/TUI/serve) calling `engine/runtime`
- engine runtime wires `engine + builtin + fs` into a request-safe runtime instance
- CLI default interactive mode uses TUI and also exposes `serve -P` for HTTP integration
