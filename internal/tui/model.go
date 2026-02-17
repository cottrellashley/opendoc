package tui

import (
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Messages ────────────────────────────────────────────────

type cmdOutputMsg struct {
	output string
	err    error
}

// ── Model ───────────────────────────────────────────────────

// Model is the top-level Bubble Tea model.
type Model struct {
	screen     screen
	width      int
	height     int
	docs       []DocItem
	docCursor  int
	docDetail  bool
	cmds       []Command
	cmdCursor  int
	cmdOutput  string
	cmdRunning bool
}

// NewModel creates a new TUI model with default data.
func NewModel(allowedCmds []string) Model {
	return Model{
		screen: screenHome,
		docs:   DefaultDocs(),
		cmds:   DefaultCommands(allowedCmds),
	}
}

// ── Bubble Tea interface ────────────────────────────────────

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case cmdOutputMsg:
		m.cmdRunning = false
		if msg.err != nil {
			m.cmdOutput = "Error: " + msg.err.Error() + "\n" + msg.output
		} else {
			m.cmdOutput = msg.output
		}
		return m, nil
	}

	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	header := m.viewHeader()
	status := m.viewStatusBar()

	var content string
	switch m.screen {
	case screenHome:
		content = viewHome(m.width)
	case screenDocs:
		content = viewDocs(m.docs, m.docCursor, m.docDetail)
	case screenCommands:
		content = viewCommands(m.cmds, m.cmdCursor, m.cmdOutput, m.cmdRunning)
	}

	// Fill remaining height.
	headerH := lipgloss.Height(header)
	statusH := lipgloss.Height(status)
	contentH := m.height - headerH - statusH
	if contentH < 1 {
		contentH = 1
	}

	paddedContent := lipgloss.NewStyle().
		Width(m.width).
		Height(contentH).
		Render(content)

	return header + "\n" + paddedContent + "\n" + status
}

// ── Key handling ────────────────────────────────────────────

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global keys.
	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "1":
		m.screen = screenHome
		return m, nil
	case "2":
		m.screen = screenDocs
		return m, nil
	case "3":
		m.screen = screenCommands
		return m, nil
	}

	// Screen-specific keys.
	switch m.screen {
	case screenDocs:
		return m.handleDocsKey(key)
	case screenCommands:
		return m.handleCommandsKey(key)
	}

	return m, nil
}

func (m Model) handleDocsKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.docCursor > 0 {
			m.docCursor--
			m.docDetail = false
		}
	case "down", "j":
		if m.docCursor < len(m.docs)-1 {
			m.docCursor++
			m.docDetail = false
		}
	case "enter":
		m.docDetail = !m.docDetail
	case "esc":
		m.docDetail = false
	}
	return m, nil
}

func (m Model) handleCommandsKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.cmdCursor > 0 {
			m.cmdCursor--
		}
	case "down", "j":
		if m.cmdCursor < len(m.cmds)-1 {
			m.cmdCursor++
		}
	case "enter":
		if !m.cmdRunning && m.cmdCursor < len(m.cmds) {
			m.cmdRunning = true
			m.cmdOutput = ""
			cmdStr := m.cmds[m.cmdCursor].Cmd
			return m, runCommand(cmdStr)
		}
	}
	return m, nil
}

// ── Header + status bar ─────────────────────────────────────

func (m Model) viewHeader() string {
	tabs := []struct {
		label  string
		screen screen
	}{
		{"Home", screenHome},
		{"Docs", screenDocs},
		{"Commands", screenCommands},
	}

	var parts []string
	for _, t := range tabs {
		if t.screen == m.screen {
			parts = append(parts, accentStyle.Bold(true).Render(" "+t.label+" "))
		} else {
			parts = append(parts, mutedStyle.Render(" "+t.label+" "))
		}
	}

	left := lipgloss.JoinHorizontal(lipgloss.Center, parts...)
	title := accentStyle.Bold(true).Render("opendoc")
	line := headerStyle.Width(m.width).Render(title + "  " + left)
	return line
}

func (m Model) viewStatusBar() string {
	screenName := ""
	switch m.screen {
	case screenHome:
		screenName = "home"
	case screenDocs:
		screenName = "docs"
	case screenCommands:
		screenName = "commands"
	}
	left := mutedStyle.Render(" 1/2/3 switch · q quit")
	right := mutedStyle.Render(screenName + " ")
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	return statusBarStyle.Width(m.width).Render(left + strings.Repeat(" ", gap) + right)
}

// ── Command execution ───────────────────────────────────────

func runCommand(cmdStr string) tea.Cmd {
	return func() tea.Msg {
		parts := strings.Fields(cmdStr)
		if len(parts) == 0 {
			return cmdOutputMsg{output: "", err: nil}
		}
		cmd := exec.Command(parts[0], parts[1:]...)
		cmd.Dir = "/workspace"
		out, err := cmd.CombinedOutput()
		return cmdOutputMsg{output: string(out), err: err}
	}
}
