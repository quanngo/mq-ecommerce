package utils

import "testing"

func TestAuthConfigWithDefaults(t *testing.T) {
	got := AuthConfig{}.WithDefaults()
	want := AuthConfig{
		FrontendURL: "http://localhost:5173", CookieName: "ecom_session", SessionTTLHours: 12,
		LoginAttemptTTLSeconds: 600, MaxFailedLogins: 5, LockoutMinutes: 15,
	}
	if got.FrontendURL != want.FrontendURL || got.CookieName != want.CookieName || got.SessionTTLHours != want.SessionTTLHours ||
		got.LoginAttemptTTLSeconds != want.LoginAttemptTTLSeconds || got.MaxFailedLogins != want.MaxFailedLogins || got.LockoutMinutes != want.LockoutMinutes {
		t.Errorf("defaults = %+v", got)
	}
	if got.CookieSecure || got.Google.Enable || got.Google.AutoProvision {
		t.Error("security-relevant booleans must not be switched on by defaults")
	}
}

func TestAuthConfigWithDefaults_KeepsConfiguredValues(t *testing.T) {
	in := AuthConfig{
		FrontendURL: "https://admin.mq.com", CookieName: "s", CookieSecure: true, SessionTTLHours: 1,
		LoginAttemptTTLSeconds: 60, MaxFailedLogins: 3, LockoutMinutes: 30,
		Google: GoogleAuthConfig{Enable: true, AllowedDomains: []string{"mq.com"}},
	}
	got := in.WithDefaults()
	if got.FrontendURL != in.FrontendURL || got.CookieName != "s" || !got.CookieSecure || got.SessionTTLHours != 1 ||
		got.LoginAttemptTTLSeconds != 60 || got.MaxFailedLogins != 3 || got.LockoutMinutes != 30 || len(got.Google.AllowedDomains) != 1 {
		t.Errorf("configured values overwritten: %+v", got)
	}
}

func TestAuthConfigWithDefaults_NegativeBecomesDefault(t *testing.T) {
	got := AuthConfig{SessionTTLHours: -1, MaxFailedLogins: -5}.WithDefaults()
	if got.SessionTTLHours != 12 || got.MaxFailedLogins != 5 {
		t.Errorf("negative values kept: %+v", got)
	}
}
