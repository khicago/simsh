package cmd

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type fakeConsoleExecutor struct {
	calls  []string
	output string
	code   int
	ctxs   []context.Context
}

func (f *fakeConsoleExecutor) Execute(ctx context.Context, commandLine string) (string, int) {
	f.ctxs = append(f.ctxs, ctx)
	f.calls = append(f.calls, commandLine)
	return f.output, f.code
}

func TestNewConsoleModelViewAndFallbacks(t *testing.T) {
	env := &fakeConsoleExecutor{}
	model := newConsoleModel(context.Background(), env, ConsoleOptions{
		Profile:   " bash-plus ",
		Policy:    " full ",
		HostRoot:  " ",
		SessionID: " ",
		Mounts:    []string{"test", "docs"},
	})

	if got := model.transcript; !reflect.DeepEqual(got, []string{"simsh interactive console", "type exit or quit to stop"}) {
		t.Fatalf("transcript = %#v, want initial banner", got)
	}
	if model.View() != "loading..." {
		t.Fatalf("View() before resize = %q, want loading placeholder", model.View())
	}

	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 92, Height: 26})
	if cmd != nil {
		t.Fatalf("WindowSizeMsg returned unexpected cmd")
	}
	model = updated.(consoleModel)

	view := model.View()
	for _, needle := range []string{
		"simsh command runtime",
		"root=(auto) session=ephemeral",
		"profile=bash-plus policy=full mounts=test,docs",
	} {
		if !strings.Contains(view, needle) {
			t.Fatalf("View() missing %q in %q", needle, view)
		}
	}
}

func TestConsoleModelIgnoresBlankAndQuitsOnExit(t *testing.T) {
	env := &fakeConsoleExecutor{}
	model := newConsoleModel(context.Background(), env, ConsoleOptions{})
	before := append([]string(nil), model.transcript...)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("blank enter returned unexpected cmd")
	}
	model = updated.(consoleModel)
	if model.running {
		t.Fatalf("blank enter set running=true")
	}
	if !reflect.DeepEqual(model.transcript, before) {
		t.Fatalf("blank enter changed transcript = %#v, want %#v", model.transcript, before)
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
}

func TestConsoleModelCommandLifecycleAndClear(t *testing.T) {
	env := &fakeConsoleExecutor{output: "first line\nsecond line\n", code: 17}
	model := newConsoleModel(context.Background(), env, ConsoleOptions{})

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

	transcript := strings.Join(model.transcript, "\n")
	for _, needle := range []string{"$ echo demo", "  first line", "  second line", "  [exit 17]"} {
		if !strings.Contains(transcript, needle) {
			t.Fatalf("transcript missing %q in %q", needle, transcript)
		}
	}

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyCtrlL})
	if cmd != nil {
		t.Fatalf("ctrl+l returned unexpected cmd")
	}
	model = updated.(consoleModel)
	if len(model.transcript) != 0 {
		t.Fatalf("transcript len = %d, want 0 after ctrl+l", len(model.transcript))
	}
}

func TestConsoleModelQuitCancelsInFlightExecution(t *testing.T) {
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
	_ = batch[0]()
	if len(env.ctxs) != 1 {
		t.Fatalf("executor ctx count = %d, want 1", len(env.ctxs))
	}
	if err := env.ctxs[0].Err(); err != nil {
		t.Fatalf("execution ctx err before quit = %v, want nil", err)
	}

	updated, quitCmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if quitCmd == nil {
		t.Fatal("ctrl+c returned nil cmd, want quit cmd")
	}
	if _, ok := quitCmd().(tea.QuitMsg); !ok {
		t.Fatalf("ctrl+c cmd did not return tea.QuitMsg")
	}
	model = updated.(consoleModel)
	if model.cancelRunning != nil {
		t.Fatal("cancelRunning not cleared after quit")
	}

	deadline := time.After(time.Second)
	for env.ctxs[0].Err() == nil {
		select {
		case <-deadline:
			t.Fatal("execution ctx was not canceled on quit")
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

func TestConsoleModelDropsLateExecuteResultAfterCancel(t *testing.T) {
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

	updated, quitCmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if quitCmd == nil {
		t.Fatal("ctrl+c returned nil cmd, want quit cmd")
	}
	model = updated.(consoleModel)
	if model.running {
		t.Fatal("running = true after cancel, want false")
	}
	if model.activeRunID != 0 {
		t.Fatalf("activeRunID = %d after cancel, want 0", model.activeRunID)
	}

	before := append([]string(nil), model.transcript...)
	updated, cmd = model.Update(result)
	if cmd != nil {
		t.Fatalf("late executeResultMsg returned unexpected cmd")
	}
	model = updated.(consoleModel)
	if model.commandCount != 0 {
		t.Fatalf("commandCount = %d, want 0 after dropping late result", model.commandCount)
	}
	if model.lastExitCode != 0 {
		t.Fatalf("lastExitCode = %d, want 0 after dropping late result", model.lastExitCode)
	}
	if !reflect.DeepEqual(model.transcript, before) {
		t.Fatalf("late result changed transcript = %#v, want %#v", model.transcript, before)
	}
}
