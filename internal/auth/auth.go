package auth

import (
	"context"
	"errors"
	"time"
)

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleReadOnly Role = "readonly"
)

func (r Role) Valid() bool {
	return r == RoleAdmin || r == RoleReadOnly
}

// CanEscalate reports whether this role is allowed to create resources with target role.
func (r Role) CanEscalate(target Role) bool {
	if r == RoleAdmin {
		return true
	}
	return target == RoleReadOnly
}

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"password_hash"`
	Role         Role      `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type APIToken struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Name      string     `json:"name"`
	TokenHash string     `json:"token_hash"`
	Prefix    string     `json:"prefix"`
	Role      Role       `json:"role"`
	CreatedAt time.Time  `json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

func (t *APIToken) Active() bool {
	return t.RevokedAt == nil
}

type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (s *Session) Expired() bool {
	return time.Now().After(s.ExpiresAt)
}

var (
	ErrNotFound       = errors.New("not found")
	ErrDuplicate      = errors.New("duplicate")
	ErrInvalidRole    = errors.New("invalid role")
	ErrBadCredentials = errors.New("bad credentials")
	ErrTokenRevoked   = errors.New("token revoked")
)

type contextKey int

const (
	ctxKeyUser contextKey = iota
	ctxKeyRole
)

func WithUser(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, ctxKeyUser, u)
}

func UserFromContext(ctx context.Context) (*User, bool) {
	u, ok := ctx.Value(ctxKeyUser).(*User)
	return u, ok
}

func WithRole(ctx context.Context, r Role) context.Context {
	return context.WithValue(ctx, ctxKeyRole, r)
}

func RoleFromContext(ctx context.Context) (Role, bool) {
	r, ok := ctx.Value(ctxKeyRole).(Role)
	return r, ok
}
