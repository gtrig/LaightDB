package cursorintegration

import "embed"

// Source files for manual copy or MCP deploy (single source of truth).
//
//go:embed README.md skills/laightdb-rolling-context/SKILL.md hooks/laightdb-session-start.sh hooks/laightdb-before-submit-prompt-search.sh
var sourceFS embed.FS
