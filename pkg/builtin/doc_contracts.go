package builtin

import (
	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

type builtinDocContract struct {
	StdinMode        string
	Operands         string
	DefaultOutput    string
	StructuredOutput string
	StructuredFlags  []string
	MutationKind     string
	SuccessOutput    string
	PipeBehavior     string
	ExitCodes        []string
}

var builtinDocContracts = map[string]builtinDocContract{
	CommandLS: {
		StdinMode:        contract.BuiltinStdinNone,
		Operands:         "zero or more PATH targets",
		DefaultOutput:    "path-per-line listing; `-l` uses fixed columns",
		StructuredOutput: "machine-readable listing",
		StructuredFlags:  []string{"--fmt json", "--fmt md"},
		MutationKind:     contract.BuiltinMutationReadOnly,
		SuccessOutput:    contract.BuiltinSuccessContent,
		PipeBehavior:     contract.BuiltinPipeGood,
	},
	CommandTree: {
		StdinMode:     contract.BuiltinStdinNone,
		Operands:      "zero or more PATH targets",
		DefaultOutput: "directory outline",
		MutationKind:  contract.BuiltinMutationReadOnly,
		SuccessOutput: contract.BuiltinSuccessContent,
		PipeBehavior:  contract.BuiltinPipeWeak,
	},
	CommandCd: {
		StdinMode:     contract.BuiltinStdinNone,
		Operands:      "zero or one PATH",
		DefaultOutput: "no output on success",
		MutationKind:  contract.BuiltinMutationMutates,
		SuccessOutput: contract.BuiltinSuccessSilent,
		PipeBehavior:  contract.BuiltinPipeWeak,
	},
	CommandPwd: {
		StdinMode:     contract.BuiltinStdinNone,
		Operands:      "no operands",
		DefaultOutput: "absolute path",
		MutationKind:  contract.BuiltinMutationReadOnly,
		SuccessOutput: contract.BuiltinSuccessContent,
		PipeBehavior:  contract.BuiltinPipeGood,
	},
	CommandEnv: {
		StdinMode:        contract.BuiltinStdinNone,
		Operands:         "zero or one KEY",
		DefaultOutput:    "`KEY=VALUE` lines or one `KEY=VALUE` row",
		StructuredOutput: "environment snapshot",
		StructuredFlags:  []string{"--json", "--split"},
		MutationKind:     contract.BuiltinMutationReadOnly,
		SuccessOutput:    contract.BuiltinSuccessContent,
		PipeBehavior:     contract.BuiltinPipeGood,
	},
	CommandFrontmatter: {
		StdinMode:        contract.BuiltinStdinNone,
		Operands:         "subcommand plus PATH operands",
		DefaultOutput:    "subcommand-specific frontmatter summary or extracted content",
		StructuredOutput: "frontmatter summary records",
		StructuredFlags:  []string{"stat --fmt json", "stat --fmt md"},
		MutationKind:     contract.BuiltinMutationReadOnly,
		SuccessOutput:    contract.BuiltinSuccessContent,
		PipeBehavior:     contract.BuiltinPipeGood,
	},
	CommandJSON: {
		StdinMode:        contract.BuiltinStdinNone,
		Operands:         "subcommand plus PATH operands",
		DefaultOutput:    "json stat rows, query rows, or extracted JSON subtree",
		StructuredOutput: "JSON shape, query rows, or subtree result",
		StructuredFlags:  []string{"stat --fmt json", "stat --fmt md", "get --fmt json", "get --fmt jsonl", "keys --fmt json", "keys --fmt jsonl", "len --fmt json", "len --fmt jsonl"},
		MutationKind:     contract.BuiltinMutationReadOnly,
		SuccessOutput:    contract.BuiltinSuccessContent,
		PipeBehavior:     contract.BuiltinPipeGood,
	},
	CommandCat: {
		StdinMode:     contract.BuiltinStdinOptional,
		Operands:      "zero or one PATH",
		DefaultOutput: "raw text",
		MutationKind:  contract.BuiltinMutationReadOnly,
		SuccessOutput: contract.BuiltinSuccessContent,
		PipeBehavior:  contract.BuiltinPipeStrong,
	},
	CommandHead: {
		StdinMode:     contract.BuiltinStdinOptional,
		Operands:      "optional PATH",
		DefaultOutput: "first N lines of text",
		MutationKind:  contract.BuiltinMutationReadOnly,
		SuccessOutput: contract.BuiltinSuccessContent,
		PipeBehavior:  contract.BuiltinPipeStrong,
	},
	CommandTail: {
		StdinMode:     contract.BuiltinStdinOptional,
		Operands:      "optional PATH",
		DefaultOutput: "last N lines of text",
		MutationKind:  contract.BuiltinMutationReadOnly,
		SuccessOutput: contract.BuiltinSuccessContent,
		PipeBehavior:  contract.BuiltinPipeStrong,
	},
	CommandGrep: {
		StdinMode:        contract.BuiltinStdinOptional,
		Operands:         "PATTERN plus optional PATH",
		DefaultOutput:    "matched lines with optional file and line prefixes",
		StructuredOutput: "flat match/context/file records",
		StructuredFlags:  []string{"--fmt jsonl"},
		MutationKind:     contract.BuiltinMutationReadOnly,
		SuccessOutput:    contract.BuiltinSuccessContent,
		PipeBehavior:     contract.BuiltinPipeStrong,
		ExitCodes: []string{
			"non-zero when no matches are found",
		},
	},
	CommandRG: {
		StdinMode:        contract.BuiltinStdinOptional,
		Operands:         "PATTERN plus zero or more PATH targets, or `--files` with optional PATH targets",
		DefaultOutput:    "recursive match rows prefixed with path and line number, or path-per-line file lists",
		StructuredOutput: "flat match/context/file records",
		StructuredFlags:  []string{"--fmt jsonl"},
		MutationKind:     contract.BuiltinMutationReadOnly,
		SuccessOutput:    contract.BuiltinSuccessContent,
		PipeBehavior:     contract.BuiltinPipeStrong,
		ExitCodes: []string{
			"non-zero when no matches are found",
		},
	},
	CommandFind: {
		StdinMode:        contract.BuiltinStdinNone,
		Operands:         "optional DIR plus expression flags",
		DefaultOutput:    "matched paths, one per line",
		StructuredOutput: "flat path records",
		StructuredFlags:  []string{"--fmt jsonl"},
		MutationKind:     contract.BuiltinMutationReadOnly,
		SuccessOutput:    contract.BuiltinSuccessContent,
		PipeBehavior:     contract.BuiltinPipeGood,
	},
	CommandWhich: {
		StdinMode:        contract.BuiltinStdinNone,
		Operands:         "one or more COMMAND references",
		DefaultOutput:    "resolved path per line",
		StructuredOutput: "lookup summary",
		StructuredFlags:  []string{"--fmt json"},
		MutationKind:     contract.BuiltinMutationReadOnly,
		SuccessOutput:    contract.BuiltinSuccessContent,
		PipeBehavior:     contract.BuiltinPipeGood,
	},
	CommandType: {
		StdinMode:        contract.BuiltinStdinNone,
		Operands:         "one or more COMMAND references",
		DefaultOutput:    "name kind target rows",
		StructuredOutput: "command-resolution records",
		StructuredFlags:  []string{"--json"},
		MutationKind:     contract.BuiltinMutationReadOnly,
		SuccessOutput:    contract.BuiltinSuccessContent,
		PipeBehavior:     contract.BuiltinPipeGood,
	},
	CommandEcho: {
		StdinMode:     contract.BuiltinStdinNone,
		Operands:      "zero or more text arguments",
		DefaultOutput: "joined text",
		MutationKind:  contract.BuiltinMutationReadOnly,
		SuccessOutput: contract.BuiltinSuccessContent,
		PipeBehavior:  contract.BuiltinPipeStrong,
	},
	CommandTee: {
		StdinMode:        contract.BuiltinStdinRequired,
		Operands:         "exactly one PATH",
		DefaultOutput:    "stdin passthrough",
		StructuredOutput: "write summary",
		StructuredFlags:  []string{"--confirm", "--json"},
		MutationKind:     contract.BuiltinMutationMutates,
		SuccessOutput:    contract.BuiltinSuccessContent,
		PipeBehavior:     contract.BuiltinPipeStrong,
	},
	CommandSed: {
		StdinMode:        contract.BuiltinStdinOptional,
		Operands:         "expression plus optional PATH depending on mode",
		DefaultOutput:    "selected text lines or silent mutation",
		StructuredOutput: "in-place edit summary",
		StructuredFlags:  []string{"-i --json"},
		MutationKind:     contract.BuiltinMutationMutates,
		SuccessOutput:    contract.BuiltinSuccessContent,
		PipeBehavior:     contract.BuiltinPipeStrong,
	},
	CommandMan: {
		StdinMode:        contract.BuiltinStdinNone,
		Operands:         "command name or `--list`",
		DefaultOutput:    "summary documentation",
		StructuredOutput: "machine-readable command catalog",
		StructuredFlags:  []string{"--list --fmt json"},
		MutationKind:     contract.BuiltinMutationReadOnly,
		SuccessOutput:    contract.BuiltinSuccessContent,
		PipeBehavior:     contract.BuiltinPipeGood,
	},
	CommandDate: {
		StdinMode:     contract.BuiltinStdinNone,
		Operands:      "optional `+FORMAT`",
		DefaultOutput: "formatted date/time text",
		MutationKind:  contract.BuiltinMutationReadOnly,
		SuccessOutput: contract.BuiltinSuccessContent,
		PipeBehavior:  contract.BuiltinPipeGood,
	},
	CommandMkdir: {
		StdinMode:        contract.BuiltinStdinNone,
		Operands:         "one or more PATH operands",
		DefaultOutput:    "no output on success",
		StructuredOutput: "path status entries",
		StructuredFlags:  []string{"--confirm", "--json"},
		MutationKind:     contract.BuiltinMutationMutates,
		SuccessOutput:    contract.BuiltinSuccessSilent,
		PipeBehavior:     contract.BuiltinPipeWeak,
	},
	CommandCp: {
		StdinMode:        contract.BuiltinStdinNone,
		Operands:         "SRC_PATH DEST_PATH",
		DefaultOutput:    "no output on success",
		StructuredOutput: "copy summary",
		StructuredFlags:  []string{"--confirm", "--json"},
		MutationKind:     contract.BuiltinMutationMutates,
		SuccessOutput:    contract.BuiltinSuccessSilent,
		PipeBehavior:     contract.BuiltinPipeWeak,
	},
	CommandMv: {
		StdinMode:        contract.BuiltinStdinNone,
		Operands:         "SRC_PATH DEST_PATH",
		DefaultOutput:    "no output on success",
		StructuredOutput: "move summary",
		StructuredFlags:  []string{"--confirm", "--json"},
		MutationKind:     contract.BuiltinMutationMutates,
		SuccessOutput:    contract.BuiltinSuccessSilent,
		PipeBehavior:     contract.BuiltinPipeWeak,
	},
	CommandRm: {
		StdinMode:        contract.BuiltinStdinNone,
		Operands:         "one or more PATH operands",
		DefaultOutput:    "no output on success",
		StructuredOutput: "path status entries",
		StructuredFlags:  []string{"--confirm", "--json"},
		MutationKind:     contract.BuiltinMutationMutates,
		SuccessOutput:    contract.BuiltinSuccessSilent,
		PipeBehavior:     contract.BuiltinPipeWeak,
	},
	CommandRmdir: {
		StdinMode:        contract.BuiltinStdinNone,
		Operands:         "one or more PATH operands",
		DefaultOutput:    "no output on success",
		StructuredOutput: "path status entries",
		StructuredFlags:  []string{"--confirm", "--json"},
		MutationKind:     contract.BuiltinMutationMutates,
		SuccessOutput:    contract.BuiltinSuccessSilent,
		PipeBehavior:     contract.BuiltinPipeWeak,
	},
	CommandTouch: {
		StdinMode:        contract.BuiltinStdinNone,
		Operands:         "one or more PATH operands",
		DefaultOutput:    "no output on success",
		StructuredOutput: "path status entries",
		StructuredFlags:  []string{"--json"},
		MutationKind:     contract.BuiltinMutationMutates,
		SuccessOutput:    contract.BuiltinSuccessSilent,
		PipeBehavior:     contract.BuiltinPipeWeak,
	},
	CommandWc: {
		StdinMode:        contract.BuiltinStdinOptional,
		Operands:         "optional PATH",
		DefaultOutput:    "bare number for one metric; labeled compact text for multiple metrics",
		StructuredOutput: "count summary object",
		StructuredFlags:  []string{"--json"},
		MutationKind:     contract.BuiltinMutationReadOnly,
		SuccessOutput:    contract.BuiltinSuccessContent,
		PipeBehavior:     contract.BuiltinPipeStrong,
	},
	CommandSort: {
		StdinMode:     contract.BuiltinStdinOptional,
		Operands:      "optional PATH",
		DefaultOutput: "sorted text lines",
		MutationKind:  contract.BuiltinMutationReadOnly,
		SuccessOutput: contract.BuiltinSuccessContent,
		PipeBehavior:  contract.BuiltinPipeStrong,
	},
	CommandUniq: {
		StdinMode:     contract.BuiltinStdinOptional,
		Operands:      "optional PATH",
		DefaultOutput: "deduplicated text lines",
		MutationKind:  contract.BuiltinMutationReadOnly,
		SuccessOutput: contract.BuiltinSuccessContent,
		PipeBehavior:  contract.BuiltinPipeStrong,
	},
	CommandDiff: {
		StdinMode:     contract.BuiltinStdinNone,
		Operands:      "PATH1 PATH2",
		DefaultOutput: "unified diff",
		MutationKind:  contract.BuiltinMutationReadOnly,
		SuccessOutput: contract.BuiltinSuccessContent,
		PipeBehavior:  contract.BuiltinPipeGood,
		ExitCodes: []string{
			"`0` when files are identical",
			"non-zero when files differ or usage/runtime errors occur",
		},
	},
	CommandEdit: {
		StdinMode:        contract.BuiltinStdinNone,
		Operands:         "PATH plus --old and usually --new",
		DefaultOutput:    "no output on success",
		StructuredOutput: "edit summary object",
		StructuredFlags:  []string{"--json", "--confirm"},
		MutationKind:     contract.BuiltinMutationMutates,
		SuccessOutput:    contract.BuiltinSuccessSilent,
		PipeBehavior:     contract.BuiltinPipeWeak,
		ExitCodes: []string{
			"non-zero when OLD is missing or matches more than once without --all",
		},
	},
	CommandGlob: {
		StdinMode:        contract.BuiltinStdinNone,
		Operands:         "PATTERN plus zero or more PATH roots",
		DefaultOutput:    "matched paths, one per line",
		StructuredOutput: "flat path records",
		StructuredFlags:  []string{"--fmt jsonl"},
		MutationKind:     contract.BuiltinMutationReadOnly,
		SuccessOutput:    contract.BuiltinSuccessContent,
		PipeBehavior:     contract.BuiltinPipeGood,
	},
	CommandView: {
		StdinMode:        contract.BuiltinStdinNone,
		Operands:         "exactly one PATH",
		DefaultOutput:    "numbered line window",
		StructuredOutput: "numbered line records",
		StructuredFlags:  []string{"--fmt jsonl"},
		MutationKind:     contract.BuiltinMutationReadOnly,
		SuccessOutput:    contract.BuiltinSuccessContent,
		PipeBehavior:     contract.BuiltinPipeGood,
	},
	CommandDirname: {
		StdinMode:     contract.BuiltinStdinNone,
		Operands:      "exactly one PATH",
		DefaultOutput: "parent path",
		MutationKind:  contract.BuiltinMutationReadOnly,
		SuccessOutput: contract.BuiltinSuccessContent,
		PipeBehavior:  contract.BuiltinPipeGood,
	},
	CommandBasename: {
		StdinMode:     contract.BuiltinStdinNone,
		Operands:      "exactly one PATH",
		DefaultOutput: "final path component",
		MutationKind:  contract.BuiltinMutationReadOnly,
		SuccessOutput: contract.BuiltinSuccessContent,
		PipeBehavior:  contract.BuiltinPipeGood,
	},
}

func applyCommandDocContract(spec engine.CommandSpec) engine.CommandSpec {
	contractSpec, ok := builtinDocContracts[spec.Name]
	if !ok {
		return spec
	}
	if spec.StdinMode == "" {
		spec.StdinMode = contractSpec.StdinMode
	}
	if spec.Operands == "" {
		spec.Operands = contractSpec.Operands
	}
	if spec.DefaultOutput == "" {
		spec.DefaultOutput = contractSpec.DefaultOutput
	}
	if spec.StructuredOutput == "" {
		spec.StructuredOutput = contractSpec.StructuredOutput
	}
	if len(spec.StructuredFlags) == 0 {
		spec.StructuredFlags = append([]string(nil), contractSpec.StructuredFlags...)
	}
	if spec.MutationKind == "" {
		spec.MutationKind = contractSpec.MutationKind
	}
	if spec.SuccessOutput == "" {
		spec.SuccessOutput = contractSpec.SuccessOutput
	}
	if spec.PipeBehavior == "" {
		spec.PipeBehavior = contractSpec.PipeBehavior
	}
	if len(spec.ExitCodes) == 0 {
		spec.ExitCodes = append([]string(nil), contractSpec.ExitCodes...)
	}
	return spec
}
