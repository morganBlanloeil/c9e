#!/usr/bin/env bash
set -euo pipefail

# Pre-commit hook: update project documentation using Claude CLI
# Runs before each git commit via Claude Code PreToolUse hook

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
cd "${PROJECT_ROOT}"

# Escape hatch: skip if marker file exists
if [[ -f ".claude/.skip-doc-update" ]]; then
  exit 0
fi

# Check if claude CLI is available
if ! command -v claude &>/dev/null; then
  exit 0
fi

# Check if there are changes since last doc update
MARKER_FILE=".claude/.last-doc-update"
CURRENT_HASH="$(git rev-parse HEAD 2>/dev/null || echo "unknown")"
HAS_UNCOMMITTED="$(git status --porcelain 2>/dev/null | head -1)"

if [[ -f "${MARKER_FILE}" ]]; then
  LAST_HASH="$(cat "${MARKER_FILE}")"
  if [[ "${CURRENT_HASH}" == "${LAST_HASH}" && -z "${HAS_UNCOMMITTED}" ]]; then
    exit 0
  fi
fi

# Ensure log directory exists
mkdir -p .claude/logs

# Build prompt as a variable to avoid multiline argument issues
PROMPT="You are a documentation updater for the c9e project (a Claude Code monitoring TUI dashboard). Read the current source code and update README.md and CONTRIBUTING.md to accurately reflect the current state of the codebase. Specifically check and update: 1) README.md: features list, TUI keyboard shortcuts table, columns table, data sources table. 2) CONTRIBUTING.md: project structure tree, architecture description, data sources documentation. Rules: Only make changes if the docs are actually out of date. Keep the same markdown style and structure. Do NOT add or remove sections, only update existing content. Do NOT modify badges, installation instructions, or release info. Be conservative: if unsure, don't change it. Read the key source files: cmd/c9e/main.go, internal/tui/model.go, internal/tui/views.go, internal/display/display.go, internal/tui/data.go"

# Run Claude to update docs (print mode, no interaction)
if claude --allowedTools "Read,Edit,Glob,Grep,Bash" -p "${PROMPT}" \
  >> .claude/logs/update-docs.log 2>&1; then
  # Update marker only on success
  echo "${CURRENT_HASH}" > "${MARKER_FILE}"
fi

exit 0
