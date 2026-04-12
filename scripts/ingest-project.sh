#!/usr/bin/env bash
set -euo pipefail

API="${LAIGHTDB_URL:-http://localhost:8080}"
TOKEN="${LAIGHTDB_API_KEY:-ldb_15477d183fe7b851834e96c542e897228bf3352f}"
PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

stored=0
failed=0
# Arithmetic returning 0 is falsy in bash; use || true with set -e

store_file() {
    local filepath="$1"
    local relpath="${filepath#$PROJECT_ROOT/}"

    local ext="${filepath##*.}"
    local content_type="code"
    local pkg=""

    case "$relpath" in
        *.md)            content_type="doc" ;;
        *.yml|*.yaml)    content_type="doc" ;;
        *.toml)          content_type="doc" ;;
        *.mod)           content_type="doc" ;;
        Dockerfile*)     content_type="doc" ;;
        Makefile)        content_type="doc" ;;
        .gitignore)      content_type="doc" ;;
    esac

    local collection="laightdb"
    local dir
    dir="$(dirname "$relpath")"
    case "$dir" in
        internal/*)  collection="${dir#internal/}" ; collection="laightdb/${collection%%/*}" ;;
        cmd/*)       collection="laightdb/cmd" ;;
        ui/*)        collection="laightdb/ui" ;;
        scripts/*)   collection="laightdb/scripts" ;;
        .)           collection="laightdb" ;;
    esac

    if [[ "$ext" == "go" ]]; then
        pkg=$(head -1 "$filepath" | grep -oP 'package \K\w+' || true)
    fi

    local is_test="false"
    [[ "$relpath" == *_test.go ]] && is_test="true"

    local content
    content=$(cat "$filepath")

    local metadata
    metadata=$(jq -n \
        --arg path "$relpath" \
        --arg dir "$dir" \
        --arg ext "$ext" \
        --arg pkg "$pkg" \
        --arg is_test "$is_test" \
        '{path: $path, dir: $dir, ext: $ext, package: $pkg, is_test: $is_test}')

    local payload
    payload=$(jq -n \
        --arg collection "$collection" \
        --arg content "$content" \
        --arg content_type "$content_type" \
        --argjson metadata "$metadata" \
        '{collection: $collection, content: $content, content_type: $content_type, metadata: $metadata}')

    local resp
    resp=$(curl -s -w "\n%{http_code}" -X POST "$API/v1/contexts" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d "$payload" 2>/dev/null)

    local code
    code=$(echo "$resp" | tail -1)
    local body
    body=$(echo "$resp" | head -n -1)

    if [[ "$code" == "200" ]]; then
        local id
        id=$(echo "$body" | jq -r '.id // "?"')
        printf "  %-55s -> %s [%s]\n" "$relpath" "$id" "$collection"
        stored=$((stored + 1))
    else
        printf "  FAIL %-50s  HTTP %s: %s\n" "$relpath" "$code" "$body"
        failed=$((failed + 1))
    fi
}

echo "Scanning $PROJECT_ROOT ..."
echo "API: $API"
echo ""

while IFS= read -r filepath; do
    store_file "$filepath"
done < <(find "$PROJECT_ROOT" -type f \
    \( -name "*.go" -o -name "*.md" -o -name "*.yml" -o -name "*.yaml" \
       -o -name "*.toml" -o -name "*.mod" -o -name "Dockerfile*" \
       -o -name "Makefile" -o -name ".gitignore" \) \
    ! -path "*/.git/*" ! -path "*/vendor/*" ! -path "*/node_modules/*" \
    ! -path "*/.cursor/*" ! -path "*/scripts/ingest*" \
    | sort)

echo ""
echo "Done: $stored stored, $failed failed"
