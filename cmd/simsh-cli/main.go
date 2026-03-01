package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	runtimecmd "github.com/khicago/simsh/pkg/cmd"
	"github.com/khicago/simsh/pkg/contract"
	runtimeengine "github.com/khicago/simsh/pkg/engine/runtime"
	"github.com/khicago/simsh/pkg/service/httpapi"
	"github.com/mattn/go-isatty"
)

type cliMode string

const (
	modeRun   cliMode = "run"
	modeServe cliMode = "serve"
)

type cliOptions struct {
	mode cliMode

	command    string
	lineREPL   bool
	disableTUI bool
	rootDir    string
	policy     string
	profile    string
	rcFiles    []string
	mounts     []string
	listenAddr string
	port       int
}

func main() {
	opts, err := parseCLIOptions(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(contract.ExitCodeUsage)
	}
	code := runCLI(context.Background(), opts, os.Stdin, os.Stdout, os.Stderr)
	os.Exit(code)
}

func parseCLIOptions(args []string) (cliOptions, error) {
	if len(args) > 0 && strings.TrimSpace(args[0]) == string(modeServe) {
		return parseServeOptions(args[1:])
	}
	return parseRunOptions(args)
}

func parseRunOptions(args []string) (cliOptions, error) {
	fs := flag.NewFlagSet("simsh-cli", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	command := fs.String("c", "", "run one command and exit")
	lineREPL := fs.Bool("repl", false, "start line-based REPL")
	disableTUI := fs.Bool("no-tui", false, "disable TUI and use line REPL")
	rootDir := fs.String("root", "", "host root directory for runtime zones")
	policy := fs.String("policy", string(contract.WriteModeReadOnly), "execution policy: disabled|read-only|write-limited|full")
	profile := fs.String("profile", string(contract.ProfileCoreStrict), "compatibility profile: core-strict|bash-plus|zsh-lite")
	rcList := fs.String("rc", "", "comma-separated rc files (absolute virtual paths)")
	mountList := fs.String("mount", "", "comma-separated mounts (test)")
	if err := fs.Parse(args); err != nil {
		return cliOptions{}, err
	}

	cmdline := strings.TrimSpace(*command)
	if cmdline == "" && fs.NArg() > 0 {
		cmdline = strings.TrimSpace(strings.Join(fs.Args(), " "))
	}
	if cmdline != "" && *lineREPL {
		return cliOptions{}, fmt.Errorf("cannot use -c with --repl")
	}
	if _, err := contract.PolicyPreset(*policy); err != nil {
		return cliOptions{}, err
	}
	if _, err := contract.ParseProfile(*profile); err != nil {
		return cliOptions{}, err
	}
	mounts, err := parseMountList(*mountList)
	if err != nil {
		return cliOptions{}, err
	}
	rcFiles := parseCSVList(*rcList)

	return cliOptions{
		mode:       modeRun,
		command:    cmdline,
		lineREPL:   *lineREPL,
		disableTUI: *disableTUI,
		rootDir:    strings.TrimSpace(*rootDir),
		policy:     strings.TrimSpace(*policy),
		profile:    strings.TrimSpace(*profile),
		rcFiles:    rcFiles,
		mounts:     mounts,
	}, nil
}

func parseServeOptions(args []string) (cliOptions, error) {
	fs := flag.NewFlagSet("simsh-cli serve", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	rootDir := fs.String("root", "", "host root directory for runtime zones")
	policy := fs.String("policy", string(contract.WriteModeReadOnly), "execution policy: disabled|read-only|write-limited|full")
	profile := fs.String("profile", string(contract.ProfileCoreStrict), "compatibility profile: core-strict|bash-plus|zsh-lite")
	rcList := fs.String("rc", "", "comma-separated rc files (absolute virtual paths)")
	mountList := fs.String("mount", "", "comma-separated mounts (test)")
	listenAddr := fs.String("listen", "", "http listen address")
	port := fs.Int("P", 18080, "port for web runtime service")
	if err := fs.Parse(args); err != nil {
		return cliOptions{}, err
	}
	if fs.NArg() > 0 {
		return cliOptions{}, fmt.Errorf("serve: unexpected positional args: %s", strings.Join(fs.Args(), " "))
	}
	if _, err := contract.PolicyPreset(*policy); err != nil {
		return cliOptions{}, err
	}
	if _, err := contract.ParseProfile(*profile); err != nil {
		return cliOptions{}, err
	}
	mounts, err := parseMountList(*mountList)
	if err != nil {
		return cliOptions{}, err
	}
	rcFiles := parseCSVList(*rcList)

	return cliOptions{
		mode:       modeServe,
		rootDir:    strings.TrimSpace(*rootDir),
		policy:     strings.TrimSpace(*policy),
		profile:    strings.TrimSpace(*profile),
		rcFiles:    rcFiles,
		mounts:     mounts,
		listenAddr: strings.TrimSpace(*listenAddr),
		port:       *port,
	}, nil
}

func parseMountList(raw string) ([]string, error) {
	items := parseCSVList(raw)
	seen := map[string]struct{}{}
	mounts := make([]string, 0)
	for _, name := range items {
		if name != "test" {
			return nil, fmt.Errorf("unsupported mount %q", name)
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		mounts = append(mounts, name)
	}
	sort.Strings(mounts)
	return mounts, nil
}

func parseCSVList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	seen := map[string]struct{}{}
	items := make([]string, 0)
	for _, item := range strings.Split(raw, ",") {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		items = append(items, value)
	}
	sort.Strings(items)
	return items
}

func runCLI(ctx context.Context, opts cliOptions, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	switch opts.mode {
	case modeServe:
		return runServe(opts, stdout, stderr)
	case modeRun:
		return runRunMode(ctx, opts, stdin, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unsupported mode %q\n", opts.mode)
		return contract.ExitCodeUsage
	}
}

func runRunMode(ctx context.Context, opts cliOptions, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	rootDir := resolveRootDir(opts.rootDir)
	runtimeOpts := runtimeOptionsFromCLI(opts, rootDir)

	if opts.command == "" {
		manager := runtimeengine.NewSessionManager(runtimeengine.SessionManagerOptions{})
		session, err := manager.Create(ctx, runtimeOpts)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return contract.ExitCodeGeneral
		}
		defer func() { _, _ = manager.Close(ctx, session.SessionID) }()
		executor := &sessionExecutor{manager: manager, sessionID: session.SessionID}
		if opts.lineREPL || opts.disableTUI || !isTerminal(stdin, stdout) {
			return runREPL(ctx, opts, rootDir, executor, session.SessionID, stdin, stdout, stderr)
		}
		return runTUI(ctx, opts, rootDir, executor, session.SessionID, stdin, stdout, stderr)
	}

	env, err := runtimeengine.New(runtimeOpts)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return contract.ExitCodeGeneral
	}

	out, code := env.Execute(ctx, opts.command)
	if out != "" {
		fmt.Fprint(stdout, out)
		if !strings.HasSuffix(out, "\n") {
			fmt.Fprint(stdout, "\n")
		}
	}
	return code
}

func runServe(opts cliOptions, stdout io.Writer, stderr io.Writer) int {
	rootDir := resolveRootDir(opts.rootDir)

	listenAddr := strings.TrimSpace(opts.listenAddr)
	if listenAddr == "" {
		if opts.port <= 0 {
			opts.port = 18080
		}
		listenAddr = fmt.Sprintf(":%d", opts.port)
	}

	_, _ = fmt.Fprintf(stdout, "simsh serve listening on %s (root=%s profile=%s policy=%s mounts=%s)\n", listenAddr, rootDir, opts.profile, opts.policy, strings.Join(opts.mounts, ","))
	handler := httpapi.NewHandler(httpapi.Config{
		DefaultHostRoot: rootDir,
		DefaultProfile:  opts.profile,
		DefaultPolicy:   opts.policy,
		DefaultRCFiles:  append([]string(nil), opts.rcFiles...),
		EnableTestMount: contains(opts.mounts, "test"),
	})
	err := http.ListenAndServe(listenAddr, handler)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return contract.ExitCodeGeneral
	}
	return 0
}

