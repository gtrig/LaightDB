---
name: laightdb-rolling-context
description: >-
  Proactively loads relevant LaightDB context at the start of substantive work and
  persists concise rolling notes before wrapping up, using LaightDB MCP tools
  without waiting for the user to say "store" or "search". Use whenever this
  workspace has the LaightDB MCP server enabled and the user is doing multi-step
  implementation, debugging, or design that should survive across sessions.
---

# LaightDB rolling context

## Defaults

- Prefer **search** before **store** when continuing prior work: run `search_context` with a short query (repo name, ticket id, branch name, feature keyword, or path prefix) and `detail` of `metadata` or `summary` first; use `full` only when verbatim text is required.
- After meaningful progress (decisions, APIs, constraints, open questions), call `store_context` with a stable `collection` (for example `<repo-name>/rolling`) and `metadata` keys such as `topic`, `branch`, or `ticket` so later searches hit reliably.
- Do not store secrets, tokens, or credentials; use references (paths, ticket ids) instead.

## At the start of a task

1. Infer 1–3 keywords for the current goal (file area, bug title, ticket, or epic).
2. Call `search_context` with those keywords (optional `collection` if the project uses one consistently).
3. Briefly incorporate any high-signal hits into the plan; if nothing useful is returned, proceed without mentioning LaightDB unless storage is clearly needed later.

## Before ending a substantive turn

If this turn changed architecture, contracts, or left follow-ups, call `store_context` with a compact summary: what changed, what is uncertain, and suggested next steps. Skip storage for trivial one-line edits unless the user asked for persistence.

## Optional graph use

When relationships between stored items matter (dependencies, parent/child design), use `link_context` or `graph_search` as described in the main LaightDB MCP skill.

## Hook interaction

If hooks from **`integrations/cursor`** are installed:

- **`sessionStart`** injects **`additional_context`** with a short policy: use LaightDB MCP for persistent memory, **`search_context`** at task start, **`store_context`** before ending substantive turns when worth preserving.
- **`beforeSubmitPrompt`** may add **`additional_context`** with LaightDB hits from the **`search_context` MCP tool** (streamable HTTP `/mcp`, not REST `/v1/search`). Treat injected lines as optional context; still call `search_context` / `get_context` when you need full control over the query or `detail` level.

Do not rely on a **`stop`** hook with **`followup_message`** for storage reminders—older setups did that and Cursor could chain into repeated agent turns (**infinite loops**). End-of-turn **`store_context`** is driven by this skill and the **`sessionStart`** policy, not by `stop`.
