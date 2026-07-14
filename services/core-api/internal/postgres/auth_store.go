package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/auth"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
)

const sessionLifetime = 12 * time.Hour

func (s *Store) CreateSession(ctx context.Context, email, password string) (auth.Session, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" {
		return auth.Session{}, platform.Invalid("invalid_credentials", "email or password is incorrect")
	}
	var session auth.Session
	var passwordHash string
	err := s.pool.QueryRow(ctx, `
		SELECT u.id,u.email,u.display_name,u.password_hash,t.id,t.name,m.role
		FROM users u
		JOIN memberships m ON m.user_id=u.id::text
		JOIN tenants t ON t.id=m.tenant_id
		WHERE lower(u.email)=lower($1) AND u.status='active' AND t.status='active'
		ORDER BY t.created_at
		LIMIT 1`, email).Scan(
		&session.UserID, &session.Email, &session.DisplayName, &passwordHash,
		&session.TenantID, &session.TenantName, &session.Role,
	)
	if err != nil || auth.VerifyPassword(passwordHash, password) != nil {
		return auth.Session{}, platform.Invalid("invalid_credentials", "email or password is incorrect")
	}
	token, err := auth.NewToken()
	if err != nil {
		return auth.Session{}, err
	}
	session.Token = token
	session.ExpiresAt = time.Now().UTC().Add(sessionLifetime)
	if err := s.pool.QueryRow(ctx, `
		INSERT INTO auth_sessions (tenant_id,user_id,token_hash,expires_at)
		VALUES ($1,$2,$3,$4)
		RETURNING id`, session.TenantID, session.UserID, auth.HashToken(token), session.ExpiresAt).Scan(&session.ID); err != nil {
		return auth.Session{}, mapError(err)
	}
	return session, nil
}

func (s *Store) ResolveSession(ctx context.Context, token string) (auth.Session, error) {
	if token == "" {
		return auth.Session{}, platform.Invalid("session_invalid", "session is missing or expired")
	}
	var session auth.Session
	err := s.pool.QueryRow(ctx, `
		UPDATE auth_sessions s
		SET last_seen_at=now()
		FROM users u,tenants t,memberships m
		WHERE s.token_hash=$1 AND s.revoked_at IS NULL AND s.expires_at>now()
		  AND u.id=s.user_id AND u.status='active'
		  AND t.id=s.tenant_id AND t.status='active'
		  AND m.tenant_id=s.tenant_id AND m.user_id=s.user_id::text
		RETURNING s.id,u.id,u.email,u.display_name,t.id,t.name,m.role,s.expires_at`, auth.HashToken(token)).Scan(
		&session.ID, &session.UserID, &session.Email, &session.DisplayName,
		&session.TenantID, &session.TenantName, &session.Role, &session.ExpiresAt,
	)
	if err != nil {
		return auth.Session{}, platform.Invalid("session_invalid", "session is missing or expired")
	}
	return session, nil
}

func (s *Store) RevokeSession(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	_, err := s.pool.Exec(ctx, `UPDATE auth_sessions SET revoked_at=now() WHERE token_hash=$1 AND revoked_at IS NULL`, auth.HashToken(token))
	return mapError(err)
}
