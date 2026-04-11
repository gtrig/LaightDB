package embedding

import (
	"testing"
)

func TestEngineRequiresModel(t *testing.T) {
	t.Parallel()
	_, err := NewEngine()
	if err != nil {
		t.Skip("gobed model not available: ", err)
	}
}
