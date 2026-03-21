# Agent Runtime Paper Notes

Date: 2026-03-22

## Downloaded PDFs
- `knowledge_base/papers/agent-runtime/osworld-2404.07972.pdf`
- `knowledge_base/papers/agent-runtime/swe-agent-2405.15793.pdf`
- `knowledge_base/papers/agent-runtime/programming-with-pixels-2502.18525.pdf`

## Shortlist

### 1. OSWorld
- Focus: realistic computer-use benchmark across OSes, apps, and file workflows.
- Main lesson: real computer tasks are important, but full realism is still very hard for agents.
- Relevance to `simsh`: supports building a more explicit and lower-noise runtime rather than a full GUI replica.

### 2. SWE-agent
- Focus: agent-computer interface design for software engineering agents.
- Main lesson: a carefully designed ACI beats raw shell access for agents.
- Relevance to `simsh`: strongest support for treating the kernel as an LM-facing runtime, not as a shell compatibility project.

### 3. Programming with Pixels
- Focus: whether computer-use agents can do software engineering inside an IDE.
- Main lesson: pure visual interaction is weak; direct file editing and bash access create a large step-change in performance.
- Relevance to `simsh`: strongest support for investing in file-system and shell contracts rather than GUI realism.

## Synthesis For simsh

The three papers point in the same direction:

- Full computer realism matters for evaluation, but not every layer of that realism needs to be exposed directly to the agent.
- The highest-value interface primitives for software work are file editing and shell operations.
- Interface quality matters as much as model quality.
- Execution environments should be judged by whether agents can act predictably, not by how faithfully they emulate a full desktop or POSIX shell.

## Kernel Implications

### P0: Make file-system semantics trustworthy
- Path capabilities must be real, not advisory.
- Read/write/remove/edit/move behavior must match the visible contract.
- Trace data must faithfully report what the file system actually did.

### P1: Make the action surface agent-native
- Prefer concise, guarded, LM-friendly operations over a broad shell surface.
- Keep error messages concrete and classifiable.
- Make repository navigation and file editing especially reliable.

### P2: Keep realism where it helps
- Keep real side effects and persistent state.
- Keep session continuity.
- Avoid GUI- or POSIX-only complexity that increases noise without improving agent performance.

## Next Research Directions
- Compare `simsh` command/file workflows to SWE-agent's ACI command design.
- Evaluate whether `ExecutionTrace` already captures the side effects an agent planner needs.
- Design a small benchmark of `simsh`-native file-system tasks that approximates the file-workflow parts of OSWorld and PwP without requiring a full desktop UI.
