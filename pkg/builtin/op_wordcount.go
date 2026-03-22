package builtin

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

type wcResult struct {
	Lines *int `json:"lines,omitempty"`
	Words *int `json:"words,omitempty"`
	Bytes *int `json:"bytes,omitempty"`
}

func specWc() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandWc,
		Manual: "wc [--json] [-l] [-w] [-c] [PATH]",
		Tips: []string{
			"Single-metric modes keep bare numeric output for pipeline composability.",
			"Default multi-metric output uses compact labels, and --json provides an explicit structured mode.",
		},
		StructuredOutput: "count summary object",
		StructuredFlags:  []string{"--json"},
		Examples:         ExamplesFor("wc"),
		DetailedManual:   LoadEmbeddedManual("wc"),
		Run:              runWc,
	}
}

func runWc(runtime engine.CommandRuntime, args []string) (string, int) {
	showLines := false
	showWords := false
	showBytes := false
	jsonOutput := false
	filePath := ""

	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOutput = true
			continue
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			for _, ch := range arg[1:] {
				switch ch {
				case 'l':
					showLines = true
				case 'w':
					showWords = true
				case 'c':
					showBytes = true
				default:
					return fmt.Sprintf("wc: unsupported flag -%c", ch), contract.ExitCodeUsage
				}
			}
			continue
		}
		if filePath != "" {
			return fmt.Sprintf("wc: unexpected argument: %s", arg), contract.ExitCodeUsage
		}
		pathValue, err := runtime.Ops.RequireAbsolutePath(arg)
		if err != nil {
			return fmt.Sprintf("wc: %v", err), contract.ExitCodeUsage
		}
		filePath = pathValue
	}

	if !showLines && !showWords && !showBytes {
		showLines = true
		showWords = true
		showBytes = true
	}

	raw, out, code := loadWcSource(runtime, filePath)
	if code != 0 {
		return out, code
	}

	lines := 0
	if len(raw) > 0 {
		lines = strings.Count(raw, "\n")
		if !strings.HasSuffix(raw, "\n") {
			lines++
		}
	}
	words := len(strings.Fields(raw))
	bytes := len(raw)

	result := wcResult{}
	enabled := 0
	if showLines {
		result.Lines = &lines
		enabled++
	}
	if showWords {
		result.Words = &words
		enabled++
	}
	if showBytes {
		result.Bytes = &bytes
		enabled++
	}
	if jsonOutput {
		rawJSON, err := json.Marshal(result)
		if err != nil {
			return fmt.Sprintf("wc: %v", err), contract.ExitCodeGeneral
		}
		return string(rawJSON), 0
	}
	if enabled == 1 {
		switch {
		case result.Lines != nil:
			return fmt.Sprintf("%d", *result.Lines), 0
		case result.Words != nil:
			return fmt.Sprintf("%d", *result.Words), 0
		default:
			return fmt.Sprintf("%d", *result.Bytes), 0
		}
	}

	parts := make([]string, 0, 3)
	if result.Lines != nil {
		parts = append(parts, fmt.Sprintf("lines=%d", *result.Lines))
	}
	if result.Words != nil {
		parts = append(parts, fmt.Sprintf("words=%d", *result.Words))
	}
	if result.Bytes != nil {
		parts = append(parts, fmt.Sprintf("bytes=%d", *result.Bytes))
	}
	return strings.Join(parts, " "), 0
}

func loadWcSource(runtime engine.CommandRuntime, filePath string) (string, string, int) {
	if runtime.HasStdin && filePath == "" {
		return runtime.Stdin, "", 0
	}
	if filePath == "" {
		return "", "wc: expected stdin input or one file path", contract.ExitCodeUsage
	}
	raw, err := runtime.Ops.ReadRawContent(runtime.Ctx, filePath)
	if err != nil {
		return "", fmt.Sprintf("wc: %v", err), contract.ExitCodeGeneral
	}
	return raw, "", 0
}
