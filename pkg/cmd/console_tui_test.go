package cmd

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type fakeConsoleExecutor struct {
	calls  []string
	output string
	code   int
	cwd    string
	ctxs   []context.Context
}

func (f *fakeConsoleExecutor) Execute(ctx context.Context, commandLine string) (string, int) {
	f.ctxs = append(f.ctxs, ctx)
	f.calls = append(f.calls, commandLine)
	return f.output, f.code
}

func (f *fakeConsoleExecutor) WorkingDir() string {
	if strings.TrimSpace(f.cwd) == "" {
		return "/"
	}
	return f.cwd
}

func transcriptText(model consoleModel) string {
	parts := make([]string, 0, len(model.lines))
	for _, line := range model.lines {
		parts = append(parts, line.Text)
	}
	return strings.Join(parts, "\n")
}

func TestNewConsoleModelViewAndFallbacks(t *testing.T) {
	env := &fakeConsoleExecutor{cwd: "/task_outputs"}
	model := newConsoleModel(context.Background(), env, ConsoleOptions{
		Profile:   " bash-plus ",
		Policy:    " full ",
		HostRoot:  " ",
		SessionID: " ",
		Mounts:    []string{"test", "docs"},
	})

	if got := transcriptText(model); !strings.Contains(got, "human operator console") {
		t.Fatalf("transcript = %q, want operator banner", got)
	}
	if model.View() != "loading..." {
		t.Fatalf("View() before resize = %q, want loading placeholder", model.View())
	}

	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 100, Height: 26})
	if cmd != nil {
		t.Fatalf("WindowSizeMsg returned unexpected cmd")
	}
	model = updated.(consoleModel)

	view := model.View()
	for _, needle := range []string{
		"simsh",
		"idle",
		"session=ephemeral",
		"cwd=/task_outputs",
		"root=(auto)",
		"bash-plus",
		"full",
		"mounts=test,docs",
		"╭",
		"╰",
		"❯",
		"Type a command",
		"/task_outputs",
	} {
		if !strings.Contains(view, needle) {
			t.Fatalf("View() missing %q in %q", needle, view)
		}
	}
}

func TestRoundedComposerBoxTitleAndPadding(t *testing.T) {
	got := roundedComposerBox("/task_outputs", "❯ ls", 24, lipgloss.Color("#C084FC"))
	if !strings.Contains(got, "╭─") || !strings.Contains(got, "/task_outputs") || !strings.Contains(got, "╰") {
		t.Fatalf("composer box = %q, want grok-style rounded frame", got)
	}
	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Fatalf("composer box lines = %d, want 3", len(lines))
	}
	if lipgloss.Width(lines[0]) != lipgloss.Width(lines[1]) || lipgloss.Width(lines[1]) != lipgloss.Width(lines[2]) {
		t.Fatalf("composer box widths = %v", []int{lipgloss.Width(lines[0]), lipgloss.Width(lines[1]), lipgloss.Width(lines[2])})
	}
}

func TestConsoleModelShowsFailedStateAfterNonzeroExit(t *testing.T) {
	env := &fakeConsoleExecutor{output: "boom\n", code: 2, cwd: "/"}
	model := newConsoleModel(context.Background(), env, ConsoleOptions{
		Profile: "core-strict",
		Policy:  "read-only",
	})
	model.now = func() time.Time { return time.Date(2026, 4, 18, 4, 0, 0, 0, time.UTC) }
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 110, Height: 24})
	model = updated.(consoleModel)
	model.input.SetValue("false")

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(consoleModel)
	if cmd == nil {
		t.Fatal("enter returned nil cmd")
	}
	batch, ok := cmd().(tea.BatchMsg)
	if !ok || len(batch) == 0 {
		t.Fatalf("cmd() = %T, want tea.BatchMsg", cmd())
	}
	result, ok := batch[0]().(executeResultMsg)
	if !ok {
		t.Fatalf("first batch cmd type = %T, want executeResultMsg", batch[0]())
	}
	updated, _ = model.Update(result)
	model = updated.(consoleModel)
	view := model.View()
	for _, needle := range []string{"failed", "last=false", "exit=2"} {
		if !strings.Contains(view, needle) {
			t.Fatalf("View() missing %q in %q", needle, view)
		}
	}
	if !strings.Contains(transcriptText(model), "exit 2") {
		t.Fatalf("transcript missing exit marker: %q", transcriptText(model))
	}
}

