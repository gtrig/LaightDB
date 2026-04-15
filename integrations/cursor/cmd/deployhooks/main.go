// Command deployhooks installs the bundled Cursor skill and hooks (same as MCP tool deploy_cursor_integration).
//
//	go run ./integrations/cursor/cmd/deployhooks /path/to/project
package main

import (
	"encoding/json"
	"log"
	"os"

	ci "github.com/gtrig/laightdb/integrations/cursor"
)

func main() {
	log.SetFlags(0)
	if len(os.Args) != 2 {
		log.Fatal("usage: deployhooks <project_root>")
	}
	res, err := ci.Deploy(os.Args[1], ci.DeployOptions{OverwriteSkill: true, MergeHooks: true})
	if err != nil {
		log.Fatal(err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(res); err != nil {
		log.Fatal(err)
	}
}
