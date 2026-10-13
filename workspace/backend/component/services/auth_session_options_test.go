package services

import (
	"context"
	"errors"
	"main/utils"
	"testing"
	"time"
)

func TestRememberMe_PasswordLogin(t *testing.T) {
	fixed := time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)
	cases := []struct {
		name       string
		cfg        utils.AuthConfig
		remember   bool
		wantTTL    time.Duration
		persistent bool
	}{
		{"default session", utils.AuthConfig{}, false, 12 * time.Hour, false},
		{"remember me, default 7 days", utils.AuthConfig{}, true, 7 * 24 * time.Hour, true},
		{"remember me, configured", utils.AuthConfig{RememberMeDays: 30, SessionTTLHours: 1}, true, 30 * 24 * time.Hour, true},
		{"no remember, configured", utils.AuthConfig{RememberMeDays: 30, SessionTTLHours: 1}, false, time.Hour, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, tc.cfg, pwUser(t))
			h.svc.now = func() time.Time { return fixed }
			sess, err := h.svc.LoginWithPassword(context.Background(), "an.nguyen@mq.com", "correct-horse-1", ClientMeta{RememberMe: tc.remember})
			if err != nil {
				t.Fatal(err)
			}
			if got := sess.ExpiresAt.Sub(fixed); got != tc.wantTTL {
				t.Errorf("TTL = %v, want %v", got, tc.wantTTL)
			}
			if sess.Persistent != tc.persistent {
				t.Errorf("Persistent = %v, want %v", sess.Persistent, tc.persistent)
			}
			if stored := h.sessions.rows[hashToken(sess.Token)].expires; !stored.Equal(sess.ExpiresAt) {
				t.Errorf("stored expiry %v != returned %v", stored, sess.ExpiresAt)
			}
		})
	}
}

func TestRememberMe_GoogleCarriesChoiceThroughAttempt(t *testing.T) {
	for _, remember := range []bool{true, false} {
		h := newHarness(t, mqConfig(), activeUser(1, "an.nguyen@mq.com"))
		ctx := context.Background()
		if _, err := h.svc.BeginGoogleLogin(ctx, "/profile", remember); err != nil {
			t.Fatal(err)
		}
		var state string
		for s, a := range h.attempts.rows {
			state = s
			if a.RememberMe != remember {
				t.Errorf("attempt.RememberMe = %v, want %v", a.RememberMe, remember)
			}
		}
		// The callback request itself can't change the choice made on the login page.
		sess, _, err := h.svc.CompleteGoogleLogin(ctx, GoogleCallbackInput{Code: "c", State: state, Meta: ClientMeta{RememberMe: !remember}})
		if err != nil {
			t.Fatal(err)
		}
		if sess.Persistent != remember {
			t.Errorf("remember=%v: session Persistent = %v", remember, sess.Persistent)
		}
	}
}

func TestLogoutAll(t *testing.T) {
	other := pwUser(t)
	other.ID, other.Email = 2, "other@mq.com"
	h := newHarness(t, utils.AuthConfig{}, pwUser(t), other)
	ctx := context.Background()

	var mine []string
	for i := 0; i < 3; i++ { // three devices
		s, err := h.svc.LoginWithPassword(ctx, "an.nguyen@mq.com", "correct-horse-1", ClientMeta{})
		if err != nil {
			t.Fatal(err)
		}
		mine = append(mine, s.Token)
	}
	theirs, _ := h.svc.LoginWithPassword(ctx, "other@mq.com", "correct-horse-1", ClientMeta{})
	h.svc.Logout(ctx, mine[0]) // already logged out on one device

	n, err := h.svc.LogoutAll(ctx, 1)
	if err != nil || n != 2 {
		t.Fatalf("LogoutAll = %d, %v; want 2 (the already-revoked one is not counted)", n, err)
	}
	for i, tok := range mine {
		if _, err := h.svc.Authenticate(ctx, tok); !errors.Is(err, ErrUnauthenticated) {
			t.Errorf("device %d still signed in", i)
		}
	}
	if u, err := h.svc.Authenticate(ctx, theirs.Token); err != nil || u.ID != 2 {
		t.Errorf("another user's session was revoked: %v", err)
	}
	if n, err := h.svc.LogoutAll(ctx, 1); err != nil || n != 0 {
		t.Errorf("second LogoutAll = %d, %v; want 0, nil", n, err)
	}
}

type revokeAllFails struct{ *fakeSessions }

func (r revokeAllFails) RevokeAllForUser(context.Context, int64) (int64, error) { return 0, errDB }

func TestLogoutAll_StoreError(t *testing.T) {
	h := newHarness(t, utils.AuthConfig{})
	svc := NewAuthService(utils.AuthConfig{}, h.users, revokeAllFails{h.sessions}, h.attempts, nil)
	_, err := svc.LogoutAll(context.Background(), 1)
	wantInternal(t, "logout all", err)
}