func TestConsoleModelIgnoresBlankAndQuitsOnExit(t *testing.T) {
	env := &fakeConsoleExecutor{}
	model := newConsoleModel(context.Background(), env, ConsoleOptions{})
	before := transcriptText(model)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("blank enter returned unexpected cmd")
	}
	model = updated.(consoleModel)
	if model.running {
		t.Fatalf("blank enter set running=true")
	}
	if transcriptText(model) != before {
		t.Fatalf("blank enter changed transcript = %q, want %q", transcriptText(model), before)
	}

	for _, input := range []string{"exit", "quit"} {
		model.input.SetValue(input)
		updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Fatalf("%q enter returned nil cmd, want quit cmd", input)
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatalf("%q enter cmd did not return tea.QuitMsg", input)
		}
		model = updated.(consoleModel)
	}

	_, cmd = model.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	if cmd == nil {
		t.Fatal("ctrl+d returned nil cmd, want quit cmd")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("ctrl+d cmd did not return tea.QuitMsg")
	}
}

func TestConsoleModelCommandLifecycleAndClear(t *testing.T) {
	env := &fakeConsoleExecutor{output: "first line\nsecond line\n", code: 17}
	model := newConsoleModel(context.Background(), env, ConsoleOptions{})
	model.now = func() time.Time { return time.Date(2026, 4, 18, 4, 0, 0, 0, time.UTC) }

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 88, Height: 24})
	model = updated.(consoleModel)
	model.input.SetValue("echo demo")

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("enter returned nil cmd for executable input")
	}
	model = updated.(consoleModel)
	if !model.running {
		t.Fatalf("running = false, want true after enter")
	}
	if model.input.Value() != "" {
		t.Fatalf("input value = %q, want cleared input", model.input.Value())
	}
	if len(env.calls) != 0 {
		t.Fatalf("executor called before command dispatch: %v", env.calls)
	}

	batch, ok := cmd().(tea.BatchMsg)
	if !ok {
		t.Fatalf("cmd() type = %T, want tea.BatchMsg", cmd())
	}
	if len(batch) != 2 {
		t.Fatalf("batch len = %d, want 2", len(batch))
	}
	result, ok := batch[0]().(executeResultMsg)
	if !ok {
		t.Fatalf("first batch cmd type = %T, want executeResultMsg", batch[0]())
	}
	if !reflect.DeepEqual(env.calls, []string{"echo demo"}) {
		t.Fatalf("executor calls = %v, want [echo demo]", env.calls)
	}

	updated, cmd = model.Update(result)
	if cmd != nil {
		t.Fatalf("executeResultMsg returned unexpected cmd")
	}
	model = updated.(consoleModel)
	if model.running {
		t.Fatalf("running = true after result, want false")
	}
	if model.commandCount != 1 {
		t.Fatalf("commandCount = %d, want 1", model.commandCount)
	}
	if model.lastExitCode != 17 {
		t.Fatalf("lastExitCode = %d, want 17", model.lastExitCode)
	}

	transcript := transcriptText(model)
	for _, needle := range []string{"echo demo", "first line", "second line", "exit 17"} {
		if !strings.Contains(transcript, needle) {
			t.Fatalf("transcript missing %q in %q", needle, transcript)
		}
	}

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyCtrlL})
	if cmd != nil {
		t.Fatalf("ctrl+l returned unexpected cmd")
	}
	model = updated.(consoleModel)
	if len(model.lines) != 0 {
		t.Fatalf("lines len = %d, want 0 after ctrl+l", len(model.lines))
	}
}

