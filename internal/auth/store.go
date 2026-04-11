package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// FileAuthStore persists users, tokens, and sessions as JSON files in a directory.
type FileAuthStore struct {
	dir        string
	sessionTTL time.Duration

	mu       sync.RWMutex
	users    []*User
	tokens   []*APIToken
	sessions []*Session

	usersByID       map[string]*User
	usersByUsername  map[string]*User
	tokensByHash    map[string]*APIToken
	tokensByID      map[string]*APIToken
	sessionsByID    map[string]*Session
}

func NewFileAuthStore(dir string, sessionTTL time.Duration) (*FileAuthStore, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create auth dir: %w", err)
	}
	s := &FileAuthStore{
		dir:        dir,
		sessionTTL: sessionTTL,
	}
	if err := s.load(); err != nil {
		return nil, fmt.Errorf("load auth data: %w", err)
	}
	return s, nil
}

// UserCount returns the number of registered users. Zero means open mode.
func (s *FileAuthStore) UserCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.users)
}

// --- User operations ---

func (s *FileAuthStore) CreateUser(_ context.Context, username, password string, role Role) (*User, error) {
	if !role.Valid() {
		return nil, ErrInvalidRole
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	now := time.Now().UTC()
	u := &User{
		ID:           uuid.New().String(),
		Username:     username,
		PasswordHash: string(hash),
		Role:         role,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.usersByUsername[username]; exists {
		return nil, fmt.Errorf("username %q: %w", username, ErrDuplicate)
	}
	s.users = append(s.users, u)
	s.usersByID[u.ID] = u
	s.usersByUsername[u.Username] = u
	if err := s.saveUsersLocked(); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *FileAuthStore) GetUser(_ context.Context, id string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.usersByID[id]
	if !ok {
		return nil, ErrNotFound
	}
	return u, nil
}

func (s *FileAuthStore) ListUsers(_ context.Context) ([]*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*User, len(s.users))
	copy(out, s.users)
	return out, nil
}

func (s *FileAuthStore) UpdatePassword(_ context.Context, id, newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.usersByID[id]
	if !ok {
		return ErrNotFound
	}
	u.PasswordHash = string(hash)
	u.UpdatedAt = time.Now().UTC()
	return s.saveUsersLocked()
}

func (s *FileAuthStore) UpdateRole(_ context.Context, id string, role Role) error {
	if !role.Valid() {
		return ErrInvalidRole
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.usersByID[id]
	if !ok {
		return ErrNotFound
	}
	u.Role = role
	u.UpdatedAt = time.Now().UTC()
	return s.saveUsersLocked()
}

func (s *FileAuthStore) DeleteUser(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.usersByID[id]
	if !ok {
		return ErrNotFound
	}

	delete(s.usersByID, u.ID)
	delete(s.usersByUsername, u.Username)
	s.users = removeFromSlice(s.users, func(x *User) bool { return x.ID == id })

	// Cascade: remove user's tokens
	s.tokens = removeFromSlice(s.tokens, func(t *APIToken) bool {
		if t.UserID == id {
			delete(s.tokensByHash, t.TokenHash)
			delete(s.tokensByID, t.ID)
			return true
		}
		return false
	})

	// Cascade: remove user's sessions
	s.sessions = removeFromSlice(s.sessions, func(sess *Session) bool {
		if sess.UserID == id {
			delete(s.sessionsByID, sess.ID)
			return true
		}
		return false
	})

	if err := s.saveUsersLocked(); err != nil {
		return err
	}
	if err := s.saveTokensLocked(); err != nil {
		return err
	}
	return s.saveSessionsLocked()
}

// --- Authentication ---

func (s *FileAuthStore) AuthenticatePassword(_ context.Context, username, password string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.usersByUsername[username]
	if !ok {
		return nil, ErrBadCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, ErrBadCredentials
	}
	return u, nil
}

func (s *FileAuthStore) AuthenticateToken(_ context.Context, bearer string) (*User, *APIToken, error) {
	h := hashToken(bearer)

	s.mu.RLock()
	defer s.mu.RUnlock()

	tok, ok := s.tokensByHash[h]
	if !ok {
		return nil, nil, ErrBadCredentials
	}
	if !tok.Active() {
		return nil, nil, ErrTokenRevoked
	}
	u, ok := s.usersByID[tok.UserID]
	if !ok {
		return nil, nil, ErrBadCredentials
	}
	return u, tok, nil
}

// --- Token operations ---

// CreateToken generates a new API token and returns the plaintext (shown once).
func (s *FileAuthStore) CreateToken(_ context.Context, userID, name string, role Role) (plaintext string, token *APIToken, err error) {
	if !role.Valid() {
		return "", nil, ErrInvalidRole
	}

	raw, err := generateTokenString()
	if err != nil {
		return "", nil, fmt.Errorf("generate token: %w", err)
	}

	now := time.Now().UTC()
	tok := &APIToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		Name:      name,
		TokenHash: hashToken(raw),
		Prefix:    raw[:8],
		Role:      role,
		CreatedAt: now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.usersByID[userID]; !ok {
		return "", nil, ErrNotFound
	}
	s.tokens = append(s.tokens, tok)
	s.tokensByHash[tok.TokenHash] = tok
	s.tokensByID[tok.ID] = tok
	if err := s.saveTokensLocked(); err != nil {
		return "", nil, err
	}
	return raw, tok, nil
}

func (s *FileAuthStore) ListTokens(_ context.Context, userID string) ([]*APIToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []*APIToken
	for _, t := range s.tokens {
		if userID == "" || t.UserID == userID {
			out = append(out, t)
		}
	}
	return out, nil
}

func (s *FileAuthStore) RevokeToken(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tok, ok := s.tokensByID[id]
	if !ok {
		return ErrNotFound
	}
	now := time.Now().UTC()
	tok.RevokedAt = &now
	return s.saveTokensLocked()
}

// --- Session operations ---

func (s *FileAuthStore) CreateSession(_ context.Context, userID string) (*Session, error) {
	sess := &Session{
		ID:        uuid.New().String(),
		UserID:    userID,
		ExpiresAt: time.Now().UTC().Add(s.sessionTTL),
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions = append(s.sessions, sess)
	s.sessionsByID[sess.ID] = sess
	if err := s.saveSessionsLocked(); err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *FileAuthStore) GetSession(_ context.Context, id string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, ok := s.sessionsByID[id]
	if !ok {
		return nil, ErrNotFound
	}
	if sess.Expired() {
		return nil, ErrNotFound
	}
	return sess, nil
}

func (s *FileAuthStore) DeleteSession(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessionsByID[id]; !ok {
		return ErrNotFound
	}
	delete(s.sessionsByID, id)
	s.sessions = removeFromSlice(s.sessions, func(sess *Session) bool { return sess.ID == id })
	return s.saveSessionsLocked()
}

func (s *FileAuthStore) PruneSessions(_ context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	before := len(s.sessions)
	s.sessions = removeFromSlice(s.sessions, func(sess *Session) bool {
		if sess.Expired() {
			delete(s.sessionsByID, sess.ID)
			return true
		}
		return false
	})
	pruned := before - len(s.sessions)
	if pruned > 0 {
		if err := s.saveSessionsLocked(); err != nil {
			return 0, err
		}
	}
	return pruned, nil
}

// --- Persistence ---

func (s *FileAuthStore) load() error {
	s.usersByID = make(map[string]*User)
	s.usersByUsername = make(map[string]*User)
	s.tokensByHash = make(map[string]*APIToken)
	s.tokensByID = make(map[string]*APIToken)
	s.sessionsByID = make(map[string]*Session)

	if err := loadJSON(filepath.Join(s.dir, "users.json"), &s.users); err != nil {
		return err
	}
	for _, u := range s.users {
		s.usersByID[u.ID] = u
		s.usersByUsername[u.Username] = u
	}

	if err := loadJSON(filepath.Join(s.dir, "tokens.json"), &s.tokens); err != nil {
		return err
	}
	for _, t := range s.tokens {
		s.tokensByHash[t.TokenHash] = t
		s.tokensByID[t.ID] = t
	}

	if err := loadJSON(filepath.Join(s.dir, "sessions.json"), &s.sessions); err != nil {
		return err
	}
	// Prune expired sessions on load
	s.sessions = removeFromSlice(s.sessions, func(sess *Session) bool {
		return sess.Expired()
	})
	for _, sess := range s.sessions {
		s.sessionsByID[sess.ID] = sess
	}

	return nil
}

func (s *FileAuthStore) saveUsersLocked() error {
	return saveJSON(filepath.Join(s.dir, "users.json"), s.users)
}

func (s *FileAuthStore) saveTokensLocked() error {
	return saveJSON(filepath.Join(s.dir, "tokens.json"), s.tokens)
}

func (s *FileAuthStore) saveSessionsLocked() error {
	return saveJSON(filepath.Join(s.dir, "sessions.json"), s.sessions)
}

// --- Helpers ---

func loadJSON[T any](path string, out *[]T) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		*out = nil
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if len(data) == 0 {
		*out = nil
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func saveJSON[T any](path string, data []T) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func hashToken(plaintext string) string {
	h := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(h[:])
}

func generateTokenString() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "ldb_" + hex.EncodeToString(b), nil
}

func removeFromSlice[T any](s []T, pred func(T) bool) []T {
	n := 0
	for _, v := range s {
		if !pred(v) {
			s[n] = v
			n++
		}
	}
	return s[:n]
}
