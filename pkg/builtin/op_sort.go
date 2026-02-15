package builtin

import (
	"fmt"
	"strconv"
	"strings"

	gosort "sort"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

func specSort() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandSort,
		Manual: "sort [-r] [-n] [-u] [ABS_FILE]",
		Tips: []string{
			"Sorts lines. Use stdin when no file is given.",
			"-r reverses order, -n sorts numerically, -u removes duplicates.",
		},
		Run: runSort,
	}
}

func runSort(runtime engine.CommandRuntime, args []string) (string, int) {
	reverse := false
	numeric := false
	unique := false
	filePath := ""

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") && arg != "-" {
			for _, ch := range arg[1:] {
				switch ch {
				case 'r':
					reverse = true
				case 'n':
					numeric = true
				case 'u':
					unique = true
				default:
					return fmt.Sprintf("sort: unsupported flag -%c", ch), contract.ExitCodeUsage
				}
			}
			continue
		}
		if filePath != "" {
			return fmt.Sprintf("sort: unexpected argument: %s", arg), contract.ExitCodeUsage
		}
		pathValue, err := runtime.Ops.RequireAbsolutePath(arg)
		if err != nil {
			return fmt.Sprintf("sort: %v", err), contract.ExitCodeUsage
		}
		filePath = pathValue
	}

	raw, out, code := loadSortSource(runtime, filePath)
	if code != 0 {
		return out, code
	}

	lines := splitRawLines(raw)
	if len(lines) == 0 {
		return "", 0
	}

	if numeric {
		gosort.SliceStable(lines, func(i, j int) bool {
			ni := parseLeadingInt(lines[i])
			nj := parseLeadingInt(lines[j])
			if reverse {
				return ni > nj
			}
			return ni < nj
		})
	} else {
		gosort.SliceStable(lines, func(i, j int) bool {
			if reverse {
				return lines[i] > lines[j]
			}
			return lines[i] < lines[j]
		})
	}

	if unique {
		deduped := make([]string, 0, len(lines))
		for i, line := range lines {
			if i == 0 || line != lines[i-1] {
				deduped = append(deduped, line)
			}
		}
		lines = deduped
	}

	return strings.Join(lines, "\n"), 0
}

func loadSortSource(runtime engine.CommandRuntime, filePath string) (string, string, int) {
	if runtime.HasStdin && filePath == "" {
		return runtime.Stdin, "", 0
	}
	if filePath == "" {
		return "", "sort: expected stdin input or one absolute file path", contract.ExitCodeUsage
	}
	raw, err := runtime.Ops.ReadRawContent(runtime.Ctx, filePath)
	if err != nil {
		return "", fmt.Sprintf("sort: %v", err), contract.ExitCodeGeneral
	}
	return raw, "", 0
}

func parseLeadingInt(s string) int {
	s = strings.TrimSpace(s)
	end := 0
	for end < len(s) && ((s[end] >= '0' && s[end] <= '9') || (end == 0 && s[end] == '-')) {
		end++
	}
	if end == 0 {
		return 0
	}
	v, err := strconv.Atoi(s[:end])
	if err != nil {
		return 0
	}
	return v
}
