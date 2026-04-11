package context

import "testing"

func TestCompact_Empty(t *testing.T) {
	t.Parallel()
	if got := Compact(""); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestCompact_FillerRemoval(t *testing.T) {
	t.Parallel()
	input := "Please note that the server is running. It is important to note that the port is 8080."
	got := Compact(input)
	if got == input {
		t.Fatal("expected filler to be removed")
	}
	if len(got) >= len(input) {
		t.Fatalf("compact should be shorter: %q (len %d) vs original (len %d)", got, len(got), len(input))
	}
	for _, must := range []string{"server", "running", "port", "8080"} {
		if !contains(got, must) {
			t.Fatalf("compact lost key word %q: %q", must, got)
		}
	}
}

func TestCompact_Abbreviations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		wantSub  string
		wantGone string
	}{
		{"the database is configured", "db", "database"},
		{"check the configuration file", "cfg", "configuration"},
		{"the application runs in production", "app", "application"},
	}
	for _, tt := range tests {
		got := Compact(tt.input)
		if !contains(got, tt.wantSub) {
			t.Errorf("Compact(%q) = %q, missing %q", tt.input, got, tt.wantSub)
		}
	}
}

func TestCompact_WhitespaceCollapse(t *testing.T) {
	t.Parallel()
	input := "hello    world\t\ttab\n\n\n\n\nmulti"
	got := Compact(input)
	if contains(got, "    ") || contains(got, "\t\t") {
		t.Fatalf("whitespace not collapsed: %q", got)
	}
}

func TestCompact_DecorativeLines(t *testing.T) {
	t.Parallel()
	input := "header\n---\nbody\n===\nfooter"
	got := Compact(input)
	if contains(got, "---") || contains(got, "===") {
		t.Fatalf("decorative lines not stripped: %q", got)
	}
	if !contains(got, "header") || !contains(got, "body") || !contains(got, "footer") {
		t.Fatalf("lost real content: %q", got)
	}
}

func TestCompact_TokenSavings(t *testing.T) {
	t.Parallel()
	input := "Please note that the application configuration documentation " +
		"provides information about the database environment. " +
		"It is important to note that the implementation requires authentication. " +
		"The function parameters describe the request and response format."
	compact := Compact(input)
	origTokens := EstimateTokens(input)
	compactTokens := EstimateTokens(compact)
	if compactTokens >= origTokens {
		t.Fatalf("expected token savings: orig=%d compact=%d text=%q", origTokens, compactTokens, compact)
	}
	savings := origTokens - compactTokens
	t.Logf("original=%d compact=%d saved=%d (%.0f%%)", origTokens, compactTokens, savings,
		float64(savings)/float64(origTokens)*100)
}

func TestCompact_PreservesCodeBlocks(t *testing.T) {
	t.Parallel()
	input := "```go\nfunc main() {\n    fmt.Println(\"hello\")\n}\n```"
	got := Compact(input)
	if !contains(got, "func main()") || !contains(got, "Println") {
		t.Fatalf("code block content lost: %q", got)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexStr(s, sub) >= 0)
}

func indexStr(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
