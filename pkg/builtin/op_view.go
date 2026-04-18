package builtin

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

func specView() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandView,
		Manual: "view [--start N] [--lines N] [--fmt jsonl] [--] PATH",
		Tips: []string{
			"view prints a numbered window so agents can inspect a slice instead of dumping the whole file.",
			"Line numbers are 1-based. Default window is 80 lines from the start of the file.",
			"Use --fmt jsonl for {line,text} records, not --json.",
		},
		StructuredOutput: "numbered line records",
		StructuredFlags:  []string{"--fmt jsonl"},
		Examples:         ExamplesFor("view"),
		DetailedManual:   LoadEmbeddedManual("view"),
		Run:              runView,
	}
}

type viewArgs struct {
	path  string
	start int
	lines int
	jsonl bool
}

type viewRecord struct {
	Line int    `json:"line"`
	Text string `json:"text"`
}

func runView(runtime engine.CommandRuntime, args []string) (string, int) {
	opts, errMsg := parseViewArgs(args, runtime.Ops.RequireAbsolutePath)
	if errMsg != "" {
		return errMsg, contract.ExitCodeUsage
	}
	if out, code, ok := preflightPathChecks(runtime, "view", []pathCheck{{
		path:               opts.path,
		op:                 contract.PathOpRead,
		unsupportedMessage: "view: read is not supported",
	}}); !ok {
		return out, code
	}
	raw, err := runtime.Ops.ReadRawContent(runtime.Ctx, opts.path)
	if err != nil {
		return formatCommandPathError("view", opts.path, "view: read is not supported", err), pathErrorCode(err)
	}
	all := splitRawLines(raw)
	total := len(all)
	if opts.start > total {
		return fmt.Sprintf("view: file has %d lines", total), contract.ExitCodeGeneral
	}
	end := opts.start + opts.lines - 1
	if opts.lines == 0 {
		end = opts.start - 1
	}
	if end > total {
		end = total
	}
	window := []string{}
	if end >= opts.start {
		window = all[opts.start-1 : end]
	}
	if opts.jsonl {
		lines := make([]string, 0, len(window)+1)
		meta, marshalErr := json.Marshal(struct {
			Kind  string `json:"kind"`
			Start int    `json:"start"`
			Shown int    `json:"shown"`
			Total int    `json:"total"`
		}{
			Kind:  "window",
			Start: opts.start,
			Shown: len(window),
			Total: total,
		})
		if marshalErr != nil {
			return fmt.Sprintf("view: %v", marshalErr), contract.ExitCodeGeneral
		}
		lines = append(lines, string(meta))
		for i, text := range window {
			encoded, lineErr := json.Marshal(viewRecord{Line: opts.start + i, Text: text})
			if lineErr != nil {
				return fmt.Sprintf("view: %v", lineErr), contract.ExitCodeGeneral
			}
			lines = append(lines, string(encoded))
		}
		return strings.Join(lines, "\n"), 0
	}
	out := make([]string, 0, len(window)+1)
	for i, text := range window {
		out = append(out, strconv.Itoa(opts.start+i)+":"+text)
	}
	out = append(out, fmt.Sprintf("shown %d/%d from %d", len(window), total, opts.start))
	return strings.Join(out, "\n"), 0
}

func parseViewArgs(args []string, requireAbsolutePath func(string) (string, error)) (viewArgs, string) {
	opts := viewArgs{start: 1, lines: 80}
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
		switch arg {
		case "--start", "-s":
			if idx+1 >= len(args) {
				return opts, "view: --start requires a positive line number"
			}
			idx++
			start, err := parsePositiveLine(args[idx])
			if err != nil {
				return opts, "view: --start must be a positive integer"
			}
			opts.start = start
		case "--lines", "-n":
			if idx+1 >= len(args) {
				return opts, "view: --lines requires a non-negative count"
			}
			idx++
			count, err := parseNonNegativeInt(args[idx])
			if err != nil {
				return opts, "view: --lines must be a non-negative integer"
			}
			opts.lines = count
		case "--fmt":
			if idx+1 >= len(args) {
				return opts, "view: --fmt requires jsonl"
			}
			idx++
			if args[idx] != "jsonl" {
				return opts, fmt.Sprintf("view: unsupported --fmt value %q", args[idx])
			}
			opts.jsonl = true
		case "--fmt=jsonl":
			opts.jsonl = true
		case "--json":
			return opts, "view: unsupported flag --json; use --fmt jsonl"
		default:
			if strings.HasPrefix(arg, "--") {
				return opts, fmt.Sprintf("view: unsupported flag %s", arg)
			}
			positional = append(positional, arg)
		}
	}
	if len(positional) != 1 {
		return opts, "view: requires exactly one PATH"
	}
	pathValue, err := requireAbsolutePath(positional[0])
	if err != nil {
		return opts, fmt.Sprintf("view: %v", err)
	}
	opts.path = pathValue
	return opts, ""
}

func parsePositiveLine(raw string) (int, error) {
	parsed, err := parseNonNegativeInt(raw)
	if err != nil || parsed < 1 {
		return 0, fmt.Errorf("line must be positive")
	}
	return parsed, nil
}
