#!/usr/bin/env sh
# Cursor project hook: runs on agent sessionStart.
# Injects policy so the model uses LaightDB MCP as persistent memory (search + store).
# stdin: hook JSON (ignored); stdout: hook response JSON
set -e

command -v jq >/dev/null 2>&1 || { echo '{"additional_context":""}'; exit 0; }

TEXT='LaightDB MCP is your persistent memory for this workspace.

- At the start of substantive work: call search_context with 1–3 keywords (repo, ticket, feature, or path) and detail metadata or summary before store_context.
- Before each chat message you send, the beforeSubmitPrompt project hook may inject additional_context from search_context (streamable HTTP /mcp at LAIGHTDB_MCP_URL). Treat those lines as hints; call search_context or get_context yourself when you need a specific query or detail level.
- Before ending a substantive turn: if the work is worth preserving (decisions, APIs, constraints, follow-ups), call store_context with a stable collection (e.g. <repo>/rolling) and metadata keys like topic or branch. Do not store secrets or credentials.'

jq -n --arg text "$TEXT" '{additional_context:$text}'
