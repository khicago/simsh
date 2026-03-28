#!/usr/bin/env bash
set -euo pipefail
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
: "${BAGAKIT_LONG_RUN_SKILL_DIR:=/Users/bytedance/proj/priv/bagakit/skills/dist_local/bagakit-long-run}"
export BAGAKIT_LONG_RUN_SKILL_DIR
exec bash "${script_dir}/.gen/check_and_resume.sh" "$@"
