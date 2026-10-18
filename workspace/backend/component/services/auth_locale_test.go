package services

import (
	"context"
	"errors"
	"main/component/models"
	"testing"
)

func TestSetLocale(t *testing.T) {
	h := newHarness(t, mqConfig(), activeUser(1, "an.nguyen@mq.com"))
	actor := &models.SessionUser{AppUser: models.AppUser{ID: 1}}

	if err := h.svc.SetLocale(context.Background(), actor, "en"); err != nil {
		t.Fatalf("SetLocale(en) = %v", err)
	}
	if got := h.users.byID[1].Locale; got != "en" {
		t.Errorf("stored locale = %q, want en", got)
	}

	for _, bad := range []string{"", "fr", "EN", "en-US", "vi; DROP TABLE"} {
		if err := h.svc.SetLocale(context.Background(), actor, bad); !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("SetLocale(%q) = %v, want INVALID_REQUEST", bad, err)
		}
	}
	if got := h.users.byID[1].Locale; got != "en" {
		t.Errorf("rejected values changed the locale to %q", got)
	}

	if err := h.svc.SetLocale(context.Background(), nil, "vi"); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("nil actor = %v, want UNAUTHENTICATED", err)
	}
	ghost := &models.SessionUser{AppUser: models.AppUser{ID: 99}}
	if err := h.svc.SetLocale(context.Background(), ghost, "vi"); !errors.Is(err, ErrInternal) {
		t.Errorf("store failure = %v, want INTERNAL_ERROR", err)
	}
}

func TestCreateUserLocale(t *testing.T) {
	h := newHarness(t, mqConfig())
	id, err := h.svc.CreateUser(context.Background(), CreateUserInput{Email: "en.user@mq.com", Locale: "en"})
	if err != nil {
		t.Fatal(err)
	}
	if got := h.users.byID[id].Locale; got != "en" {
		t.Errorf("locale = %q, want en", got)
	}
	if _, err := h.svc.CreateUser(context.Background(), CreateUserInput{Email: "fr.user@mq.com", Locale: "fr"}); err == nil {
		t.Error("unsupported locale was accepted")
	}
}
