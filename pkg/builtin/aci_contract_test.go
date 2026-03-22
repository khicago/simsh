package builtin

import (
	"context"
	"strings"
	"testing"

	"github.com/khicago/simsh/pkg/engine"
)

func TestCwdAwareCommandDocsAvoidAbsoluteOnlyLanguage(t *testing.T) {
	reg := engine.NewRegistry()
	RegisterDefaults(reg)

	docs := map[string]engineDoc{}
	for _, doc := range reg.BuiltinCommandDocs() {
		docs[doc.Name] = engineDoc{
			manual:         doc.Manual,
			detailedManual: doc.DetailedManual,
		}
	}

	commands := []string{
		"ls", "tree", "find", "cat", "grep", "head", "tail", "wc",
		"sort", "uniq", "touch", "tee", "sed", "mkdir", "rm", "rmdir",
		"cp", "mv", "diff", "frontmatter",
	}
	banned := []string{
		"ABS_",
		"All paths must be absolute",
		"absolute file path",
		"absolute file/directory path",
		"virtual root directory",
		"virtual runtime root",
		"searches from the virtual root",
	}

	for _, name := range commands {
		doc, ok := docs[name]
		if !ok {
			t.Fatalf("missing builtin doc for %q", name)
		}
		combined := strings.Join([]string{
			doc.manual,
			doc.detailedManual,
			LoadEmbeddedManual(name),
		}, "\n")
		for _, needle := range banned {
			if strings.Contains(combined, needle) {
				t.Fatalf("%s docs still contain stale absolute-only language %q", name, needle)
			}
		}
	}
}

func TestCwdAwareUsageErrorsAvoidAbsoluteOnlyLanguage(t *testing.T) {
	rt := newTestRuntime(t)
	cases := []string{
		"cat a b",
		"grep hello",
		"head",
		"tail",
		"wc",
		"sort",
		"uniq",
		"sed -n '1p'",
		"frontmatter stat",
		"frontmatter get a b",
	}

	for _, cmd := range cases {
		out, code := rt.eng.Execute(context.Background(), cmd, rt.ops)
		if code == 0 {
			t.Fatalf("expected non-zero for %q, out=%q", cmd, out)
		}
		if strings.Contains(strings.ToLower(out), "absolute") {
			t.Fatalf("usage error for %q still contains absolute-only wording: %q", cmd, out)
		}
	}
}

type engineDoc struct {
	manual         string
	detailedManual string
}