func TestConsoleModelCancelKeepsConsoleOpen(t *testing.T) {
	env := &fakeConsoleExecutor{}
	model := newConsoleModel(context.Background(), env, ConsoleOptions{})
	model.input.SetValue("echo demo")

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter returned nil cmd for executable input")
	}
	model = updated.(consoleModel)
	batch, ok := cmd().(tea.BatchMsg)
	if !ok || len(batch) == 0 {
		t.Fatalf("cmd() = %T, want non-empty tea.BatchMsg", cmd())
	}
	result, ok := batch[0]().(executeResultMsg)
	if !ok {
		t.Fatalf("first batch cmd type = %T, want executeResultMsg", batch[0]())
	}
	if len(env.ctxs) != 1 {
		t.Fatalf("executor ctx count = %d, want 1", len(env.ctxs))
	}
	if err := env.ctxs[0].Err(); err != nil {
		t.Fatalf("execution ctx err before cancel = %v, want nil", err)
	}

	updated, cancelCmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cancelCmd != nil {
		t.Fatal("ctrl+c while running returned quit cmd, want stay in console")
	}
	model = updated.(consoleModel)
	if !model.running {
		t.Fatal("running = false while canceled execution result is pending")
	}
	if model.cancelRunning != nil {
		t.Fatal("cancelRunning not cleared after cancel")
	}
	if !model.cancelRequested {
		t.Fatal("cancelRequested = false after ctrl+c")
	}
	if model.activeRunID != result.RunID {
		t.Fatalf("activeRunID = %d, want pending run %d", model.activeRunID, result.RunID)
	}
	if !strings.Contains(transcriptText(model), "cancel requested") {
		t.Fatalf("transcript missing cancel-request marker: %q", transcriptText(model))
	}

	deadline := time.After(time.Second)
	for env.ctxs[0].Err() == nil {
		select {
		case <-deadline:
			t.Fatal("execution ctx was not canceled")
		default:
			time.Sleep(time.Millisecond)
		}
	}

	updated, _ = model.Update(result)
	model = updated.(consoleModel)
	if model.running || model.cancelRequested || model.activeRunID != 0 {
		t.Fatalf("execution state after result = running:%v cancel:%v run:%d", model.running, model.cancelRequested, model.activeRunID)
	}
}

func TestConsoleModelWaitsForCanceledResultBeforeNextCommand(t *testing.T) {
	env := &fakeConsoleExecutor{}
	model := newConsoleModel(context.Background(), env, ConsoleOptions{})
	model.input.SetValue("echo demo")

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter returned nil cmd for executable input")
	}
	model = updated.(consoleModel)
	batch, ok := cmd().(tea.BatchMsg)
	if !ok || len(batch) == 0 {
		t.Fatalf("cmd() = %T, want non-empty tea.BatchMsg", cmd())
	}
	result, ok := batch[0]().(executeResultMsg)
	if !ok {
		t.Fatalf("first batch cmd type = %T, want executeResultMsg", batch[0]())
	}
	if result.RunID == 0 {
		t.Fatalf("execute result run id = %d, want non-zero", result.RunID)
	}

	updated, cancelCmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cancelCmd != nil {
		t.Fatal("ctrl+c while running returned quit cmd")
	}
	model = updated.(consoleModel)
	if !model.running || !model.cancelRequested {
		t.Fatalf("cancel state = running:%v requested:%v", model.running, model.cancelRequested)
	}
	if model.activeRunID != result.RunID {
		t.Fatalf("activeRunID = %d after cancel, want %d", model.activeRunID, result.RunID)
	}

	model.input.SetValue("echo next")
	updated, nextCmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(consoleModel)
	if nextCmd != nil {
		t.Fatal("enter started a second command while canceled result was pending")
	}
	if len(env.calls) != 1 {
		t.Fatalf("executor calls = %v, want only first command", env.calls)
	}

	env.cwd = "/after-cancel"
	updated, cmd = model.Update(result)
	model = updated.(consoleModel)
	if cmd != nil {
		t.Fatalf("executeResultMsg returned unexpected cmd")
	}
	if model.running || model.cancelRequested || model.activeRunID != 0 {
		t.Fatalf("execution state after result = running:%v cancel:%v run:%d", model.running, model.cancelRequested, model.activeRunID)
	}
	if model.commandCount != 1 || model.cwd != "/after-cancel" {
		t.Fatalf("result reconciliation = count:%d cwd:%q", model.commandCount, model.cwd)
	}
}

