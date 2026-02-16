package uniq

import (
	"embed"
	"fmt"
	"strings"

	"github.com/khicago/simsh/pkg/builtin/shared"
	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

var examples = []string{"sort /task_outputs/log.txt | uniq -c", "uniq -d /task_outputs/sorted.txt"}

//go:embed manual.md
var manualFS embed.FS

func detailedManual() string {
	data, err := manualFS.ReadFile("manual.md")
	if err != nil {
		return ""
	}
	return string(data)
}

func Spec() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   "uniq",
		Manual: "uniq [-c] [-d] [ABS_FILE]",
		Tips: []string{
			"Removes adjacent duplicate lines. Use stdin when no file is given.",
			"-c prefixes lines with occurrence count, -d only prints duplicates.",
		},
		Examples:       append([]string(nil), examples...),
		DetailedManual: detailedManual(),
		Run:            run,
	}
}

func run(runtime engine.CommandRuntime, args []string) (string, int) {
	showCount := false
	duplicatesOnly := false
	filePath := ""

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") && arg != "-" {
			for _, ch := range arg[1:] {
				switch ch {
				case 'c':
					showCount = true
				case 'd':
					duplicatesOnly = true
				default:
					return fmt.Sprintf("uniq: unsupported flag -%c", ch), contract.ExitCodeUsage
				}
			}
			continue
		}
		if filePath != "" {
			return fmt.Sprintf("uniq: unexpected argument: %s", arg), contract.ExitCodeUsage
		}
		pathValue, err := runtime.Ops.RequireAbsolutePath(arg)
		if err != nil {
			return fmt.Sprintf("uniq: %v", err), contract.ExitCodeUsage
		}
		filePath = pathValue
	}

	raw, out, code := loadSource(runtime, filePath)
	if code != 0 {
		return out, code
	}

	lines := shared.SplitRawLines(raw)
	if len(lines) == 0 {
		return "", 0
	}

	type group struct {
		line  string
		count int
	}
	groups := make([]group, 0, len(lines))
	for _, line := range lines {
		if len(groups) > 0 && groups[len(groups)-1].line == line {
			groups[len(groups)-1].count++
		} else {
			groups = append(groups, group{line: line, count: 1})
		}
	}

	result := make([]string, 0, len(groups))
	for _, g := range groups {
		if duplicatesOnly && g.count < 2 {
			continue
		}
		if showCount {
			result = append(result, fmt.Sprintf("%d %s", g.count, g.line))
		} else {
			result = append(result, g.line)
		}
	}
	return strings.Join(result, "\n"), 0
}

func loadSource(runtime engine.CommandRuntime, filePath string) (string, string, int) {
	if runtime.HasStdin && filePath == "" {
		return runtime.Stdin, "", 0
	}
	if filePath == "" {
		return "", "uniq: expected stdin input or one absolute file path", contract.ExitCodeUsage
	}
	raw, err := runtime.Ops.ReadRawContent(runtime.Ctx, filePath)
	if err != nil {
		return "", fmt.Sprintf("uniq: %v", err), contract.ExitCodeGeneral
	}
	return raw, "", 0
}
