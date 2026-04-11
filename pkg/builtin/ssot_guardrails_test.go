package builtin

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestDefaultBuiltinRegistrationsDeclareCanonicalOwnership(t *testing.T) {
	registrations := defaultBuiltinRegistrations()
	if len(registrations) == 0 {
		t.Fatal("defaultBuiltinRegistrations() returned no registrations")
	}

	seenNames := make(map[string]string, len(registrations))
	for _, registration := range registrations {
		if strings.TrimSpace(registration.Name) == "" {
			t.Fatal("default builtin registration has empty name")
		}
		if registration.Build == nil {
			t.Fatalf("default builtin registration %q has nil Build", registration.Name)
		}
		if strings.TrimSpace(registration.CanonicalSource) == "" {
			t.Fatalf("default builtin registration %q is missing CanonicalSource", registration.Name)
		}
		if !strings.HasPrefix(registration.CanonicalSource, "pkg/builtin/") {
			t.Fatalf("default builtin registration %q canonical source %q must stay under pkg/builtin", registration.Name, registration.CanonicalSource)
		}
		if prev, exists := seenNames[registration.Name]; exists {
			t.Fatalf("default builtin registration %q declared twice: %s and %s", registration.Name, prev, registration.CanonicalSource)
		}
		seenNames[registration.Name] = registration.CanonicalSource

		spec := registration.Build()
		if spec.Name != registration.Name {
			t.Fatalf("default builtin registration %q built spec %q", registration.Name, spec.Name)
		}
	}
}

func TestDefaultBuiltinRegistrationsLeaveNoShadowCommandImplementations(t *testing.T) {
	commandGlob := filepath.Join("commands", "*", "command.go")
	commandFiles, err := filepath.Glob(commandGlob)
	if err != nil {
		t.Fatalf("glob command packages failed: %v", err)
	}
	if len(commandFiles) != 0 {
		slices.Sort(commandFiles)
		t.Fatalf("unexpected shadow command implementations remain: %v", commandFiles)
	}
}

func TestCanonicalBuiltinSourceTreeIsLegacySpecSurface(t *testing.T) {
	if canonicalDefaultBuiltinSourceTree != "pkg/builtin/spec* implementations in pkg/builtin/*.go" {
		t.Fatalf("canonicalDefaultBuiltinSourceTree changed unexpectedly: %q", canonicalDefaultBuiltinSourceTree)
	}
}
