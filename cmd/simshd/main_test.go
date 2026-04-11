package main

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/service/httpapi"
)

func TestRunUsesRootFallbackPortOverrideAndTestMount(t *testing.T) {
	t.Helper()

	sentinel := errors.New("serve stopped")
	var gotAddr string
	var gotConfig httpapi.Config
	var handlerCalls int
	var serveCalls int

	err := run([]string{"-listen", "127.0.0.1:19090", "-P", "19091", "-enable-test-mount"}, launcherDeps{
		getwd: func() (string, error) { return "/tmp/simshd-root", nil },
		newHandler: func(cfg httpapi.Config) http.Handler {
			handlerCalls++
			gotConfig = cfg
			return http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
		},
		serve: func(addr string, handler http.Handler) error {
			serveCalls++
			gotAddr = addr
			if handler == nil {
				t.Fatalf("serve received nil handler")
			}
			return sentinel
		},
		logf: func(string, ...any) {},
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("run(...) error = %v, want wrapped sentinel", err)
	}
	if gotAddr != "127.0.0.1:19091" {
		t.Fatalf("listen addr = %q, want %q", gotAddr, "127.0.0.1:19091")
	}
	if gotConfig.DefaultHostRoot != "/tmp/simshd-root" {
		t.Fatalf("DefaultHostRoot = %q, want root fallback", gotConfig.DefaultHostRoot)
	}
	if gotConfig.DefaultProfile != string(contract.ProfileCoreStrict) {
		t.Fatalf("DefaultProfile = %q, want %q", gotConfig.DefaultProfile, contract.ProfileCoreStrict)
	}
	if gotConfig.DefaultPolicy != string(contract.WriteModeReadOnly) {
		t.Fatalf("DefaultPolicy = %q, want %q", gotConfig.DefaultPolicy, contract.WriteModeReadOnly)
	}
	if !gotConfig.EnableTestMount {
		t.Fatalf("EnableTestMount = false, want true")
	}
	if handlerCalls != 1 || serveCalls != 1 {
		t.Fatalf("handlerCalls=%d serveCalls=%d, want 1 each", handlerCalls, serveCalls)
	}
}

func TestParseLaunchConfigDefaultsToLoopbackListen(t *testing.T) {
	cfg, err := parseLaunchConfig(nil, func() (string, error) { return "/tmp/simshd-root", nil })
	if err != nil {
		t.Fatalf("parseLaunchConfig(...) error = %v", err)
	}
	if cfg.listenAddr != "127.0.0.1:18080" {
		t.Fatalf("listen addr = %q, want %q", cfg.listenAddr, "127.0.0.1:18080")
	}
}

func TestRunRejectsInvalidProfileBeforeServing(t *testing.T) {
	t.Helper()

	var handlerCalled bool
	var serveCalled bool

	err := run([]string{"-profile", "broken"}, launcherDeps{
		getwd:      func() (string, error) { return "/tmp/simshd-root", nil },
		newHandler: func(httpapi.Config) http.Handler { handlerCalled = true; return nil },
		serve:      func(string, http.Handler) error { serveCalled = true; return nil },
		logf:       func(string, ...any) {},
	})
	if err == nil || !strings.Contains(err.Error(), "invalid profile") {
		t.Fatalf("run(...) error = %v, want invalid profile", err)
	}
	if handlerCalled || serveCalled {
		t.Fatalf("invalid profile still called handler=%v serve=%v", handlerCalled, serveCalled)
	}
}

func TestRunRejectsInvalidPolicyBeforeServing(t *testing.T) {
	t.Helper()

	var handlerCalled bool
	var serveCalled bool

	err := run([]string{"-policy", "broken"}, launcherDeps{
		getwd:      func() (string, error) { return "/tmp/simshd-root", nil },
		newHandler: func(httpapi.Config) http.Handler { handlerCalled = true; return nil },
		serve:      func(string, http.Handler) error { serveCalled = true; return nil },
		logf:       func(string, ...any) {},
	})
	if err == nil || !strings.Contains(err.Error(), "invalid policy") {
		t.Fatalf("run(...) error = %v, want invalid policy", err)
	}
	if handlerCalled || serveCalled {
		t.Fatalf("invalid policy still called handler=%v serve=%v", handlerCalled, serveCalled)
	}
}