func runTUI(ctx context.Context, opts cliOptions, rootDir string, env runtimecmd.Executor, sessionID string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	err := runtimecmd.RunConsoleTUI(ctx, env, runtimecmd.ConsoleOptions{
		Profile:   opts.profile,
		Policy:    opts.policy,
		HostRoot:  rootDir,
		SessionID: sessionID,
		Mounts:    opts.mounts,
		Input:     stdin,
		Output:    stdout,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return contract.ExitCodeGeneral
	}
	return 0
}

func runREPL(ctx context.Context, opts cliOptions, rootDir string, env runtimecmd.Executor, sessionID string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	_, _ = fmt.Fprintf(stdout, "simsh line-repl profile=%s policy=%s host_root=%s session=%s\n", opts.profile, opts.policy, rootDir, sessionID)
	if len(opts.mounts) > 0 {
		_, _ = fmt.Fprintf(stdout, "mounts=%s\n", strings.Join(opts.mounts, ","))
	}
	_, _ = fmt.Fprintln(stdout, "type 'exit' or 'quit' to leave")

	scanner := bufio.NewScanner(stdin)
	for {
		_, _ = fmt.Fprint(stdout, "simsh> ")
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				fmt.Fprintf(stderr, "read input failed: %v\n", err)
				return contract.ExitCodeGeneral
			}
			return 0
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			return 0
		}
		out, code := env.Execute(ctx, line)
		if out != "" {
			fmt.Fprint(stdout, out)
			if !strings.HasSuffix(out, "\n") {
				fmt.Fprint(stdout, "\n")
			}
		}
		if code != 0 {
			fmt.Fprintf(stderr, "[exit %d]\n", code)
		}
	}
}

