package builtin

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

func specEdit() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandEdit,
		Manual: "edit [--json] [--confirm] [--all] [--count] --old OLD [--new NEW] [--] PATH",
		Tips: []string{
			"Default replace requires a unique OLD snippet so edits cannot silently hit the wrong copy.",
			"Use --all only when every match should change, and --count to inspect matches without writing.",
			"Use --confirm or --json for an explicit summary; default success is silent like other mutations.",
		},
		StructuredOutput: "edit summary object",
		StructuredFlags:  []string{"--json", "--confirm"},
		Examples:         ExamplesFor("edit"),
		DetailedManual:   LoadEmbeddedManual("edit"),
		Run:              runEdit,
	}
}

type editArgs struct {
	path       string
	old        string
	new        string
	replaceAll bool
	countOnly  bool
	confirm    bool
	jsonOutput bool
	newSet     bool
}

func runEdit(runtime engine.CommandRuntime, args []string) (string, int) {
	opts, errMsg := parseEditArgs(args, runtime.Ops.RequireAbsolutePath)
	if errMsg != "" {
		return errMsg, contract.ExitCodeUsage
	}
	if opts.old == "" {
		return "edit: --old must not be empty", contract.ExitCodeUsage
	}
	if !opts.countOnly && !opts.newSet {
		return "edit: --new is required unless --count is set", contract.ExitCodeUsage
	}

	if out, code, ok := preflightPathChecks(runtime, "edit", []pathCheck{{
		path:               opts.path,
		op:                 contract.PathOpRead,
		unsupportedMessage: "edit: read is not supported",
	}}); !ok {
		return out, code
	}

	raw, err := runtime.Ops.ReadRawContent(runtime.Ctx, opts.path)
	if err != nil {
		return formatCommandPathError("edit", opts.path, "edit: read is not supported", err), pathErrorCode(err)
	}
	matchLines := matchLineNumbers(raw, opts.old)
	matches := len(matchLines)
	if opts.countOnly {
		if opts.jsonOutput {
			encoded, marshalErr := json.Marshal(struct {
				Path    string `json:"path"`
				Old     string `json:"old"`
				Matches int    `json:"matches"`
				Lines   []int  `json:"lines"`
			}{
				Path:    opts.path,
				Old:     opts.old,
				Matches: matches,
				Lines:   matchLines,
			})
			if marshalErr != nil {
				return fmt.Sprintf("edit: %v", marshalErr), contract.ExitCodeGeneral
			}
			return string(encoded), 0
		}
		return renderEditCount(raw, opts.old, matchLines), 0
	}
	if matches == 0 {
		return "edit: old string not found", contract.ExitCodeGeneral
	}
	if !opts.replaceAll && matches > 1 {
		return fmt.Sprintf("edit: old string appears %d times on lines %s; use --all or a unique snippet", matches, formatLineList(matchLines)), contract.ExitCodeGeneral
	}
	if !runtime.Ops.Policy.AllowWrite() {
		traceDeniedPaths(runtime, opts.path)
		return "edit: write is not allowed by policy", contract.ExitCodeUnsupported
	}
	if out, code, ok := preflightPathChecks(runtime, "edit", []pathCheck{{
		path:               opts.path,
		op:                 contract.PathOpWrite,
		unsupportedMessage: "edit: write is not supported",
	}}); !ok {
		return out, code
	}
	if err := runtime.Ops.EditFile(runtime.Ctx, opts.path, opts.old, opts.new, opts.replaceAll); err != nil {
		return formatCommandPathError("edit", opts.path, "edit: in-place edit is not supported", err), pathErrorCode(err)
	}
	replaced := 1
	if opts.replaceAll {
		replaced = matches
	}
	if opts.jsonOutput {
		encoded, marshalErr := json.Marshal(struct {
			Path       string `json:"path"`
			Old        string `json:"old"`
			New        string `json:"new"`
			Matches    int    `json:"matches"`
			Replaced   int    `json:"replaced"`
			ReplaceAll bool   `json:"replace_all"`
			Lines      []int  `json:"lines"`
		}{
			Path:       opts.path,
			Old:        opts.old,
			New:        opts.new,
			Matches:    matches,
			Replaced:   replaced,
			ReplaceAll: opts.replaceAll,
			Lines:      matchLines,
		})
		if marshalErr != nil {
			return fmt.Sprintf("edit: %v", marshalErr), contract.ExitCodeGeneral
		}
		return string(encoded), 0
	}
	if opts.confirm {
		return fmt.Sprintf("replaced %d in %s", replaced, opts.path), 0
	}
	return "", 0
}

func parseEditArgs(args []string, requireAbsolutePath func(string) (string, error)) (editArgs, string) {
	opts := editArgs{}
	positional := make([]string, 0, 1)
	parseOptions := true
	for idx := 0; idx < len(args); idx++ {
		arg := args[idx]
		if parseOptions && arg == "--" {
			parseOptions = false
			continue
		}
		if !parseOptions {
			positional = append(positional, arg)
			continue
		}
		if strings.HasPrefix(arg, "--old=") {
			opts.old = strings.TrimPrefix(arg, "--old=")
			continue
		}
		if strings.HasPrefix(arg, "--new=") {
			opts.new = strings.TrimPrefix(arg, "--new=")
			opts.newSet = true
			continue
		}
		switch arg {
		case "--old":
			if idx+1 >= len(args) || isEditFlag(args[idx+1]) {
				return opts, "edit: --old requires a value"
			}
			idx++
			opts.old = args[idx]
		case "--new":
			if idx+1 >= len(args) || isEditFlag(args[idx+1]) {
				return opts, "edit: --new requires a value"
			}
			idx++
			opts.new = args[idx]
			opts.newSet = true
		case "--all":
			opts.replaceAll = true
		case "--count":
			opts.countOnly = true
		case "--json":
			opts.jsonOutput = true
		case "--confirm":
			opts.confirm = true
		default:
			if strings.HasPrefix(arg, "--") {
				return opts, fmt.Sprintf("edit: unsupported flag %s", arg)
			}
			positional = append(positional, arg)
		}
	}
	if len(positional) != 1 {
		return opts, "edit: requires exactly one PATH"
	}
	pathValue, err := requireAbsolutePath(positional[0])
	if err != nil {
		return opts, fmt.Sprintf("edit: %v", err)
	}
	opts.path = pathValue
	return opts, ""
}

func isEditFlag(arg string) bool {
	switch arg {
	case "--old", "--new", "--all", "--count", "--json", "--confirm":
		return true
	default:
		return false
	}
}

func renderEditCount(raw, old string, lines []int) string {
	out := []string{fmt.Sprintf("%d", len(lines))}
	seen := make(map[int]struct{}, len(lines))
	content := splitRawLines(raw)
	for _, line := range lines {
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		text := ""
		if line >= 1 && line <= len(content) {
			text = content[line-1]
		}
		out = append(out, fmt.Sprintf("%d:%s", line, text))
	}
	return strings.Join(out, "\n")
}

func pathErrorCode(err error) int {
	if errors.Is(err, contract.ErrUnsupported) {
		return contract.ExitCodeUnsupported
	}
	return contract.ExitCodeGeneral
}
