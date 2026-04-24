package axis

import "testing"

func TestBuildSetupStateRequiresSetupWithoutNodeSource(t *testing.T) {
	config := &Config{Admin: AdminConfig{Username: "admin", RequiresPasswordReset: false}}
	state := BuildSetupState(config)
	if !state.Required || state.HasNodeSources {
		t.Fatalf("缺少节点来源时应要求初始化: %#v", state)
	}
}

func TestBuildSetupStateReadyWhenCoreDataExists(t *testing.T) {
	config := newValidProductConfig()
	config.Admin.RequiresPasswordReset = false
	state := BuildSetupState(config)
	if state.Required {
		t.Fatalf("完整核心数据时不应要求 setup: %#v", state)
	}
	if !state.HasNodeSources || !state.HasRoutes || !state.HasUsage {
		t.Fatal("完整核心数据应全部就绪")
	}
}

func TestBuildSetupStateIncludesAdminUsername(t *testing.T) {
	config := &Config{Admin: AdminConfig{Username: "axis-admin", RequiresPasswordReset: true}}
	state := BuildSetupState(config)
	if state.AdminUsername != "axis-admin" {
		t.Fatalf("expected admin username to be exposed, got %q", state.AdminUsername)
	}
}
