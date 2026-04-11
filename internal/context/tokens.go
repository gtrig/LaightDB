package context

// EstimateTokens approximates English token count (chars/4 heuristic).
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}
	n := len(text) / 4
	if n < 1 {
		return 1
	}
	return n
}
