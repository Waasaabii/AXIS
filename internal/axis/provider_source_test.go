package axis

import (
	"strings"
	"testing"
)

func TestParseManualProviderInputSupportsColonStyle(t *testing.T) {
	subscription, err := parseManualProviderInput("socks5://207.211.181.215:45001:user:pass", "socks5")
	if err != nil {
		t.Fatalf("parseManualProviderInput() error = %v", err)
	}
	if subscription.Type != "socks5" || subscription.Server != "207.211.181.215" || subscription.Port != 45001 {
		t.Fatalf("unexpected parsed subscription: %#v", subscription)
	}
	if subscription.Username != "user" || subscription.Password != "pass" {
		t.Fatalf("unexpected credentials: %#v", subscription)
	}
}

func TestParseManualProviderInputSupportsAtStyle(t *testing.T) {
	subscription, err := parseManualProviderInput("http://alice:secret@proxy.example.com:8080", "socks5")
	if err != nil {
		t.Fatalf("parseManualProviderInput() error = %v", err)
	}
	if subscription.Type != "http" || subscription.Server != "proxy.example.com" || subscription.Port != 8080 {
		t.Fatalf("unexpected parsed subscription: %#v", subscription)
	}
	if subscription.Username != "alice" || subscription.Password != "secret" {
		t.Fatalf("unexpected credentials: %#v", subscription)
	}
}

func TestBuildManualProviderFileContent(t *testing.T) {
	content, err := buildManualProviderFileContent(Subscription{
		Name:     "tokyo-fixed",
		Type:     "socks",
		Server:   "proxy.example.com",
		Port:     9001,
		Username: "relay",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("buildManualProviderFileContent() error = %v", err)
	}
	raw := string(content)
	for _, expected := range []string{
		"name: tokyo-fixed",
		"type: socks5",
		"server: proxy.example.com",
		"port: 9001",
		"username: relay",
		"password: secret",
	} {
		if !strings.Contains(raw, expected) {
			t.Fatalf("manual provider file missing %q:\n%s", expected, raw)
		}
	}
}
