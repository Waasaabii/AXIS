package axis

import "testing"

func TestBuildSetupStateRequiresSetupForPlaceholderSubscription(t *testing.T) {
	config := &Config{
		Admin: AdminConfig{
			Username:              "admin",
			RequiresPasswordReset: false,
		},
		Subscriptions: []Subscription{
			{Name: "sample", URL: "https://example.com/your-subscription-url"},
		},
		EgressGroups: []EgressGroup{
			{Name: "egress-hk", Provider: "sample"},
		},
		Listeners: []Listener{
			{Name: "listener-hk", Port: 1080, EgressGroup: "egress-hk"},
		},
	}

	state := BuildSetupState(config)
	if !state.Required {
		t.Fatal("占位订阅时应要求进入 setup")
	}
	if state.HasRealSubscriptions {
		t.Fatal("占位订阅不应被视为真实订阅")
	}
}

func TestBuildSetupStateReadyWhenCoreDataExists(t *testing.T) {
	config := &Config{
		Admin: AdminConfig{
			Username:              "admin",
			RequiresPasswordReset: false,
		},
		Subscriptions: []Subscription{
			{Name: "airport-main", URL: "https://sub.example.net/real"},
		},
		EgressGroups: []EgressGroup{
			{Name: "egress-hk", Provider: "airport-main"},
		},
		Listeners: []Listener{
			{Name: "listener-hk", Port: 1080, EgressGroup: "egress-hk"},
		},
	}

	state := BuildSetupState(config)
	if state.Required {
		t.Fatal("完整核心数据时不应要求 setup")
	}
	if !state.HasRealSubscriptions || !state.HasEgressGroups || !state.HasListeners {
		t.Fatal("完整核心数据应全部就绪")
	}
}
