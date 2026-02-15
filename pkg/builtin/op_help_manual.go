package builtin

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

func specMan() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandMan,
		Manual: "man [-v] [-l|--list] <command>",
		Tips: []string{
			"man CMD shows summary with tips and examples.",
			"man -v CMD shows full documentation.",
			"man --list shows all available commands.",
		},
		Examples:       ExamplesFor("man"),
		DetailedManual: LoadEmbeddedManual("man"),
		Run:            runMan,
	}
}

func runMan(runtime engine.CommandRuntime, args []string) (string, int) {
	verbose := false
	listMode := false
	target := ""

	for _, arg := range args {
		switch arg {
		case "-v":
			verbose = true
		case "-l", "--list":
			listMode = true
		default:
			if strings.HasPrefix(arg, "-") {
				return fmt.Sprintf("man: unsupported flag %s", arg), contract.ExitCodeUsage
			}
			if target != "" {
				return "man: expected at most one command name", contract.ExitCodeUsage
			}
			target = arg
		}
	}

	if listMode {
		return runManList(runtime)
	}

	if target == "" {
		return "man: expected command name (use --list to see all commands)", contract.ExitCodeUsage
	}

	target = strings.TrimSpace(target)
	target = strings.TrimPrefix(target, contract.VirtualSystemBinDir+"/")
	target = strings.TrimPrefix(target, contract.VirtualExternalBinDir+"/")

	// Verbose mode: try detailed manual first.
	if verbose {
		if runtime.LookupDetailedManual != nil {
			if manual, ok := runtime.LookupDetailedManual(target); ok {
				manual = strings.TrimSpace(manual)
				if manual != "" {
					return manual, 0
				}
			}
		}
	}

	// Summary mode (or verbose fallback): use LookupManual.
	if runtime.LookupManual != nil {
		if manual, ok := runtime.LookupManual(target); ok {
			manual = strings.TrimSpace(manual)
			if manual != "" {
				return manual, 0
			}
		}
	}

	// External command manual lookup.
	if runtime.Ops.ReadExternalManual != nil {
		manual, err := runtime.Ops.ReadExternalManual(runtime.Ctx, target)
		if err == nil {
			if strings.TrimSpace(manual) != "" {
				return manual, 0
			}
		} else if !errors.Is(err, contract.ErrUnsupported) {
			return fmt.Sprintf("man: %v", err), contract.ExitCodeGeneral
		}
	}

	// Fallback: check external command list for summary.
	if runtime.Ops.ListExternalCommands != nil {
		cmds, err := runtime.Ops.ListExternalCommands(runtime.Ctx)
		if err != nil && !errors.Is(err, contract.ErrUnsupported) {
			return fmt.Sprintf("man: %v", err), contract.ExitCodeGeneral
		}
		for _, c := range cmds {
			if strings.TrimSpace(c.Name) == target {
				if strings.TrimSpace(c.Summary) != "" {
					return c.Summary, 0
				}
				return fmt.Sprintf("binary: %s", c.Name), 0
			}
		}
	}

	return fmt.Sprintf("man: %s: not found", target), contract.ExitCodeGeneral
}

func runManList(runtime engine.CommandRuntime) (string, int) {
	var sb strings.Builder
	sb.WriteString("Available commands:\n\n")

	// Builtin commands.
	if runtime.ListBuiltinNames != nil {
		names := runtime.ListBuiltinNames()
		sort.Strings(names)
		if len(names) > 0 {
			sb.WriteString("Builtins:\n")
			for _, name := range names {
				synopsis := name
				if runtime.LookupManual != nil {
					// Extract just the first line (the synopsis) from the manual.
					if manual, ok := runtime.LookupManual(name); ok {
						firstLine := strings.SplitN(strings.TrimSpace(manual), "\n", 2)[0]
						synopsis = firstLine
					}
				}
				sb.WriteString(fmt.Sprintf("  %-12s %s\n", name, synopsis))
			}
		}
	}

	// External commands.
	if runtime.Ops.ListExternalCommands != nil {
		cmds, err := runtime.Ops.ListExternalCommands(runtime.Ctx)
		if err == nil && len(cmds) > 0 {
			sb.WriteString("\nExternal:\n")
			for _, c := range cmds {
				summary := strings.TrimSpace(c.Summary)
				if summary == "" {
					summary = "(no description)"
				}
				sb.WriteString(fmt.Sprintf("  %-12s %s\n", c.Name, summary))
			}
		}
	}

	return strings.TrimRight(sb.String(), "\n"), 0
}
