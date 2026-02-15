# simsh Runtime Profile

## Shell Runtime
- deterministic script operators: `;`, `&&`, `||`, `|`
- deterministic redirections: `>`, `>>`, `<`, `<<`
- command mounts: `/sys/bin` (builtin), `/bin` (injected external)
- builtin commands: `cat`, `date`, `echo`, `env`, `find`, `grep`, `head`, `ls`, `man`, `sed`, `tail`, `tee`
- profile gates: `core-strict`, `bash-plus`, `zsh-lite`

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
