#!/usr/bin/env bash
set -euo pipefail
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
: "${BAGAKIT_LONG_RUN_SKILL_DIR:=/Users/bytedance/proj/priv/bagakit/skills/dist_local/bagakit-long-run}"
if [[ -z "${BAGAKIT_AGENT_CMD:-}" ]]; then
  BAGAKIT_AGENT_CMD='codexL exec --dangerously-bypass-approvals-and-sandbox {prompt_text}'
fi
export BAGAKIT_LONG_RUN_SKILL_DIR
export BAGAKIT_AGENT_CMD
exec bash "${script_dir}/.gen/ralphloop-runner.sh" "$@"
