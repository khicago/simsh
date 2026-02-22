package main

import "testing"

func TestParseCLIOptionsServePort(t *testing.T) {
	opts, err := parseCLIOptions([]string{"serve", "-P", "19090", "-profile", "bash-plus", "-policy", "full", "-mount", "test", "-rc", "/etc/simshrc"})
	if err != nil {
		t.Fatalf("parse serve options failed: %v", err)
	}
	if opts.mode != modeServe {
		t.Fatalf("mode = %q, want %q", opts.mode, modeServe)
	}
	if opts.port != 19090 {
		t.Fatalf("port = %d, want %d", opts.port, 19090)
	}
	if opts.profile != "bash-plus" {
		t.Fatalf("profile = %q, want %q", opts.profile, "bash-plus")
	}
	if opts.policy != "full" {
		t.Fatalf("policy = %q, want %q", opts.policy, "full")
	}
	if len(opts.mounts) != 1 || opts.mounts[0] != "test" {
		t.Fatalf("mounts = %v, want [test]", opts.mounts)
	}
	if len(opts.rcFiles) != 1 || opts.rcFiles[0] != "/etc/simshrc" {
		t.Fatalf("rcFiles = %v, want [/etc/simshrc]", opts.rcFiles)
	}
}

func TestParseCLIOptionsRunDefaults(t *testing.T) {
	opts, err := parseCLIOptions(nil)
	if err != nil {
		t.Fatalf("parse run options failed: %v", err)
	}
	if opts.mode != modeRun {
		t.Fatalf("mode = %q, want %q", opts.mode, modeRun)
	}
	if opts.command != "" {
		t.Fatalf("command = %q, want empty", opts.command)
	}
}

func TestParseCLIOptionsInvalidMount(t *testing.T) {
	_, err := parseCLIOptions([]string{"serve", "-mount", "invalid"})
	if err == nil {
		t.Fatalf("expected error for invalid mount")
	}
}
