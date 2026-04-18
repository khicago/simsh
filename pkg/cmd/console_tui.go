package cmd

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/termenv"
)

type ConsoleOptions struct {
	Profile   string
	Policy    string
	HostRoot  string
	SessionID string
	Mounts    []string
	Input     io.Reader
	Output    io.Writer
}

type executeResultMsg struct {
	Output   string
	ExitCode int
	RunID    int
	Duration time.Duration
}

type workingDirReader interface {
	WorkingDir() string
}

type transcriptKind int

const (
	lineBanner transcriptKind = iota
	lineCommand
	lineOutput
	lineMeta
)

type transcriptLine struct {
	Kind transcriptKind
	Text string
}

type consoleModel struct {
	ctx context.Context
	env Executor
	now func() time.Time

	profile string
	policy  string
	hostDir string
	session string
	mounts  string
	cwd     string

	input   textinput.Model
	output  viewport.Model
	spinner spinner.Model

	width  int
	height int

	running         bool
	cancelRequested bool
	showHelp        bool
	followTail      bool
	commandCount    int
	lastExitCode    int
	lastCommand     string
	lastDuration    time.Duration
	lines           []transcriptLine

	history      []string
	historyIndex int
	draft        string

	cancelRunning context.CancelFunc
	activeRunID   int
	nextRunID     int
}

