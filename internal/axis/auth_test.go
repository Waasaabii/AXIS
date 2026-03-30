package axis

import "testing"

func TestPasswordHashAndSession(t *testing.T) {
	hash, err := CreatePasswordHash("secret-pass")
	if err != nil {
		t.Fatalf("CreatePasswordHash() error = %v", err)
	}

	auth := NewAuthContext("admin", hash, "", "test-secret", 1)
	if !auth.VerifyCredentials("admin", "secret-pass") {
		t.Fatalf("expected credentials to verify")
	}
	if auth.VerifyCredentials("admin", "wrong") {
		t.Fatalf("expected wrong password to fail")
	}

	cookie, err := auth.CreateSessionCookie("admin")
	if err != nil {
		t.Fatalf("CreateSessionCookie() error = %v", err)
	}
	sessionValue := ParseCookies(cookie)["proxyrelay_session"]
	valid, username := auth.ReadSession(sessionValue)
	if !valid || username != "admin" {
		t.Fatalf("expected valid session, got valid=%v username=%q", valid, username)
	}
}
