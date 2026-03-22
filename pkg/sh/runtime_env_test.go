package sh

import (
	"context"
	"strings"
	"testing"

	simfs "github.com/khicago/simsh/pkg/fs"
)

func TestRuntimeProvidesBuiltinsAndMarkdownDescription(t *testing.T) {
	runtime := NewRuntime()

	docs := runtime.BuiltinDocs()
	if len(docs) == 0 {
		t.Fatal("Runtime.BuiltinDocs() returned no builtin docs")
	}
	foundEcho := false
	for _, doc := range docs {
		if doc.Name == "echo" {
			foundEcho = true
			break
		}
	}
	if !foundEcho {
		t.Fatalf("Runtime.BuiltinDocs() did not include echo: %#v", docs)
	}

	ops, err := simfs.NewRuntimeOps(simfs.EnvironmentOptions{})
	if err != nil {
		t.Fatalf("fs.NewRuntimeOps(...) error = %v", err)
	}
	stdout, exitCode := runtime.Execute(context.Background(), "echo hello", ops)
	if exitCode != 0 {
		t.Fatalf("Runtime.Execute(%q) exitCode = %d, want 0", "echo hello", exitCode)
	}
	if stdout != "hello" {
		t.Fatalf("Runtime.Execute(%q) stdout = %q, want %q", "echo hello", stdout, "hello")
	}

	markdown := DescribeMarkdown()
	for _, want := range []string{"## Shell Runtime", "#### echo", "`/sys/bin`", "`/bin`"} {
		if !strings.Contains(markdown, want) {
			t.Errorf("DescribeMarkdown() missing %q in output:\n%s", want, markdown)
		}
	}
}
