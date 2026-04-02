package axis

import "testing"

func TestBuildFallbackRuntimeMembers(t *testing.T) {
	members := buildFallbackRuntimeMembers(EgressGroup{
		Name:    "egress-hk-fallback",
		Proxies: []string{"HK 01", "HK 02", "HK 01"},
	}, []string{"HK 01", "HK 02", "HK 03"})

	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
	if members[0].Order != 1 || members[0].Name != "HK 01" {
		t.Fatalf("unexpected first member: %#v", members[0])
	}
	if members[1].Order != 2 || members[1].Name != "HK 02" {
		t.Fatalf("unexpected second member: %#v", members[1])
	}
}
