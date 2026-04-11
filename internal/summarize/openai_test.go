package summarize

import "testing"

func TestOpenAIRequiresAPIKey(t *testing.T) {
	t.Parallel()
	cases := []struct {
		base string
		want bool
	}{
		{"https://api.openai.com/v1", true},
		{"https://API.OPENAI.COM/v1", true},
		{"http://127.0.0.1:1234/v1", false},
		{"http://localhost:1234/v1", false},
		{"http://192.168.1.10:1234/v1", false},
		{"not-a-url", true},
	}
	for _, tc := range cases {
		if got := openAIRequiresAPIKey(tc.base); got != tc.want {
			t.Errorf("openAIRequiresAPIKey(%q) = %v, want %v", tc.base, got, tc.want)
		}
	}
}
