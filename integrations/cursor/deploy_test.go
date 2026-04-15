package cursorintegration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestDeployWritesSkillAndHook(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	res, err := Deploy(root, DeployOptions{OverwriteSkill: true, MergeHooks: true})
	if err != nil {
		t.Fatal(err)
	}
	skill := filepath.Join(root, ".cursor/skills/laightdb-rolling-context/SKILL.md")
	hookSession := filepath.Join(root, ".cursor/hooks/laightdb-session-start.sh")
	hookPrompt := filepath.Join(root, ".cursor/hooks/laightdb-before-submit-prompt-search.sh")
	if _, err := os.Stat(skill); err != nil {
		t.Fatalf("skill: %v", err)
	}
	if _, err := os.Stat(hookSession); err != nil {
		t.Fatalf("sessionStart hook: %v", err)
	}
	if _, err := os.Stat(hookPrompt); err != nil {
		t.Fatalf("before-submit-prompt hook: %v", err)
	}
	if fi, err := os.Stat(hookSession); err != nil || fi.Mode()&0o111 == 0 {
		t.Errorf("sessionStart hook should be executable, mode=%v err=%v", fi.Mode(), err)
	}
	if fi, err := os.Stat(hookPrompt); err != nil || fi.Mode()&0o111 == 0 {
		t.Errorf("before-submit-prompt hook should be executable, mode=%v err=%v", fi.Mode(), err)
	}
	if !res.HooksMerged {
		t.Error("expected hooks merged")
	}
	raw, err := os.ReadFile(filepath.Join(root, ".cursor/hooks.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	hooks := doc["hooks"].(map[string]any)
	ss := hooks["sessionStart"].([]any)
	if len(ss) != 1 {
		t.Fatalf("sessionStart hooks: %v", ss)
	}
	ss0 := ss[0].(map[string]any)
	if ss0["command"] != hookSessionCommand {
		t.Fatalf("sessionStart command: %v", ss0["command"])
	}
	submit := hooks["beforeSubmitPrompt"].([]any)
	if len(submit) != 1 {
		t.Fatalf("beforeSubmitPrompt hooks: %v", submit)
	}
	sp0 := submit[0].(map[string]any)
	if sp0["command"] != hookPromptCommand {
		t.Fatalf("beforeSubmitPrompt command: %v", sp0["command"])
	}
}

func TestDeployMergeHooksIdempotent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if _, err := Deploy(root, DeployOptions{OverwriteSkill: true, MergeHooks: true}); err != nil {
		t.Fatal(err)
	}
	res, err := Deploy(root, DeployOptions{OverwriteSkill: true, MergeHooks: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.HooksMerged {
		t.Error("second deploy should not merge hooks again")
	}
	if res.HooksMergeNote == "" {
		t.Error("expected note on second merge")
	}
	if !strings.Contains(res.HooksMergeNote, "sessionStart hook already") ||
		!strings.Contains(res.HooksMergeNote, "beforeSubmitPrompt hook already") {
		t.Errorf("unexpected merge note: %q", res.HooksMergeNote)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".cursor/hooks.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	hooks := doc["hooks"].(map[string]any)
	ss := hooks["sessionStart"].([]any)
	if len(ss) != 1 {
		t.Fatalf("expected single sessionStart hook, got %d", len(ss))
	}
	submit := hooks["beforeSubmitPrompt"].([]any)
	if len(submit) != 1 {
		t.Fatalf("expected single beforeSubmitPrompt hook, got %d", len(submit))
	}
}

func TestDeploySkipExistingSkill(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if _, err := Deploy(root, DeployOptions{OverwriteSkill: true, MergeHooks: false}); err != nil {
		t.Fatal(err)
	}
	skill := filepath.Join(root, ".cursor/skills/laightdb-rolling-context/SKILL.md")
	if err := os.WriteFile(skill, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Deploy(root, DeployOptions{OverwriteSkill: false, MergeHooks: false})
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(skill)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "keep" {
		t.Fatalf("skill overwritten: %q", b)
	}
	if len(res.Skipped) == 0 {
		t.Error("expected skipped path")
	}
}

func TestMergePreservesOtherHooks(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	cursor := filepath.Join(root, ".cursor")
	if err := os.MkdirAll(cursor, 0o755); err != nil {
		t.Fatal(err)
	}
	initial := `{
  "version": 1,
  "hooks": {
    "beforeSubmitPrompt": [
      { "command": ".cursor/hooks/other.sh" }
    ]
  }
}
`
	if err := os.WriteFile(filepath.Join(cursor, "hooks.json"), []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Deploy(root, DeployOptions{OverwriteSkill: true, MergeHooks: true}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(cursor, "hooks.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	hooks := doc["hooks"].(map[string]any)
	submit := hooks["beforeSubmitPrompt"].([]any)
	if len(submit) != 2 {
		t.Fatalf("expected two beforeSubmitPrompt hooks, got %v", submit)
	}
	var cmds []string
	for _, h := range submit {
		m := h.(map[string]any)
		cmds = append(cmds, m["command"].(string))
	}
	if !slices.Contains(cmds, ".cursor/hooks/other.sh") || !slices.Contains(cmds, hookPromptCommand) {
		t.Fatalf("commands: %v", cmds)
	}
	ss := hooks["sessionStart"].([]any)
	if len(ss) != 1 {
		t.Fatalf("sessionStart: %v", ss)
	}
	if ss[0].(map[string]any)["command"] != hookSessionCommand {
		t.Fatalf("sessionStart command: %v", ss[0])
	}
}
