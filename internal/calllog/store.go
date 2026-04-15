package calllog

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gtrig/laightdb/internal/auth"
)

// Entry is one recorded MCP tool call (newest entries returned first from List).
type Entry struct {
	ID         string    `json:"id"`
	TS         time.Time `json:"ts"`
	DurationMS int64     `json:"duration_ms"`
	Channel    string    `json:"channel"` // always "mcp" for new records

	UserID   string `json:"user_id,omitempty"`
	Username string `json:"username,omitempty"`

	Method string `json:"method,omitempty"`
	Path   string `json:"path,omitempty"`
	Query  string `json:"query,omitempty"`

	Tool string `json:"tool,omitempty"`

	Status int  `json:"status,omitempty"`
	OK     bool `json:"ok,omitempty"`

	Request  string `json:"request,omitempty"`
	Response string `json:"response,omitempty"`
}

// Options configures the call log store.
type Options struct {
	MaxEntries   int
	MaxBodyRunes int
}

// DefaultOptions returns conservative defaults.
func DefaultOptions() Options {
	return Options{
		MaxEntries:   500,
		MaxBodyRunes: 32 * 1024,
	}
}

// Store is a bounded in-memory ring of call entries.
type Store struct {
	opt Options

	mu   sync.Mutex
	ring []Entry
}

// New creates a Store with the given options (zero Options uses DefaultOptions).
func New(opt Options) *Store {
	if opt.MaxEntries <= 0 {
		opt.MaxEntries = DefaultOptions().MaxEntries
	}
	if opt.MaxBodyRunes <= 0 {
		opt.MaxBodyRunes = DefaultOptions().MaxBodyRunes
	}
	return &Store{opt: opt}
}

func (s *Store) truncate(str string) string {
	runes := []rune(str)
	if len(runes) <= s.opt.MaxBodyRunes {
		return str
	}
	return string(runes[:s.opt.MaxBodyRunes]) + "…(truncated)"
}

func (s *Store) append(e Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ring = append(s.ring, e)
	if len(s.ring) > s.opt.MaxEntries {
		s.ring = s.ring[len(s.ring)-s.opt.MaxEntries:]
	}
}

func callerFromContext(ctx context.Context) (userID, username string) {
	u, ok := auth.UserFromContext(ctx)
	if !ok || u == nil {
		return "", ""
	}
	return u.ID, u.Username
}

// List returns up to limit entries, newest first.
func (s *Store) List(limit int) []Entry {
	if limit <= 0 {
		limit = 100
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.ring)
	if n == 0 {
		return nil
	}
	if limit > n {
		limit = n
	}
	out := make([]Entry, limit)
	for i := 0; i < limit; i++ {
		out[i] = s.ring[n-1-i]
	}
	return out
}

// RecordMCP records one MCP tool invocation.
func (s *Store) RecordMCP(ctx context.Context, start time.Time, toolName, inputJSON, responseText string, isError bool, d time.Duration) {
	uid, uname := callerFromContext(ctx)
	status := 200
	ok := !isError
	if isError {
		status = 500
	}
	e := Entry{
		ID:         uuid.New().String(),
		TS:         start.UTC(),
		DurationMS: d.Milliseconds(),
		Channel:    "mcp",
		UserID:     uid,
		Username:   uname,
		Tool:       toolName,
		Status:     status,
		OK:         ok,
		Request:    s.truncate(inputJSON),
		Response:   s.truncate(responseText),
	}
	s.append(e)
}