func runtimeOptionsFromCLI(opts cliOptions, rootDir string) runtimeengine.Options {
	policy, _ := contract.PolicyPreset(opts.policy)
	profile, _ := contract.ParseProfile(opts.profile)
	return runtimeengine.Options{
		HostRoot:         rootDir,
		Profile:          profile,
		Policy:           policy,
		RCFiles:          opts.rcFiles,
		EnableTestCorpus: contains(opts.mounts, "test"),
	}
}

type sessionExecutor struct {
	manager   *runtimeengine.SessionManager
	sessionID string
}

func (s *sessionExecutor) Execute(ctx context.Context, commandLine string) (string, int) {
	if s == nil || s.manager == nil {
		return "execute: session runtime is not initialized", contract.ExitCodeGeneral
	}
	executed, err := s.manager.Execute(ctx, s.sessionID, commandLine, contract.ExecutionPolicy{})
	if err != nil {
		return "execute: " + err.Error(), contract.ExitCodeGeneral
	}
	return executed.Result.FlattenOutput(), executed.Result.ExitCode
}

func resolveRootDir(rootDir string) string {
	root := strings.TrimSpace(rootDir)
	if root != "" {
		return root
	}
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
}

func isTerminal(stdin io.Reader, stdout io.Writer) bool {
	inFile, ok := stdin.(*os.File)
	if !ok {
		return false
	}
	outFile, ok := stdout.(*os.File)
	if !ok {
		return false
	}
	return (isatty.IsTerminal(inFile.Fd()) || isatty.IsCygwinTerminal(inFile.Fd())) &&
		(isatty.IsTerminal(outFile.Fd()) || isatty.IsCygwinTerminal(outFile.Fd()))
}

func contains(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}
