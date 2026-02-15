package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

type ConsoleOptions struct {
	Profile  string
	Policy   string
	HostRoot string
	Mounts   []string
	Input    io.Reader
	Output   io.Writer
}

type executeResultMsg struct {
	Output   string
	ExitCode int
}

type consoleModel struct {
	ctx context.Context
	env *Environment

	profile string
	policy  string
	hostDir string
	mounts  string

	input   textinput.Model
	output  viewport.Model
	spinner spinner.Model

	width  int
	height int

	running      bool
	commandCount int
	lastExitCode int
	transcript   []string
}

func RunConsoleTUI(ctx context.Context, env *Environment, opts ConsoleOptions) error {
	model := newConsoleModel(ctx, env, opts)
	programOptions := []tea.ProgramOption{tea.WithAltScreen()}
	if opts.Input != nil {
		programOptions = append(programOptions, tea.WithInput(opts.Input))
	}
	if opts.Output != nil {
		programOptions = append(programOptions, tea.WithOutput(opts.Output))
	}
	p := tea.NewProgram(model, programOptions...)
	_, err := p.Run()
	return err
}

func newConsoleModel(ctx context.Context, env *Environment, opts ConsoleOptions) consoleModel {
	in := textinput.New()
	in.Placeholder = "Enter command and press Enter"
	in.Focus()
	in.CharLimit = 8192
	in.Prompt = "> "
	in.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	in.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	sp := spinner.New()
	sp.Spinner = spinner.Points
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	view := viewport.New(76, 12)

	mounts := "none"
	if len(opts.Mounts) > 0 {
		mounts = strings.Join(opts.Mounts, ",")
	}
	root := strings.TrimSpace(opts.HostRoot)
	if root == "" {
		root = "(auto)"
	}

	m := consoleModel{
		ctx:     ctx,
		env:     env,
		profile: strings.TrimSpace(opts.Profile),
		policy:  strings.TrimSpace(opts.Policy),
		hostDir: root,
		mounts:  mounts,
		input:   in,
		output:  view,
		spinner: sp,
		transcript: []string{
			"simsh interactive console",
			"type exit or quit to stop",
		},
	}
	m.refreshViewport()
	return m
}

func (m consoleModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m consoleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch message := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(message.Width, message.Height)
		return m, nil
	case tea.KeyMsg:
		switch message.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyCtrlL:
			m.transcript = nil
			m.refreshViewport()
			return m, nil
		case tea.KeyEnter:
			if m.running {
				return m, nil
			}
			commandLine := strings.TrimSpace(m.input.Value())
			if commandLine == "" {
				return m, nil
			}
			if commandLine == "exit" || commandLine == "quit" {
				return m, tea.Quit
			}
			m.appendCommand(commandLine)
			m.input.SetValue("")
			m.running = true
			return m, tea.Batch(runCommand(m.ctx, m.env, commandLine), m.spinner.Tick)
		}
	case executeResultMsg:
		m.running = false
		m.commandCount++
		m.lastExitCode = message.ExitCode
		m.appendOutput(message.Output, message.ExitCode)
		return m, nil
	}

	var cmds []tea.Cmd
	if m.running {
		var spinCmd tea.Cmd
		m.spinner, spinCmd = m.spinner.Update(msg)
		if spinCmd != nil {
			cmds = append(cmds, spinCmd)
		}
	}
	var inputCmd tea.Cmd
	m.input, inputCmd = m.input.Update(msg)
	if inputCmd != nil {
		cmds = append(cmds, inputCmd)
	}
	if len(cmds) == 0 {
		return m, nil
	}
	return m, tea.Batch(cmds...)
}

func (m consoleModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "loading..."
	}

	status := "idle"
	if m.running {
		status = m.spinner.View() + " running"
	}
	topWidth := maxInt(1, m.width-2)
	leftStatus := fmt.Sprintf("profile=%s policy=%s mounts=%s", m.profile, m.policy, m.mounts)
	rightStatus := fmt.Sprintf("cmd=%d exit=%d %s", m.commandCount, m.lastExitCode, status)
	statusLine := composeStatusLine(leftStatus, rightStatus, topWidth)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("63")).Padding(0, 1)
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("237")).Padding(0, 1)
	outputStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63")).Padding(0, 1)
	inputStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("99")).Padding(0, 1)
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	title := titleStyle.Width(topWidth).Render("simsh command runtime")
	meta := metaStyle.Render("root=" + m.hostDir)
	statusBar := statusStyle.Width(topWidth).Render(statusLine)
	outputPane := outputStyle.Render(m.output.View())
	inputPane := inputStyle.Render(m.input.View())
	help := helpStyle.Render("Enter run | Ctrl+L clear | Ctrl+C quit")

	return lipgloss.JoinVertical(lipgloss.Left, title, meta, statusBar, outputPane, inputPane, help)
}

func runCommand(ctx context.Context, env *Environment, commandLine string) tea.Cmd {
	return func() tea.Msg {
		out, code := env.Execute(ctx, commandLine)
		return executeResultMsg{Output: out, ExitCode: code}
	}
}

func (m *consoleModel) resize(width, height int) {
	m.width = width
	m.height = height

	innerWidth := maxInt(24, width-4)
	outputHeight := height - 12
	if outputHeight < 6 {
		outputHeight = 6
	}

	m.output.Width = innerWidth
	m.output.Height = outputHeight
	m.input.Width = maxInt(12, innerWidth-2)
	m.refreshViewport()
}

func (m *consoleModel) appendCommand(commandLine string) {
	timestamp := time.Now().Format("15:04:05")
	m.transcript = append(m.transcript, fmt.Sprintf("[%s] $ %s", timestamp, commandLine))
	m.refreshViewport()
}

func (m *consoleModel) appendOutput(output string, exitCode int) {
	trimmed := strings.TrimRight(output, "\n")
	if trimmed != "" {
		for _, line := range strings.Split(trimmed, "\n") {
			m.transcript = append(m.transcript, "  "+line)
		}
	}
	if exitCode != 0 {
		m.transcript = append(m.transcript, fmt.Sprintf("  [exit %d]", exitCode))
	}
	m.refreshViewport()
}

func (m *consoleModel) refreshViewport() {
	m.output.SetContent(strings.Join(m.transcript, "\n"))
	m.output.GotoBottom()
}

func composeStatusLine(left, right string, width int) string {
	if width <= 0 {
		return ""
	}
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if right == "" {
		return fillToWidth(left, width)
	}

	leftWidth := runewidth.StringWidth(left)
	rightWidth := runewidth.StringWidth(right)
	if leftWidth+rightWidth+1 > width {
		leftAllowance := width - rightWidth - 1
		if leftAllowance <= 0 {
			return fillToWidth(right, width)
		}
		left = runewidth.Truncate(left, leftAllowance, "…")
		leftWidth = runewidth.StringWidth(left)
	}

	gap := width - leftWidth - rightWidth
	if gap < 1 {
		gap = 1
	}
	line := left + strings.Repeat(" ", gap) + right
	return fillToWidth(line, width)
}

func fillToWidth(text string, width int) string {
	if width <= 0 {
		return ""
	}
	trimmed := runewidth.Truncate(text, width, "")
	missing := width - runewidth.StringWidth(trimmed)
	if missing > 0 {
		trimmed += strings.Repeat(" ", missing)
	}
	return trimmed
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
