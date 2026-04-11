package context

import (
	"regexp"
	"strings"
)

// Compact transforms human-readable content into a token-minimized form
// that LLMs can still parse. Techniques: strip filler, collapse whitespace,
// abbreviate common words, remove redundant punctuation/formatting.
func Compact(content string) string {
	if content == "" {
		return ""
	}
	s := content

	s = stripFillerPhrases(s)
	s = abbreviateCommon(s)
	s = collapseWhitespace(s)
	s = stripRedundantPunctuation(s)

	s = strings.TrimSpace(s)
	if s == "" {
		return content
	}
	return s
}

var fillerPhrases = []string{
	"please note that ",
	"it is important to note that ",
	"it should be noted that ",
	"it is worth mentioning that ",
	"as you can see, ",
	"as we can see, ",
	"in order to ",
	"for the purpose of ",
	"at the end of the day ",
	"at this point in time ",
	"due to the fact that ",
	"in the event that ",
	"on the other hand, ",
	"as a matter of fact, ",
	"in other words, ",
	"for what it's worth, ",
	"needless to say, ",
	"it goes without saying that ",
	"the fact of the matter is ",
	"basically, ",
	"essentially, ",
	"actually, ",
	"in terms of ",
}

func stripFillerPhrases(s string) string {
	lower := strings.ToLower(s)
	for _, f := range fillerPhrases {
		for {
			idx := strings.Index(strings.ToLower(lower), f)
			if idx < 0 {
				break
			}
			s = s[:idx] + s[idx+len(f):]
			lower = strings.ToLower(s)
		}
	}
	return s
}

// Ordered longest-first to avoid partial matches (e.g. "function" before "for").
var abbreviations = []struct{ full, short string }{
	{"information", "info"},
	{"configuration", "cfg"},
	{"implementation", "impl"},
	{"documentation", "docs"},
	{"application", "app"},
	{"environment", "env"},
	{"development", "dev"},
	{"production", "prod"},
	{"performance", "perf"},
	{"description", "desc"},
	{"repository", "repo"},
	{"specification", "spec"},
	{"authentication", "auth"},
	{"authorization", "authz"},
	{"organization", "org"},
	{"notification", "notif"},
	{"parameter", "param"},
	{"parameters", "params"},
	{"function", "fn"},
	{"directory", "dir"},
	{"database", "db"},
	{"password", "pwd"},
	{"message", "msg"},
	{"request", "req"},
	{"response", "resp"},
	{"maximum", "max"},
	{"minimum", "min"},
	{"because", "bc"},
	{"between", "btwn"},
	{"example", "eg"},
	{"without", "w/o"},
	{"through", "thru"},
	{"approximately", "~"},
	{"however", "but"},
	{"therefore", "so"},
	{"although", "tho"},
	{"which is", "="},
	{"number", "num"},
	{"string", "str"},
	{"return", "ret"},
	{"integer", "int"},
	{"boolean", "bool"},
	{"package", "pkg"},
	{"library", "lib"},
	{"version", "ver"},
	{"command", "cmd"},
	{"execute", "exec"},
	{"address", "addr"},
	{"context", "ctx"},
	{"memory", "mem"},
	{"buffer", "buf"},
	{"object", "obj"},
	{"result", "res"},
	{"length", "len"},
	{"previous", "prev"},
	{"current", "cur"},
	{"temporary", "tmp"},
	{"original", "orig"},
	{"reference", "ref"},
	{"alternative", "alt"},
	{"continue", "cont"},
	{"received", "recv"},
	{"generate", "gen"},
	{"calculate", "calc"},
	{"initialize", "init"},
	{"synchronize", "sync"},
}

var wordBoundaryReplacer = buildAbbreviationReplacer()

func buildAbbreviationReplacer() *strings.Replacer {
	pairs := make([]string, 0, len(abbreviations)*2)
	for _, a := range abbreviations {
		pairs = append(pairs, " "+a.full+" ", " "+a.short+" ")
	}
	return strings.NewReplacer(pairs...)
}

func abbreviateCommon(s string) string {
	// Pad so word-boundary replacement works at edges.
	s = " " + s + " "
	s = wordBoundaryReplacer.Replace(s)
	return s[1 : len(s)-1]
}

var multiSpace = regexp.MustCompile(`[ \t]+`)
var multiNewline = regexp.MustCompile(`\n{3,}`)

func collapseWhitespace(s string) string {
	s = multiSpace.ReplaceAllString(s, " ")
	s = multiNewline.ReplaceAllString(s, "\n\n")
	return s
}

// Strip decorative markdown/punctuation that doesn't help AI comprehension.
var decorativeLine = regexp.MustCompile(`(?m)^[-=*]{3,}\s*$`)

func stripRedundantPunctuation(s string) string {
	s = decorativeLine.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "...", "…")
	return s
}
