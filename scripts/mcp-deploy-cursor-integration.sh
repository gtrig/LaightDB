#!/usr/bin/env sh
# Call LaightDB MCP tool deploy_cursor_integration over streamable HTTP (same wire as hooks use).
#
# Usage:
#   LAIGHTDB_MCP_URL=http://127.0.0.1:8080/mcp \
#   LAIGHTDB_API_TOKEN=... \
#   ./scripts/mcp-deploy-cursor-integration.sh /absolute/path/to/project
#
# Requires: curl, jq
set -e

die() {
	printf '%s\n' "$1" >&2
	exit 1
}

[ "$#" -eq 1 ] || die "usage: $0 <project_root>"

command -v curl >/dev/null 2>&1 || die "curl is required"
command -v jq >/dev/null 2>&1 || die "jq is required"

ROOT=$1
MCP_URL="${LAIGHTDB_MCP_URL:-http://127.0.0.1:8080/mcp}"
TOKEN="${LAIGHTDB_API_TOKEN:-}"

d="$(mktemp -d "${TMPDIR:-/tmp}/mcp-deploy.XXXXXX")" || exit 1
trap 'rm -rf "$d"' EXIT INT HUP

curl_mcp() {
	if [ -n "$TOKEN" ]; then
		curl -sS --max-time 15 -H "Content-Type: application/json" \
			-H "Accept: application/json, text/event-stream" \
			-H "Authorization: Bearer ${TOKEN}" "$@"
	else
		curl -sS --max-time 15 -H "Content-Type: application/json" \
			-H "Accept: application/json, text/event-stream" "$@"
	fi
}

init_body='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"mcp-deploy-cursor","version":"0.0.1"}}}'
curl_mcp -D "$d/h1" -o "$d/b1" -X POST "$MCP_URL" -d "$init_body" || die "initialize request failed"

sess_line=$(grep -i '^mcp-session-id:' "$d/h1" | head -1) || true
sess=$(printf '%s' "$sess_line" | sed 's/^[^:]*:[[:space:]]*//' | tr -d '\r')
[ -n "$sess" ] || die "no Mcp-Session-Id in response (is LaightDB running with LAIGHTDB_MCP_TRANSPORT=http?)"

if grep -q '^data: ' "$d/b1" 2>/dev/null; then
	pv=$(sed -n 's/^data: //p' "$d/b1" | jq -sr 'map(select(.result.protocolVersion)) | .[-1].result.protocolVersion // empty')
else
	pv=$(jq -r '.result.protocolVersion // empty' "$d/b1" 2>/dev/null || true)
fi
[ -n "$pv" ] || pv="2025-11-25"

notif_body='{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}'
curl_mcp -D "$d/h2" -o "$d/b2" -X POST "$MCP_URL" \
	-H "Mcp-Session-Id: ${sess}" \
	-H "Mcp-Protocol-Version: ${pv}" \
	-d "$notif_body" || die "initialized notification failed"

call_body=$(jq -n --arg root "$ROOT" \
	'{jsonrpc:"2.0",id:3,method:"tools/call",params:{name:"deploy_cursor_integration",arguments:{project_root:$root,overwrite_skill:true,merge_hooks:true}}}') || exit 1

curl_mcp -D "$d/h3" -o "$d/b3" -X POST "$MCP_URL" \
	-H "Mcp-Session-Id: ${sess}" \
	-H "Mcp-Protocol-Version: ${pv}" \
	-d "$call_body" || die "deploy_cursor_integration failed"

rpc_raw=$(if grep -q '^data: ' "$d/b3" 2>/dev/null; then
	sed -n 's/^data: //p' "$d/b3" | jq -s --argjson want 3 'map(select(.id==$want)) | .[0] // empty'
else
	jq --argjson want 3 'select(.id==$want)' "$d/b3" 2>/dev/null || echo ""
fi)

[ -n "$rpc_raw" ] && [ "$rpc_raw" != "null" ] || die "empty tools/call response"

if [ "$(echo "$rpc_raw" | jq -r '.error != null')" = "true" ]; then
	echo "$rpc_raw" | jq . >&2
	exit 1
fi

if [ "$(echo "$rpc_raw" | jq -r '.result.isError // false')" = "true" ]; then
	echo "$rpc_raw" | jq -r '.result.content[0].text // .' >&2
	exit 1
fi

text=$(echo "$rpc_raw" | jq -r '.result.content[0].text // empty')
printf '%s\n' "$text" | jq .

curl -sS --max-time 5 -o /dev/null -X DELETE \
	-H "Mcp-Session-Id: ${sess}" \
	${TOKEN:+-H "Authorization: Bearer ${TOKEN}"} \
	"$MCP_URL" 2>/dev/null || true