func RunConsoleTUI(ctx context.Context, env Executor, opts ConsoleOptions) error {
	configureOperatorConsoleRenderer()
	model := newConsoleModel(ctx, env, opts)
	model.resize(80, 24)
	programOptions := []tea.ProgramOption{
		tea.WithAltScreen(),
		tea.WithContext(ctx),
		tea.WithMouseCellMotion(),
	}
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

func newConsoleModel(ctx context.Context, env Executor, opts ConsoleOptions) consoleModel {
	in := textinput.New()
	in.Placeholder = "Type a command…"
	in.Focus()
	in.CharLimit = 8192
	in.Prompt = "❯ "
	in.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#C084FC"))
	in.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#E4E4E7"))
	in.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#71717A"))
	in.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#E879F9"))

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	view := viewport.New(76, 12)
	view.MouseWheelEnabled = true

	mounts := "none"
	if len(opts.Mounts) > 0 {
		mounts = strings.Join(opts.Mounts, ",")
	}
	root := strings.TrimSpace(opts.HostRoot)
	if root == "" {
		root = "(auto)"
	}

	m := consoleModel{
		ctx:        ctx,
		env:        env,
		now:        time.Now,
		profile:    strings.TrimSpace(opts.Profile),
		policy:     strings.TrimSpace(opts.Policy),
		hostDir:    root,
		session:    strings.TrimSpace(opts.SessionID),
		mounts:     mounts,
		cwd:        readWorkingDir(env),
		input:      in,
		output:     view,
		spinner:    sp,
		followTail: true,
		lines: []transcriptLine{
			{Kind: lineBanner, Text: "human operator console"},
			{Kind: lineBanner, Text: "agents should use the CLI or HTTP execute surface"},
		},
	}
	m.historyIndex = 0
	m.setPrompt()
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
	case tea.MouseMsg:
		var cmd tea.Cmd
		m.output, cmd = m.output.Update(message)
		m.followTail = m.output.AtBottom()
		return m, cmd
	case spinner.TickMsg:
		if !m.running {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(message)
		return m, cmd
	case tea.KeyMsg:
		return m.updateKey(message)
	case executeResultMsg:
		if message.RunID == 0 || message.RunID != m.activeRunID {
			return m, nil
		}
		m.running = false
		m.cancelRequested = false
		if m.cancelRunning != nil {
			m.cancelRunning()
			m.cancelRunning = nil
		}
		m.activeRunID = 0
		m.commandCount++
		m.lastExitCode = message.ExitCode
		m.lastDuration = message.Duration
		m.cwd = readWorkingDir(m.env)
		m.setPrompt()
		m.appendOutput(message.Output, message.ExitCode)
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.sanitizeInput()
	return m, cmd
}

func (m consoleModel) updateKey(message tea.KeyMsg) (tea.Model, tea.Cmd) {
	if message.Type == tea.KeyCtrlC && m.running {
		m.showHelp = false
		if m.cancelExecution() {
			m.appendMeta("cancel requested")
		}
		return m, nil
	}
	if m.showHelp {
		switch message.Type {
		case tea.KeyEsc, tea.KeyEnter, tea.KeyCtrlC:
			m.showHelp = false
			return m, nil
		}
		if message.String() == "?" || message.String() == "q" {
			m.showHelp = false
			return m, nil
		}
	}

	switch message.Type {
	case tea.KeyCtrlD:
		m.cancelExecution()
		return m, tea.Quit
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		if m.running {
			if m.cancelExecution() {
				m.appendMeta("cancel requested")
			}
		}
		return m, nil
	case tea.KeyCtrlL:
		m.lines = nil
		m.followTail = true
		m.refreshViewport()
		return m, nil
	case tea.KeyPgUp:
		m.output.ViewUp()
		m.followTail = m.output.AtBottom()
		return m, nil
	case tea.KeyPgDown:
		m.output.ViewDown()
		m.followTail = m.output.AtBottom()
		return m, nil
	case tea.KeyUp:
		m.historyPrev()
		return m, nil
	case tea.KeyDown:
		m.historyNext()
		return m, nil
	case tea.KeyEnter:
		if m.running {
			return m, nil
		}
		commandLine := strings.TrimSpace(stripTerminalReportJunk(m.input.Value()))
		if commandLine == "" {
			return m, nil
		}
		if commandLine == "exit" || commandLine == "quit" {
			m.cancelExecution()
			return m, tea.Quit
		}
		if commandLine == "?" || commandLine == "help" {
			m.showHelp = true
			m.input.SetValue("")
			return m, nil
		}
		m.pushHistory(commandLine)
		m.appendCommand(commandLine)
		m.lastCommand = commandLine
		m.input.SetValue("")
		m.running = true
		m.nextRunID++
		m.activeRunID = m.nextRunID
		commandCtx, cancel := context.WithCancel(m.ctx)
		m.cancelRunning = cancel
		return m, tea.Batch(runCommand(commandCtx, m.env, commandLine, m.activeRunID), m.spinner.Tick)
	}

	if message.String() == "?" && strings.TrimSpace(m.input.Value()) == "" {
		m.showHelp = true
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(message)
	m.sanitizeInput()
	return m, cmd
}

func configureOperatorConsoleRenderer() {
	// Bubble Tea's package init may already have queried OSC 11. Pin the
	// renderer so later lipgloss calls do not query again. Do not Read stdin
	// here: a TTY Read without a working deadline will hang make cli.
	lipgloss.SetColorProfile(termenv.TrueColor)
	lipgloss.SetHasDarkBackground(true)
}

var terminalReportPattern = regexp.MustCompile(`(?i)\x1b?\](?:10|11|12);(?:rgb:)?[0-9a-f/]+(?:\x1b\\|\x07|\\)?|\x1b\[[0-9]+;[0-9]+R`)

func stripTerminalReportJunk(s string) string {
	return terminalReportPattern.ReplaceAllString(s, "")
}

func (m *consoleModel) sanitizeInput() {
	current := m.input.Value()
	cleaned := stripTerminalReportJunk(current)
	if cleaned == current {
		return
	}
	m.input.SetValue(cleaned)
	m.input.CursorEnd()
}

func (m consoleModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "loading..."
	}

	state, stateColor := consoleState(m.running, m.commandCount, m.lastExitCode)
	if m.running {
		if m.cancelRequested {
			state = m.spinner.View() + " canceling"
		} else {
			state = m.spinner.View() + " running"
		}
	}
	topWidth := maxInt(1, m.width)

	brand := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252")).Render("simsh")
	chip := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(stateColor)).Render(state)
	session := "session=" + fallbackConsoleLabel(m.session, "ephemeral")
	header := composeStatusLine(brand+"  "+chip, session, topWidth)

	leftMeta := strings.Join([]string{
		fallbackConsoleLabel(m.profile, "core-strict"),
		fallbackConsoleLabel(m.policy, "read-only"),
		"cwd=" + middleElide(m.cwd, 28),
		"root=" + middleElide(m.hostDir, 20),
	}, "  ")
	rightMeta := fmt.Sprintf("cmds=%d  exit=%d", m.commandCount, m.lastExitCode)
	if m.lastDuration > 0 {
		rightMeta += "  " + formatDuration(m.lastDuration)
	}
	if last := strings.TrimSpace(m.lastCommand); last != "" {
		rightMeta += "  last=" + middleElide(last, 24)
	}
	meta := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(composeStatusLine(leftMeta, rightMeta, topWidth))
	rule := lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render(strings.Repeat("─", topWidth))

	body := m.output.View()
	if m.showHelp {
		body = renderHelpOverlay(m.output.Width, m.output.Height)
	}

	help := lipgloss.NewStyle().Foreground(lipgloss.Color("#71717A")).Render(m.footerHelp(topWidth))
	composer := m.composerView(topWidth)
	return lipgloss.JoinVertical(lipgloss.Left, header, meta, rule, body, composer, help)
}

func (m consoleModel) composerView(width int) string {
	accent := lipgloss.Color("#C084FC")
	if m.running {
		accent = lipgloss.Color("#FBBF24")
	} else if m.commandCount > 0 && m.lastExitCode != 0 {
		accent = lipgloss.Color("#F87171")
	}
	title := fallbackConsoleLabel(m.cwd, "/")
	return roundedComposerBox(title, m.input.View(), width, accent)
}

func roundedComposerBox(title, body string, width int, border lipgloss.Color) string {
	if width < 8 {
		width = 8
	}
	inner := width - 2
	label := ""
	if strings.TrimSpace(title) != "" {
		label = " " + middleElide(strings.TrimSpace(title), maxInt(1, inner-4)) + " "
	}
	remain := inner - 1 - runewidth.StringWidth(label)
	var top string
	if remain < 0 || label == "" {
		top = "╭" + strings.Repeat("─", inner) + "╮"
	} else {
		top = "╭─" + label + strings.Repeat("─", remain) + "╮"
	}
	visible := lipgloss.Width(body)
	if visible > inner {
		body = runewidth.Truncate(body, inner, "…")
		visible = lipgloss.Width(body)
	}
	if pad := inner - visible; pad > 0 {
		body += strings.Repeat(" ", pad)
	}
	edge := lipgloss.NewStyle().Foreground(border)
	mid := edge.Render("│") + body + edge.Render("│")
	bot := "╰" + strings.Repeat("─", inner) + "╯"
	return edge.Render(top) + "\n" + mid + "\n" + edge.Render(bot)
}

func (m consoleModel) footerHelp(width int) string {
	parts := []string{"enter run", "↑↓ history", "pgup/pgdn scroll", "? help"}
	if m.cancelRequested {
		parts = append([]string{"cancel requested"}, parts...)
	} else if m.running {
		parts = append([]string{"^c cancel"}, parts...)
	} else {
		parts = append(parts, "^d quit")
	}
	if !m.followTail {
		parts = append([]string{"follow paused"}, parts...)
	}
	return composeStatusLine(strings.Join(parts, "  ·  "), "mounts="+m.mounts, width)
}

func consoleState(running bool, commandCount int, lastExitCode int) (string, string) {
	switch {
	case running:
		return "running", "214"
	case commandCount > 0 && lastExitCode != 0:
		return "failed", "203"
	default:
		return "idle", "108"
	}
}

func runCommand(ctx context.Context, env Executor, commandLine string, runID int) tea.Cmd {
	started := time.Now()
	return func() tea.Msg {
		out, code := env.Execute(ctx, commandLine)
		return executeResultMsg{
			Output:   out,
			ExitCode: code,
			RunID:    runID,
			Duration: time.Since(started),
		}
	}
}

func fallbackConsoleLabel(value string, fallback string) string {
	label := strings.TrimSpace(value)
	if label == "" {
		return fallback
	}
	return label
}

func readWorkingDir(env Executor) string {
	if reader, ok := env.(workingDirReader); ok {
		if cwd := strings.TrimSpace(reader.WorkingDir()); cwd != "" {
			return cwd
		}
	}
	return "/"
}

func (m *consoleModel) setPrompt() {
	m.input.Prompt = "❯ "
}

func (m *consoleModel) resize(width, height int) {
	m.width = width
	m.height = height

	innerWidth := maxInt(20, width)
	outputHeight := height - 7
	if outputHeight < 6 {
		outputHeight = 6
	}

	m.output.Width = innerWidth
	m.output.Height = outputHeight
	composerInner := maxInt(8, innerWidth-2)
	m.input.Width = maxInt(4, composerInner-runewidth.StringWidth(m.input.Prompt))
	m.refreshViewport()
}

func (m *consoleModel) appendCommand(commandLine string) {
	stamp := m.now().Format("15:04:05")
	m.lines = append(m.lines, transcriptLine{
		Kind: lineCommand,
		Text: fmt.Sprintf("%s  %s", stamp, commandLine),
	})
	m.followTail = true
	m.refreshViewport()
}

func (m *consoleModel) appendOutput(output string, exitCode int) {
	trimmed := strings.TrimRight(output, "\n")
	if trimmed != "" {
		for _, line := range strings.Split(trimmed, "\n") {
			m.lines = append(m.lines, transcriptLine{Kind: lineOutput, Text: line})
		}
	}
	if exitCode != 0 {
		m.appendMeta(fmt.Sprintf("exit %d", exitCode))
		return
	}
	m.refreshViewport()
}

func (m *consoleModel) appendMeta(text string) {
	m.lines = append(m.lines, transcriptLine{Kind: lineMeta, Text: text})
	m.refreshViewport()
}

func (m *consoleModel) refreshViewport() {
	rendered := make([]string, 0, len(m.lines))
	commandStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Bold(true)
	bannerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	outputStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	for _, line := range m.lines {
		switch line.Kind {
		case lineCommand:
			rendered = append(rendered, commandStyle.Render(line.Text))
		case lineMeta:
			rendered = append(rendered, metaStyle.Render(line.Text))
		case lineBanner:
			rendered = append(rendered, bannerStyle.Render(line.Text))
		default:
			rendered = append(rendered, outputStyle.Render(line.Text))
		}
	}
	m.output.SetContent(strings.Join(rendered, "\n"))
	if m.followTail {
		m.output.GotoBottom()
	}
}

func (m *consoleModel) cancelExecution() bool {
	if !m.running || m.cancelRequested {
		return false
	}
	m.cancelRequested = true
	if m.cancelRunning != nil {
		m.cancelRunning()
		m.cancelRunning = nil
	}
	return true
}

func (m *consoleModel) pushHistory(commandLine string) {
	if n := len(m.history); n > 0 && m.history[n-1] == commandLine {
		m.historyIndex = len(m.history)
		m.draft = ""
		return
	}
	m.history = append(m.history, commandLine)
	m.historyIndex = len(m.history)
	m.draft = ""
}

func (m *consoleModel) historyPrev() {
	if len(m.history) == 0 {
		return
	}
	if m.historyIndex == len(m.history) {
		m.draft = m.input.Value()
	}
	if m.historyIndex == 0 {
		return
	}
	m.historyIndex--
	m.input.SetValue(m.history[m.historyIndex])
	m.input.CursorEnd()
}

func (m *consoleModel) historyNext() {
	if m.historyIndex >= len(m.history) {
		return
	}
	m.historyIndex++
	if m.historyIndex == len(m.history) {
		m.input.SetValue(m.draft)
		m.input.CursorEnd()
		return
	}
	m.input.SetValue(m.history[m.historyIndex])
	m.input.CursorEnd()
}

func renderHelpOverlay(width, height int) string {
	if width < 1 || height < 1 {
		return ""
	}
	lines := []string{
		"keys",
		"enter           run command",
		"up / down       command history",
		"pgup / pgdn     scroll output",
		"ctrl+l          clear transcript",
		"ctrl+c          cancel running command",
		"ctrl+d          quit",
		"exit / quit     quit",
		"? / help        this overlay",
		"",
		"agents should call the CLI or HTTP execute API,",
		"not this console.",
	}
	boxWidth := minInt(width-2, 52)
	if boxWidth < 20 {
		boxWidth = maxInt(1, width)
	}
	inner := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(boxWidth).
		Render(strings.Join(lines, "\n"))
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, inner)
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

func middleElide(text string, maxWidth int) string {
	text = strings.TrimSpace(text)
	if maxWidth <= 0 {
		return ""
	}
	if runewidth.StringWidth(text) <= maxWidth {
		return text
	}
	if maxWidth == 1 {
		return "…"
	}
	content := maxWidth - 1
	left := content / 2
	right := content - left
	return runewidth.Truncate(text, left, "") + "…" + takeSuffixWidth(text, right)
}

func takeSuffixWidth(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	runes := []rune(text)
	width := 0
	var out []rune
	for i := len(runes) - 1; i >= 0; i-- {
		w := runewidth.RuneWidth(runes[i])
		if width+w > maxWidth {
			break
		}
		out = append(out, runes[i])
		width += w
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return "<1ms"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
