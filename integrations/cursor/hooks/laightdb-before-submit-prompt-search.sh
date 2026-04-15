#!/usr/bin/env sh
# Cursor beforeSubmitPrompt: call LaightDB MCP tool search_context using the
# streamable HTTP MCP endpoint (POST /mcp), not REST /v1/search.
#
# Uses the user's prompt text as the hybrid search query (better fit than shell
# commands). Returns continue:true plus additional_context when hits exist.
#
# Requires: curl, jq. Optional: LAIGHTDB_API_TOKEN (Bearer).
# Env: LAIGHTDB_MCP_URL (default http://127.0.0.1:8080/mcp),
#      LAIGHTDB_PROMPT_SEARCH=0 to disable.
set -e

die_continue() {
	printf '%s\n' '{"continue":true}'
	exit 0
}

command -v jq >/dev/null 2>&1 || die_continue
command -v curl >/dev/null 2>&1 || die_continue

input=$(cat || true)
prompt=$(printf '%s' "$input" | jq -r '.prompt // empty' 2>/dev/null) || prompt=""
if [ -z "$prompt" ]; then
	die_continue
fi

case "${LAIGHTDB_PROMPT_SEARCH:-1}" in 0|false|no|off|OFF) die_continue ;; esac

MCP_URL="${LAIGHTDB_MCP_URL:-http://127.0.0.1:8080/mcp}"
TOKEN="${LAIGHTDB_API_TOKEN:-}"

d="$(mktemp -d "${TMPDIR:-/tmp}/laightdb-hook.XXXXXX")" || die_continue
trap 'rm -rf "$d"' EXIT INT HUP

curl_common() {
	# shellcheck disable=SC2086
	if [ -n "$TOKEN" ]; then
		curl -sS --max-time 8 -H "Content-Type: application/json" \
			-H "Accept: application/json, text/event-stream" \
			-H "Authorization: Bearer ${TOKEN}" "$@"
	else
		curl -sS --max-time 8 -H "Content-Type: application/json" \
			-H "Accept: application/json, text/event-stream" "$@"
	fi
}

init_body='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"laightdb-prompt-hook","version":"0.0.1"}}}'

if ! curl_common -D "$d/h1" -o "$d/b1" -X POST "$MCP_URL" -d "$init_body"; then
	die_continue
fi

sess_line=$(grep -i '^mcp-session-id:' "$d/h1" | head -1) || true
sess=$(printf '%s' "$sess_line" | sed 's/^[^:]*:[[:space:]]*//' | tr -d '\r')
if [ -z "$sess" ]; then
	die_continue
fi

if grep -q '^data: ' "$d/b1" 2>/dev/null; then
	pv=$(sed -n 's/^data: //p' "$d/b1" | jq -sr 'map(select(.result.protocolVersion)) | .[-1].result.protocolVersion // empty')
else
	pv=$(jq -r '.result.protocolVersion // empty' "$d/b1" 2>/dev/null || true)
fi
if [ -z "$pv" ]; then
	pv="2025-11-25"
fi

notif_body='{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}'
if ! curl_common -D "$d/h2" -o "$d/b2" -X POST "$MCP_URL" \
	-H "Mcp-Session-Id: ${sess}" \
	-H "Mcp-Protocol-Version: ${pv}" \
	-d "$notif_body"; then
	die_continue
fi

call_body=$(jq -n --arg q "$prompt" \
	'{jsonrpc:"2.0",id:3,method:"tools/call",params:{name:"search_context",arguments:{query:$q,top_k:5,detail:"summary"}}}') || die_continue

if ! curl_common -D "$d/h3" -o "$d/b3" -X POST "$MCP_URL" \
	-H "Mcp-Session-Id: ${sess}" \
	-H "Mcp-Protocol-Version: ${pv}" \
	-d "$call_body"; then
	die_continue
fi

rpc_raw=$(if grep -q '^data: ' "$d/b3" 2>/dev/null; then
	sed -n 's/^data: //p' "$d/b3" | jq -s --argjson want 3 'map(select(.id==$want)) | .[0] // empty'
else
	jq --argjson want 3 'select(.id==$want)' "$d/b3" 2>/dev/null || echo ""
fi)

if [ -z "$rpc_raw" ] || [ "$rpc_raw" = "null" ]; then
	die_continue
fi

if [ "$(echo "$rpc_raw" | jq -r '.error != null')" = "true" ]; then
	die_continue
fi

if [ "$(echo "$rpc_raw" | jq -r '.result.isError // false')" = "true" ]; then
	die_continue
fi

inner=$(echo "$rpc_raw" | jq -r '.result.content[0].text // empty')
if [ -z "$inner" ]; then
	die_continue
fi

ctx=$(jq -rn --arg p "$prompt" --arg inner "$inner" '
  def preview: if ($p|length) > 500 then ($p[0:500] + "...") else $p end;
  def trunc: if length > 240 then .[0:240] + "..." else . end;
  ($inner | try fromjson catch null) as $doc |
  if $doc == null then
    ("LaightDB MCP search_context (unparsed). Snippet:\n" + (if ($inner|length) > 800 then $inner[0:800] + "..." else $inner end))
  elif (($doc.hits // []) | length) == 0 then
    ""
  else
    ("LaightDB MCP search (from your prompt before submit):\n"
    + "Prompt: " + preview + "\n"
    + "Related stored context:\n"
    + (($doc.hits // []) | .[0:5] | to_entries
        | map("\(.key + 1). id=\(.value.id) collection=\(.value.entry.collection // "") score=\(.value.score) — \((.value.entry.summary // "") | trunc)")
        | join("\n")))
  end
')

if [ -z "$ctx" ]; then
	die_continue
fi

(
	curl -sS --max-time 3 -o /dev/null -X DELETE \
		-H "Mcp-Session-Id: ${sess}" \
		${TOKEN:+-H "Authorization: Bearer ${TOKEN}"} \
		"$MCP_URL" || true
) >/dev/null 2>&1 || true

jq -n --arg c "$ctx" '{continue:true,additional_context:$c}'
