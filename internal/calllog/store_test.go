package calllog

import (
	"testing"
	"time"

	"github.com/gtrig/laightdb/internal/auth"
)

func TestListNewestFirst(t *testing.T) {
	t.Parallel()
	s := New(Options{MaxEntries: 10, MaxBodyRunes: 1024})
	s.RecordMCP(t.Context(), time.Now(), "tool_a", "{}", "{}", false, time.Millisecond)
	time.Sleep(2 * time.Millisecond)
	s.RecordMCP(t.Context(), time.Now(), "tool_b", "{}", "{}", false, time.Millisecond)

	list := s.List(10)
	if len(list) != 2 {
		t.Fatalf("len=%d", len(list))
	}
	if list[0].Tool != "tool_b" || list[1].Tool != "tool_a" {
		t.Fatalf("order: %#v", list)
	}
}

func TestRecordMCPWithUserContext(t *testing.T) {
	t.Parallel()
	s := New(DefaultOptions())
	u := &auth.User{ID: "uid1", Username: "bob", Role: auth.RoleAdmin}
	ctx := auth.WithUser(t.Context(), u)
	start := time.Now()
	s.RecordMCP(ctx, start, "get_stats", "{}", `{"entries":1}`, false, time.Millisecond)
	list := s.List(1)
	if len(list) != 1 {
		t.Fatal(len(list))
	}
	if list[0].UserID != "uid1" || list[0].Username != "bob" {
		t.Fatalf("%+v", list[0])
	}
	if list[0].Channel != "mcp" || list[0].Tool != "get_stats" {
		t.Fatalf("%+v", list[0])
	}
}
