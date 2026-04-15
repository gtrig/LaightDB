# LaightDB Cursor integration (manual install)

Bundled files for **Cursor**: a **skill** (rolling context behavior), a **`sessionStart` hook** that injects **`additional_context`** so the model uses LaightDB MCP for persistent memory (search + store policy), and a **`beforeSubmitPrompt` hook** that runs **`search_context` over MCP** (streamable HTTP), using the **user’s prompt text** as the query. Together they complement the LaightDB MCP tools (`store_context`, `search_context`, etc.).

## Contents

| Path | Purpose |
|------|---------|
| `skills/laightdb-rolling-context/SKILL.md` | Agent skill: proactive search + store guidance |
| `hooks/laightdb-session-start.sh` | **`sessionStart`**: returns **`additional_context`** with LaightDB MCP memory policy (search at task start, store before ending substantive turns; no secrets) |
| `hooks/laightdb-before-submit-prompt-search.sh` | **`beforeSubmitPrompt`**: MCP `search_context` via `POST` to **`/mcp`**; on hits, returns `continue: true` and **`additional_context`** with a short hit list |

## Hook requirements

**`sessionStart`** (`laightdb-session-start.sh`):

- **`jq`** on `PATH` (prints empty `additional_context` if missing).

**`beforeSubmitPrompt`** (`laightdb-before-submit-prompt-search.sh`):

- **`curl`** and **`jq`** on `PATH` (JSON-RPC over streamable HTTP MCP).
- A **running LaightDB that serves streamable HTTP MCP** at **`LAIGHTDB_MCP_URL`** (default `http://127.0.0.1:8080/mcp`). This is the same endpoint Cursor uses for HTTP MCP; it is **not** `POST /v1/search` and not MCP stdio. Typically run the binary with **`LAIGHTDB_MCP_TRANSPORT=http`** (REST + `/mcp` on the same listener) or use Docker Compose per the project README.
- If auth is enabled, set **`LAIGHTDB_API_TOKEN`** (Bearer) in the environment Cursor uses when running hooks.
- **`LAIGHTDB_PROMPT_SEARCH=0`** disables the prompt hook.

### `beforeSubmitPrompt` output

Cursor’s documented schema for this hook lists **`continue`** and **`user_message`** (when blocked). This script also emits **`additional_context`** (same field name as **`sessionStart`**) so relevant LaightDB hits can be merged into the conversation when your Cursor build supports it. If your client ignores unknown fields, you still get **`continue: true`** and no prompt blocking.

## Manual installation

From the directory that contains your project (the workspace root):

1. **Skill** — copy the skill folder into the project:

   ```bash
   mkdir -p .cursor/skills
   cp -r /path/to/LaightDB/integrations/cursor/skills/laightdb-rolling-context .cursor/skills/
   ```

2. **Hook scripts** — copy and make executable:

   ```bash
   mkdir -p .cursor/hooks
   cp /path/to/LaightDB/integrations/cursor/hooks/laightdb-session-start.sh .cursor/hooks/
   cp /path/to/LaightDB/integrations/cursor/hooks/laightdb-before-submit-prompt-search.sh .cursor/hooks/
   chmod +x .cursor/hooks/laightdb-session-start.sh \
     .cursor/hooks/laightdb-before-submit-prompt-search.sh
   ```

3. **Register the hooks** — merge into `.cursor/hooks.json` (create if missing). Minimal example:

   ```json
   {
     "version": 1,
     "hooks": {
       "sessionStart": [
         {
           "command": ".cursor/hooks/laightdb-session-start.sh"
         }
       ],
       "beforeSubmitPrompt": [
         {
           "command": ".cursor/hooks/laightdb-before-submit-prompt-search.sh"
         }
       ]
     }
   }
   ```

   If you already have other hooks, merge into the existing `sessionStart` and `beforeSubmitPrompt` arrays instead of replacing the whole file.

   **Upgrading from older bundles:** If your `hooks.json` still lists **`stop`** → `laightdb-stop-reminder.sh` or **`beforeShellExecution`** → `laightdb-before-shell-search.sh`, you can remove those entries manually; they are no longer shipped.

4. Reload Cursor (or save `hooks.json`) so hooks are picked up.

## Automatic install

### Via LaightDB MCP (assistant or HTTP)

Use the MCP tool **`deploy_cursor_integration`** with **`project_root`** set to your workspace root (absolute path). In Cursor, ask the assistant to call it after the LaightDB MCP server is connected.

From a shell (LaightDB must be running with **`LAIGHTDB_MCP_TRANSPORT=http`** so **`/mcp`** is available):

```bash
cd /path/to/LaightDB
./scripts/mcp-deploy-cursor-integration.sh /absolute/path/to/your-project
```

Set **`LAIGHTDB_MCP_URL`** if your MCP base is not `http://127.0.0.1:8080/mcp` (for example the dev stack on port **9090**: `export LAIGHTDB_MCP_URL=http://127.0.0.1:9090/mcp`).

### Without MCP (same files on disk)

Equivalent to the MCP tool (writes the same assets from this repository):

```bash
go run ./integrations/cursor/cmd/deployhooks /absolute/path/to/your-project
```

See the main [README.md](../../README.md) for the full Cursor rolling-context section.

## Requirements

- LaightDB MCP configured in Cursor (stdio or HTTP) for normal assistant tools.
- For **`beforeSubmitPrompt`**: streamable MCP at **`LAIGHTDB_MCP_URL`**; optional Bearer token when not in open mode.
