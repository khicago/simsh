package builtin

import (
	"fmt"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

func specWc() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandWc,
		Manual: "wc [-l] [-w] [-c] [ABS_FILE]",
		Tips: []string{
			"Counts lines, words, and bytes. Use stdin when no file is given.",
		},
		Run: runWc,
	}
}

func runWc(runtime engine.CommandRuntime, args []string) (string, int) {
	showLines := false
	showWords := false
	showBytes := false
	filePath := ""

	for _, arg := range args {
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

	parts := make([]string, 0, 3)
	if showLines {
		parts = append(parts, fmt.Sprintf("%d", lines))
	}
	if showWords {
		parts = append(parts, fmt.Sprintf("%d", words))
	}
	if showBytes {
		parts = append(parts, fmt.Sprintf("%d", bytes))
	}
	return strings.Join(parts, " "), 0
}

func loadWcSource(runtime engine.CommandRuntime, filePath string) (string, string, int) {
	if runtime.HasStdin && filePath == "" {
		return runtime.Stdin, "", 0
	}
	if filePath == "" {
		return "", "wc: expected stdin input or one absolute file path", contract.ExitCodeUsage
	}
	raw, err := runtime.Ops.ReadRawContent(runtime.Ctx, filePath)
	if err != nil {
		return "", fmt.Sprintf("wc: %v", err), contract.ExitCodeGeneral
	}
	return raw, "", 0
}
