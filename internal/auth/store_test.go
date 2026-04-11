package auth

import (
	"testing"
	"time"
)

func newTestStore(t *testing.T) *FileAuthStore {
	t.Helper()
	dir := t.TempDir()
	s, err := NewFileAuthStore(dir, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewFileAuthStore: %v", err)
	}
	return s
}

func TestUserCRUD(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()

	if s.UserCount() != 0 {
		t.Fatal("expected 0 users")
	}

	u, err := s.CreateUser(ctx, "alice", "secret123", RoleAdmin)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if u.Username != "alice" || u.Role != RoleAdmin {
		t.Fatalf("unexpected user: %+v", u)
	}
	if s.UserCount() != 1 {
		t.Fatal("expected 1 user")
	}

	// Duplicate username
	_, err = s.CreateUser(ctx, "alice", "other", RoleReadOnly)
	if err == nil {
		t.Fatal("expected duplicate error")
	}

	// Get by ID
	got, err := s.GetUser(ctx, u.ID)
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if got.Username != "alice" {
		t.Fatal("wrong user")
	}

	// List
	users, err := s.ListUsers(ctx)
	if err != nil || len(users) != 1 {
		t.Fatal("ListUsers failed")
	}

	// Update password
	if err := s.UpdatePassword(ctx, u.ID, "newpass"); err != nil {
		t.Fatalf("UpdatePassword: %v", err)
	}

	// Update role
	if err := s.UpdateRole(ctx, u.ID, RoleReadOnly); err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}
	got, _ = s.GetUser(ctx, u.ID)
	if got.Role != RoleReadOnly {
		t.Fatal("role not updated")
	}

	// Delete
	if err := s.DeleteUser(ctx, u.ID); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	if s.UserCount() != 0 {
		t.Fatal("expected 0 users after delete")
	}
}

func TestInvalidRole(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.CreateUser(t.Context(), "bob", "pass", Role("superuser"))
	if err == nil {
		t.Fatal("expected invalid role error")
	}
}

func TestAuthenticatePassword(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()

	_, err := s.CreateUser(ctx, "alice", "secret", RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}

	u, err := s.AuthenticatePassword(ctx, "alice", "secret")
	if err != nil {
		t.Fatalf("AuthenticatePassword: %v", err)
	}
	if u.Username != "alice" {
		t.Fatal("wrong user")
	}

	// Wrong password
	_, err = s.AuthenticatePassword(ctx, "alice", "wrong")
	if err != ErrBadCredentials {
		t.Fatalf("expected ErrBadCredentials, got %v", err)
	}

	// Unknown user
	_, err = s.AuthenticatePassword(ctx, "nobody", "x")
	if err != ErrBadCredentials {
		t.Fatalf("expected ErrBadCredentials, got %v", err)
	}
}

func TestTokenCRUD(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()

	u, _ := s.CreateUser(ctx, "alice", "secret", RoleAdmin)

	plaintext, tok, err := s.CreateToken(ctx, u.ID, "ci-key", RoleReadOnly)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}
	if len(plaintext) < 10 {
		t.Fatal("plaintext too short")
	}
	if tok.Prefix != plaintext[:8] {
		t.Fatal("prefix mismatch")
	}
	if !tok.Active() {
		t.Fatal("new token should be active")
	}

	// Authenticate with token
	gotUser, gotTok, err := s.AuthenticateToken(ctx, plaintext)
	if err != nil {
		t.Fatalf("AuthenticateToken: %v", err)
	}
	if gotUser.ID != u.ID || gotTok.ID != tok.ID {
		t.Fatal("wrong auth result")
	}

	// List tokens
	tokens, err := s.ListTokens(ctx, u.ID)
	if err != nil || len(tokens) != 1 {
		t.Fatal("ListTokens failed")
	}

	// Revoke
	if err := s.RevokeToken(ctx, tok.ID); err != nil {
		t.Fatalf("RevokeToken: %v", err)
	}

	// Authenticate revoked token
	_, _, err = s.AuthenticateToken(ctx, plaintext)
	if err != ErrTokenRevoked {
		t.Fatalf("expected ErrTokenRevoked, got %v", err)
	}
}

func TestSessionLifecycle(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()

	u, _ := s.CreateUser(ctx, "alice", "secret", RoleAdmin)

	sess, err := s.CreateSession(ctx, u.ID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	got, err := s.GetSession(ctx, sess.ID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.UserID != u.ID {
		t.Fatal("wrong session user")
	}

	// Delete session
	if err := s.DeleteSession(ctx, sess.ID); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	_, err = s.GetSession(ctx, sess.ID)
	if err != ErrNotFound {
		t.Fatal("expected ErrNotFound after delete")
	}
}

func TestPersistenceAcrossReload(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ctx := t.Context()

	s1, err := NewFileAuthStore(dir, 1*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s1.CreateUser(ctx, "alice", "pass", RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}

	// Reload from same directory
	s2, err := NewFileAuthStore(dir, 1*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if s2.UserCount() != 1 {
		t.Fatal("expected 1 user after reload")
	}
	_, err = s2.AuthenticatePassword(ctx, "alice", "pass")
	if err != nil {
		t.Fatal("password should still work after reload")
	}
}

func TestDeleteUserCascade(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()

	u, _ := s.CreateUser(ctx, "alice", "pass", RoleAdmin)
	_, _, _ = s.CreateToken(ctx, u.ID, "tok1", RoleAdmin)
	_, _ = s.CreateSession(ctx, u.ID)

	if err := s.DeleteUser(ctx, u.ID); err != nil {
		t.Fatal(err)
	}

	tokens, _ := s.ListTokens(ctx, u.ID)
	if len(tokens) != 0 {
		t.Fatal("tokens should be cascaded")
	}
}

func TestSessionExpiry(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ctx := t.Context()

	// Very short TTL
	s, err := NewFileAuthStore(dir, 1*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := s.CreateUser(ctx, "alice", "pass", RoleAdmin)
	sess, _ := s.CreateSession(ctx, u.ID)

	time.Sleep(5 * time.Millisecond)

	_, err = s.GetSession(ctx, sess.ID)
	if err != ErrNotFound {
		t.Fatal("expected expired session to be not found")
	}

	pruned, err := s.PruneSessions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if pruned != 1 {
		t.Fatalf("expected 1 pruned, got %d", pruned)
	}
}
