package cursorintegration

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	skillRelPath = "skills/laightdb-rolling-context/SKILL.md"
	hookSessionRel = "hooks/laightdb-session-start.sh"
	hookPromptRel  = "hooks/laightdb-before-submit-prompt-search.sh"

	hookSessionCommand = ".cursor/hooks/laightdb-session-start.sh"
	hookPromptCommand  = ".cursor/hooks/laightdb-before-submit-prompt-search.sh"
)

// DeployOptions controls how Deploy writes into a project tree.
type DeployOptions struct {
	OverwriteSkill bool // If false, skip SKILL.md when it already exists.
	MergeHooks     bool // If true, merge LaightDB hooks into .cursor/hooks.json; if false, skip hooks.json.
}

// DeployResult lists written paths and any skipped steps.
type DeployResult struct {
	Written        []string `json:"written"`
	Skipped        []string `json:"skipped,omitempty"`
	HooksMerged    bool     `json:"hooks_merged"`
	HooksMergeNote string   `json:"hooks_merge_note,omitempty"`
}

// Deploy writes the bundled skill and hook under projectRoot/.cursor and optionally merges hooks.json.
func Deploy(projectRoot string, opts DeployOptions) (DeployResult, error) {
	var out DeployResult
	if strings.TrimSpace(projectRoot) == "" {
		return out, errors.New("project_root is required")
	}
	root, err := filepath.Abs(projectRoot)
	if err != nil {
		return out, fmt.Errorf("project_root: %w", err)
	}
	st, err := os.Stat(root)
	if err != nil {
		return out, fmt.Errorf("project_root: %w", err)
	}
	if !st.IsDir() {
		return out, fmt.Errorf("project_root is not a directory: %s", root)
	}

	cursorDir := filepath.Join(root, ".cursor")
	skillsDir := filepath.Join(cursorDir, "skills", "laightdb-rolling-context")
	hooksDir := filepath.Join(cursorDir, "hooks")
	skillDest := filepath.Join(skillsDir, "SKILL.md")
	hookSessionDest := filepath.Join(hooksDir, "laightdb-session-start.sh")
	hookPromptDest := filepath.Join(hooksDir, "laightdb-before-submit-prompt-search.sh")

	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		return out, fmt.Errorf("mkdir skills: %w", err)
	}
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return out, fmt.Errorf("mkdir hooks: %w", err)
	}

	// Skill
	if _, err := os.Stat(skillDest); err == nil && !opts.OverwriteSkill {
		out.Skipped = append(out.Skipped, skillDest+" (exists; set overwrite_skill to replace)")
	} else {
		if err := copyFromEmbed(skillRelPath, skillDest, 0o644); err != nil {
			return out, err
		}
		out.Written = append(out.Written, skillDest)
	}

	// Hook scripts (always overwrite so fixes apply)
	if err := copyFromEmbed(hookSessionRel, hookSessionDest, 0o755); err != nil {
		return out, err
	}
	out.Written = append(out.Written, hookSessionDest)
	if err := copyFromEmbed(hookPromptRel, hookPromptDest, 0o755); err != nil {
		return out, err
	}
	out.Written = append(out.Written, hookPromptDest)

	if opts.MergeHooks {
		merged, note, err := mergeLaightHooks(filepath.Join(cursorDir, "hooks.json"))
		if err != nil {
			out.HooksMergeNote = "hooks.json not updated: " + err.Error()
		} else {
			out.HooksMerged = merged
			out.HooksMergeNote = note
		}
	} else {
		out.Skipped = append(out.Skipped, ".cursor/hooks.json (merge_hooks=false)")
	}

	return out, nil
}

func copyFromEmbed(rel, dest string, mode fs.FileMode) error {
	b, err := fs.ReadFile(sourceFS, rel)
	if err != nil {
		return fmt.Errorf("read embedded %s: %w", rel, err)
	}
	if err := os.WriteFile(dest, b, mode); err != nil {
		return fmt.Errorf("write %s: %w", dest, err)
	}
	return nil
}

// laightHookInstalls lists Cursor hook events and project-relative commands to register.
var laightHookInstalls = []struct {
	Event   string
	Command string
}{
	{"sessionStart", hookSessionCommand},
	{"beforeSubmitPrompt", hookPromptCommand},
}

// mergeLaightHooks registers bundled LaightDB hooks in hooks.json (idempotent).
func mergeLaightHooks(hooksPath string) (merged bool, note string, err error) {
	raw, err := os.ReadFile(hooksPath)
	var doc map[string]interface{}
	if err != nil {
		if !os.IsNotExist(err) {
			return false, "", err
		}
		doc = map[string]interface{}{
			"version": 1,
			"hooks":   map[string]interface{}{},
		}
	} else {
		if err := json.Unmarshal(raw, &doc); err != nil {
			return false, "", fmt.Errorf("parse hooks.json: %w", err)
		}
		if doc == nil {
			doc = map[string]interface{}{}
		}
		if _, ok := doc["version"]; !ok {
			doc["version"] = 1
		}
	}

	hooks, ok := doc["hooks"].(map[string]interface{})
	if !ok || hooks == nil {
		hooks = map[string]interface{}{}
		doc["hooks"] = hooks
	}

	var notes []string
	anyMerged := false
	for _, hi := range laightHookInstalls {
		var list []interface{}
		if s, ok := hooks[hi.Event]; ok {
			if arr, ok := s.([]interface{}); ok {
				list = arr
			}
		}
		found := false
		for _, item := range list {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			cmd, _ := m["command"].(string)
			if cmd == hi.Command {
				found = true
				break
			}
		}
		if found {
			notes = append(notes, hi.Event+" hook already references "+hi.Command)
			continue
		}
		hooks[hi.Event] = append(list, map[string]interface{}{"command": hi.Command})
		anyMerged = true
	}

	if !anyMerged {
		return false, strings.Join(notes, "; "), nil
	}

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return false, "", err
	}
	out = append(out, '\n')
	if err := os.WriteFile(hooksPath, out, 0o644); err != nil {
		return false, "", err
	}
	return true, "", nil
}
