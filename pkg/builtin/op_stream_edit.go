package builtin

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

var sedPrintExprRe = regexp.MustCompile(`^(\d+)(?:,(\d+))?p$`)

func specSed() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandSed,
		Manual: "sed -i [--json] 's/old/new/[g]' PATH | sed -n 'Np'|'M,Np' [PATH]",
		Tips: []string{
			"Only a focused subset of sed is supported for deterministic behavior.",
			"Print mode stays text-first; use --json only with -i when you want a machine-readable mutation summary.",
			"Use -n with Np or M,Np to print line ranges.",
		},
		StructuredOutput: "in-place edit summary",
		StructuredFlags:  []string{"-i --json"},
		Examples:         ExamplesFor("sed"),
		DetailedManual:   LoadEmbeddedManual("sed"),
		Run:              runSed,
	}
}

func runSed(runtime engine.CommandRuntime, args []string) (string, int) {
	if len(args) == 0 {
		return "sed: missing arguments", contract.ExitCodeUsage
	}
	switch args[0] {
	case "-i":
		return runSedInPlace(runtime, args)
	case "-n":
		for _, arg := range args[1:] {
			if arg == "--json" {
				return "sed: --json is only supported with -i", contract.ExitCodeUsage
			}
		}
		return runSedPrint(runtime, args)
	default:
		return fmt.Sprintf("sed: unsupported flag %s", args[0]), contract.ExitCodeUsage
	}
}

func runSedInPlace(runtime engine.CommandRuntime, args []string) (string, int) {
	filteredArgs, _, jsonOutput, out, code, ok := extractMutationOutputFlags("sed", args[1:])
	if !ok {
		return out, code
	}
	if len(filteredArgs) != 2 {
		return "sed: only supports -i 's/old/new/[g]' PATH", contract.ExitCodeUsage
	}
	oldValue, newValue, replaceAll, err := parseSedSubstituteExpr(filteredArgs[0])
	if err != nil {
		return fmt.Sprintf("sed: %v", err), contract.ExitCodeUsage
	}
	pathValue, err := runtime.Ops.RequireAbsolutePath(filteredArgs[1])
	if err != nil {
		return fmt.Sprintf("sed: %v", err), contract.ExitCodeUsage
	}
	if !runtime.Ops.Policy.AllowWrite() {
		traceDeniedPaths(runtime, pathValue)
		return "sed: write is not allowed by policy", contract.ExitCodeUnsupported
	}
	if err := runtime.Ops.EditFile(runtime.Ctx, pathValue, oldValue, newValue, replaceAll); err != nil {
		if errors.Is(err, contract.ErrUnsupported) {
			return "sed: in-place edit is not supported", contract.ExitCodeUnsupported
		}
		return fmt.Sprintf("sed: %v", err), contract.ExitCodeGeneral
	}
	if jsonOutput {
		raw, err := json.Marshal(struct {
			Path       string `json:"path"`
			Mode       string `json:"mode"`
			Old        string `json:"old"`
			New        string `json:"new"`
			ReplaceAll bool   `json:"replace_all"`
		}{
			Path:       pathValue,
			Mode:       "in_place_edit",
			Old:        oldValue,
			New:        newValue,
			ReplaceAll: replaceAll,
		})
		if err != nil {
			return fmt.Sprintf("sed: %v", err), contract.ExitCodeGeneral
		}
		return string(raw), 0
	}
	return "", 0
}

func runSedPrint(runtime engine.CommandRuntime, args []string) (string, int) {
	if len(args) < 2 || len(args) > 3 {
		return "sed: only supports -n 'Np' or -n 'M,Np' [PATH]", contract.ExitCodeUsage
	}
	start, end, err := parseSedPrintExpr(args[1])
	if err != nil {
		return fmt.Sprintf("sed: %v", err), contract.ExitCodeUsage
	}

	raw := ""
	if len(args) == 3 {
		pathValue, pathErr := runtime.Ops.RequireAbsolutePath(args[2])
		if pathErr != nil {
			return fmt.Sprintf("sed: %v", pathErr), contract.ExitCodeUsage
		}
		raw, err = runtime.Ops.ReadRawContent(runtime.Ctx, pathValue)
		if err != nil {
			return fmt.Sprintf("sed: %v", err), contract.ExitCodeGeneral
		}
	} else {
		if !runtime.HasStdin {
			return "sed: -n requires stdin input or one file path", contract.ExitCodeUsage
		}
		raw = runtime.Stdin
	}

	lines := splitRawLines(raw)
	if len(lines) == 0 || start > len(lines) {
		return "", 0
	}
	if end > len(lines) {
		end = len(lines)
	}
	if start > end {
		return "", 0
	}
	return strings.Join(lines[start-1:end], "\n"), 0
}

func parseSedPrintExpr(expr string) (int, int, error) {
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" {
		return 0, 0, fmt.Errorf("print expression must not be empty")
	}
	matches := sedPrintExprRe.FindStringSubmatch(trimmed)
	if len(matches) == 0 {
		return 0, 0, fmt.Errorf("print expression must be Np or M,Np")
	}
	start, err := strconv.Atoi(matches[1])
	if err != nil || start <= 0 {
		return 0, 0, fmt.Errorf("line index must be positive")
	}
	end := start
	if len(matches) > 2 && strings.TrimSpace(matches[2]) != "" {
		end, err = strconv.Atoi(matches[2])
		if err != nil || end <= 0 {
			return 0, 0, fmt.Errorf("line index must be positive")
		}
	}
	if end < start {
		return 0, 0, fmt.Errorf("range end must be greater than or equal to start")
	}
	return start, end, nil
}

func parseSedSubstituteExpr(expr string) (string, string, bool, error) {
	if len(expr) < 4 || expr[0] != 's' {
		return "", "", false, fmt.Errorf("expression must start with s")
	}
	delim := expr[1]
	parts := make([]string, 0, 4)
	current := strings.Builder{}
	escaped := false
	for i := 2; i < len(expr); i++ {
		ch := expr[i]
		if escaped {
			current.WriteByte(ch)
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == delim {
			parts = append(parts, current.String())
			current.Reset()
			continue
		}
		current.WriteByte(ch)
	}
	parts = append(parts, current.String())
	if len(parts) < 3 {
		return "", "", false, fmt.Errorf("expression must be s%cold%cnew%c[g]", delim, delim, delim)
	}
	oldValue := parts[0]
	newValue := parts[1]
	flags := parts[2]
	if strings.TrimSpace(oldValue) == "" {
		return "", "", false, fmt.Errorf("old string must not be empty")
	}
	replaceAll := strings.Contains(flags, "g")
	return oldValue, newValue, replaceAll, nil
}
