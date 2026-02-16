package mount

import (
	"context"
	"strings"
	"testing"
)

func TestStaticMountKeepsNestedChildren(t *testing.T) {
	files := map[string]string{
		"/test/README.md":                       "# root\n",
		"/test/core-strict/README.md":           "# profile\n",
		"/test/core-strict/cases/echo-basic.sh": "echo hello\n",
	}

	// Repeat construction to guard against map-iteration non-determinism.
	for i := 0; i < 200; i++ {
		m, err := NewStaticMount("/test", "test", files)
		if err != nil {
			t.Fatalf("iteration %d: NewStaticMount failed: %v", i, err)
		}
		children, err := m.ListChildren(context.Background(), "/test")
		if err != nil {
			t.Fatalf("iteration %d: ListChildren(/test) failed: %v", i, err)
		}
		joined := strings.Join(children, "\n")
		if !strings.Contains(joined, "/test/core-strict") {
			t.Fatalf("iteration %d: missing /test/core-strict in children: %q", i, joined)
		}
		if !strings.Contains(joined, "/test/README.md") {
			t.Fatalf("iteration %d: missing /test/README.md in children: %q", i, joined)
		}
	}
}