func TestConsoleModelCancelTakesPriorityOverHelp(t *testing.T) {
	env := &fakeConsoleExecutor{}
	model := newConsoleModel(context.Background(), env, ConsoleOptions{})
	model.input.SetValue("echo demo")

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter returned nil cmd")
	}
	model = updated.(consoleModel)
	model.showHelp = true

	updated, cancelCmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cancelCmd != nil {
		t.Fatal("ctrl+c returned quit command while execution was running")
	}
	model = updated.(consoleModel)
	if model.showHelp || !model.cancelRequested || !model.running {
		t.Fatalf("help cancel state = help:%v requested:%v running:%v", model.showHelp, model.cancelRequested, model.running)
	}
}

func TestConsoleModelHistoryAndHelp(t *testing.T) {
	env := &fakeConsoleExecutor{output: "ok\n", code: 0}
	model := newConsoleModel(context.Background(), env, ConsoleOptions{})
	model.now = func() time.Time { return time.Date(2026, 4, 18, 4, 0, 0, 0, time.UTC) }
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	model = updated.(consoleModel)
	model.input.SetValue("ls -l")

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(consoleModel)
	if cmd == nil {
		t.Fatal("enter returned nil cmd")
	}
	batch := cmd().(tea.BatchMsg)
	updated, _ = model.Update(batch[0]().(executeResultMsg))
	model = updated.(consoleModel)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(consoleModel)
	if got := model.input.Value(); got != "ls -l" {
		t.Fatalf("history up = %q, want ls -l", got)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(consoleModel)
	if got := model.input.Value(); got != "" {
		t.Fatalf("history down = %q, want empty draft", got)
	}

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if cmd != nil {
		t.Fatalf("? returned unexpected cmd: %T", cmd)
	}
	model = updated.(consoleModel)
	if !model.showHelp {
		t.Fatal("showHelp = false after ?, want true")
	}
	if !strings.Contains(model.View(), "command history") {
		t.Fatalf("help overlay missing history hint: %q", model.View())
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(consoleModel)
	if model.showHelp {
		t.Fatal("showHelp stayed true after esc")
	}
}

func TestDrainHelpersDoNotBlockOnUnreadPipe(t *testing.T) {
	// Startup used to Read(stdin) to drop OSC replies. On a TTY without a
	// working read deadline that blocks forever. Keep stripping replies from
	// the input model instead, and never drain stdin before the program starts.
	done := make(chan struct{})
	go func() {
		configureOperatorConsoleRenderer()
		_ = stripTerminalReportJunk("]11;rgb:0a0a/0e0e/1414\\")
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("renderer setup blocked")
	}
}

func TestStripTerminalReportJunk(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"]11;rgb:0a0a/0e0e/1414\\", ""},
		{"\x1b]11;rgb:0a0a/0e0e/1414\x1b\\", ""},
		{"ls -l ]11;rgb:ffff/ffff/ffff\\", "ls -l "},
		{"echo hello", "echo hello"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := stripTerminalReportJunk(tc.in); got != tc.want {
			t.Errorf("stripTerminalReportJunk(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestConsoleModelSanitizesColorQueryInput(t *testing.T) {
	env := &fakeConsoleExecutor{}
	model := newConsoleModel(context.Background(), env, ConsoleOptions{})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	model = updated.(consoleModel)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]11;rgb:0a0a/0e0e/1414\\")})
	model = updated.(consoleModel)
	if got := model.input.Value(); got != "" {
		t.Fatalf("input after OSC junk = %q, want empty", got)
	}
}

func TestMiddleElideKeepsSuffix(t *testing.T) {
	got := middleElide("/knowledge_base/experience/agent-runtime", 16)
	if runewidth := len([]rune(got)); !strings.Contains(got, "…") || runewidth == 0 {
		t.Fatalf("middleElide() = %q, want elided path", got)
	}
	if !strings.Contains(got, "runtime") && !strings.HasSuffix(got, "ime") && !strings.Contains(got, "time") {
		t.Fatalf("middleElide() = %q, want trailing path fragment", got)
	}
}
