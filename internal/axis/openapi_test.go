package axis

import "testing"

func TestBuildOpenAPISpecIncludesTypedContracts(t *testing.T) {
	spec := BuildOpenAPISpec()

	components, ok := spec["components"].(map[string]any)
	if !ok {
		t.Fatal("components 缺失")
	}

	schemas, ok := components["schemas"].(map[string]any)
	if !ok {
		t.Fatal("schemas 缺失")
	}

	for _, name := range []string{"Config", "StatusResponse", "MihomoVersionsResponse"} {
		if _, exists := schemas[name]; !exists {
			t.Fatalf("schema %s 缺失", name)
		}
	}

	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		t.Fatal("paths 缺失")
	}

	for _, pathname := range []string{
		"/api/config",
		"/api/session",
		"/api/mihomo/versions/{version}/activate",
	} {
		if _, exists := paths[pathname]; !exists {
			t.Fatalf("path %s 缺失", pathname)
		}
	}
}
