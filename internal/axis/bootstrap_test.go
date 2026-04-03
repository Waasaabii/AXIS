package axis

import "testing"

func TestDetermineBootstrapNextStep(t *testing.T) {
	t.Run("首次初始化时未登录先去 setup", func(t *testing.T) {
		nextStep, blockingReason := determineBootstrapNextStep(
			BootstrapAuthStatus{Authenticated: false},
			SetupState{Required: true, NeedsPasswordReset: true},
			MainServiceStatus{Ready: true},
		)
		if nextStep != "setup" || blockingReason != "" {
			t.Fatalf("unexpected result: %s %s", nextStep, blockingReason)
		}
	})

	t.Run("未登录且无需初始化时先去登录", func(t *testing.T) {
		nextStep, blockingReason := determineBootstrapNextStep(
			BootstrapAuthStatus{Authenticated: false},
			SetupState{Required: false},
			MainServiceStatus{Ready: true},
		)
		if nextStep != "login" || blockingReason != "" {
			t.Fatalf("unexpected result: %s %s", nextStep, blockingReason)
		}
	})

	t.Run("已登录但未初始化先去 setup", func(t *testing.T) {
		nextStep, blockingReason := determineBootstrapNextStep(
			BootstrapAuthStatus{Authenticated: true},
			SetupState{Required: true, NeedsPasswordReset: true},
			MainServiceStatus{Ready: true},
		)
		if nextStep != "setup" || blockingReason != "" {
			t.Fatalf("unexpected result: %s %s", nextStep, blockingReason)
		}
	})

	t.Run("已登录但仅缺订阅等配置时直接进 dashboard", func(t *testing.T) {
		nextStep, blockingReason := determineBootstrapNextStep(
			BootstrapAuthStatus{Authenticated: true},
			SetupState{Required: true, NeedsPasswordReset: false},
			MainServiceStatus{Ready: true},
		)
		if nextStep != "dashboard" || blockingReason != "" {
			t.Fatalf("unexpected result: %s %s", nextStep, blockingReason)
		}
	})

	t.Run("主服务未就绪时停在 launch", func(t *testing.T) {
		nextStep, blockingReason := determineBootstrapNextStep(
			BootstrapAuthStatus{Authenticated: true},
			SetupState{Required: false},
			MainServiceStatus{Ready: false, Message: "主服务未就绪"},
		)
		if nextStep != "launch" {
			t.Fatalf("unexpected next step: %s", nextStep)
		}
		if blockingReason == "" {
			t.Fatal("blocking reason should not be empty")
		}
	})
}
