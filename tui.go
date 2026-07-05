package main

import (
	"fmt"
	"os"
	"strings"

	osc52 "github.com/aymanbagabas/go-osc52/v2"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF6B6B"))

	doneStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4ECDC4")).
			Strikethrough(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))
)

type model struct {
	todos        []Todo
	cursor       int
	db           *Database
	err          error
	mode         string // "list", "add", "edit", "import"
	input        string
	inputCursor  int
	editID       int
	windowHeight int
	windowWidth  int
	offset       int      // scroll offset for the list viewport
	importLines  []string // pending tasks awaiting import confirmation
}

func initialModel() model {
	db, err := NewDatabase()
	if err != nil {
		return model{err: err}
	}

	todos, err := db.GetTodos()
	if err != nil {
		return model{err: err, db: db}
	}

	return model{
		todos:  todos,
		cursor: 0,
		db:     db,
		mode:   "list",
	}
}

func (m model) Init() tea.Cmd {
	return tea.WindowSize()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowHeight = msg.Height
		m.windowWidth = msg.Width
		m.ensureCursorVisible()
		return m, nil
	case tea.KeyMsg:
		switch m.mode {
		case "list":
			return m.updateList(msg)
		case "add":
			return m.updateAdd(msg)
		case "edit":
			return m.updateEdit(msg)
		case "import":
			return m.updateImport(msg)
		}
	}
	return m, nil
}

// visibleListItems returns how many list items can fit on screen.
// Accounts for title (2 lines), help bar (2 lines), and scroll indicators (2 lines).
func (m model) visibleListItems() int {
	// title + blank = 2, blank + help = 2, up/down indicators = 2 => 6 chrome lines
	available := m.windowHeight - 6
	if available < 1 {
		available = 1
	}
	return available
}

// ensureCursorVisible adjusts the scroll offset so the cursor is visible.
func (m *model) ensureCursorVisible() {
	visible := m.visibleListItems()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// A bracketed paste (e.g. Ctrl+Shift+V in the terminal) arrives as a single
	// KeyMsg with Paste set. Parse it into tasks and ask for confirmation
	// before importing.
	if msg.Paste {
		lines := parseImportLines(string(msg.Runes), 100)
		if len(lines) > 0 {
			m.importLines = lines
			m.mode = "import"
		}
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.todos)-1 {
			m.cursor++
		}

	case "enter", " ":
		if len(m.todos) > 0 {
			todo := m.todos[m.cursor]
			err := m.db.ToggleTodo(todo.ID)
			if err != nil {
				m.err = err
				return m, nil
			}
			m.todos[m.cursor].Done = !m.todos[m.cursor].Done
		}

	case "a":
		m.mode = "add"
		m.input = ""
		m.inputCursor = 0

	case "e":
		if len(m.todos) > 0 {
			m.mode = "edit"
			m.editID = m.todos[m.cursor].ID
			m.input = m.todos[m.cursor].Text
			m.inputCursor = len(m.input)
		}

	case "d":
		if len(m.todos) > 0 {
			todo := m.todos[m.cursor]
			err := m.db.DeleteTodo(todo.ID)
			if err != nil {
				m.err = err
				return m, nil
			}

			todos, err := m.db.GetTodos()
			if err != nil {
				m.err = err
				return m, nil
			}
			m.todos = todos

			if m.cursor >= len(m.todos) && len(m.todos) > 0 {
				m.cursor = len(m.todos) - 1
			}
			if len(m.todos) == 0 {
				m.cursor = 0
			}
		}

	case "r":
		todos, err := m.db.GetTodos()
		if err != nil {
			m.err = err
			return m, nil
		}
		m.todos = todos
		if m.cursor >= len(m.todos) && len(m.todos) > 0 {
			m.cursor = len(m.todos) - 1
		}
		if len(m.todos) == 0 {
			m.cursor = 0
		}

	case "u":
		_, err := m.db.UndoLastDelete()
		if err != nil {
			if err.Error() == "sql: no rows in result set" {
				m.err = fmt.Errorf("no recently deleted todos to restore")
			} else {
				m.err = err
			}
			return m, nil
		}

		todos, err := m.db.GetTodos()
		if err != nil {
			m.err = err
			return m, nil
		}
		m.todos = todos
		m.cursor = len(m.todos) - 1

	case "y":
		if len(m.todos) > 0 {
			text := m.todos[m.cursor].Text
			_, _ = os.Stderr.WriteString(osc52.New(text).String())
		}

	case "ctrl+up", "K":
		if len(m.todos) > 0 && m.cursor > 0 {
			todo := m.todos[m.cursor]
			err := m.db.MoveTodoUp(todo.ID)
			if err != nil {
				m.err = err
				return m, nil
			}

			todos, err := m.db.GetTodos()
			if err != nil {
				m.err = err
				return m, nil
			}
			m.todos = todos
			m.cursor--
		}

	case "ctrl+down", "J":
		if len(m.todos) > 0 && m.cursor < len(m.todos)-1 {
			todo := m.todos[m.cursor]
			err := m.db.MoveTodoDown(todo.ID)
			if err != nil {
				m.err = err
				return m, nil
			}

			todos, err := m.db.GetTodos()
			if err != nil {
				m.err = err
				return m, nil
			}
			m.todos = todos
			m.cursor++
		}
	}

	m.ensureCursorVisible()
	return m, nil
}

func (m model) updateAdd(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc":
		m.mode = "list"
		m.input = ""

	case "enter":
		if m.input != "" {
			err := m.db.AddTodo(m.input)
			if err != nil {
				m.err = err
				return m, nil
			}

			todos, err := m.db.GetTodos()
			if err != nil {
				m.err = err
				return m, nil
			}
			m.todos = todos
			m.cursor = len(m.todos) - 1
		}
		m.mode = "list"
		m.input = ""

	case "left":
		if m.inputCursor > 0 {
			m.inputCursor--
		}

	case "right":
		if m.inputCursor < len(m.input) {
			m.inputCursor++
		}

	case "home", "ctrl+a":
		m.inputCursor = 0

	case "end", "ctrl+e":
		m.inputCursor = len(m.input)

	case "backspace":
		if m.inputCursor > 0 {
			m.input = m.input[:m.inputCursor-1] + m.input[m.inputCursor:]
			m.inputCursor--
		}

	case "delete":
		if m.inputCursor < len(m.input) {
			m.input = m.input[:m.inputCursor] + m.input[m.inputCursor+1:]
		}

	default:
		if s := insertableRunes(msg); s != "" {
			m.input = m.input[:m.inputCursor] + s + m.input[m.inputCursor:]
			m.inputCursor += len(s)
		}
	}

	return m, nil
}

func (m model) updateEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc":
		m.mode = "list"
		m.input = ""

	case "enter":
		if m.input != "" {
			err := m.db.UpdateTodo(m.editID, m.input)
			if err != nil {
				m.err = err
				return m, nil
			}

			todos, err := m.db.GetTodos()
			if err != nil {
				m.err = err
				return m, nil
			}
			m.todos = todos
		}
		m.mode = "list"
		m.input = ""

	case "left":
		if m.inputCursor > 0 {
			m.inputCursor--
		}

	case "right":
		if m.inputCursor < len(m.input) {
			m.inputCursor++
		}

	case "home", "ctrl+a":
		m.inputCursor = 0

	case "end", "ctrl+e":
		m.inputCursor = len(m.input)

	case "backspace":
		if m.inputCursor > 0 {
			m.input = m.input[:m.inputCursor-1] + m.input[m.inputCursor:]
			m.inputCursor--
		}

	case "delete":
		if m.inputCursor < len(m.input) {
			m.input = m.input[:m.inputCursor] + m.input[m.inputCursor+1:]
		}

	default:
		if s := insertableRunes(msg); s != "" {
			m.input = m.input[:m.inputCursor] + s + m.input[m.inputCursor:]
			m.inputCursor += len(s)
		}
	}

	return m, nil
}

func (m model) updateImport(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "y", "Y", "enter":
		if len(m.importLines) > 0 {
			err := m.db.AddTodos(m.importLines)
			if err != nil {
				m.err = err
				return m, nil
			}

			todos, err := m.db.GetTodos()
			if err != nil {
				m.err = err
				return m, nil
			}
			m.todos = todos
			m.cursor = len(m.todos) - 1
		}
		m.importLines = nil
		m.mode = "list"
		m.ensureCursorVisible()

	case "n", "N", "esc":
		m.importLines = nil
		m.mode = "list"
	}

	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
	}

	var s strings.Builder

	s.WriteString(titleStyle.Render("KAJ LIST"))
	s.WriteString("\n\n")

	switch m.mode {
	case "add":
		s.WriteString("Add new todo:\n")
		inputWithCursor := m.input[:m.inputCursor] + "│" + m.input[m.inputCursor:]
		s.WriteString(m.renderInputPrompt(inputWithCursor))
		s.WriteString("\n\n")
		s.WriteString(helpStyle.Render("Enter to save • Esc to cancel • ←/→ to move cursor"))

	case "edit":
		s.WriteString("Edit todo:\n")
		inputWithCursor := m.input[:m.inputCursor] + "│" + m.input[m.inputCursor:]
		s.WriteString(m.renderInputPrompt(inputWithCursor))
		s.WriteString("\n\n")
		s.WriteString(helpStyle.Render("Enter to save • Esc to cancel • ←/→ to move cursor"))

	case "import":
		n := len(m.importLines)
		noun := "task"
		if n != 1 {
			noun = "tasks"
		}
		s.WriteString(fmt.Sprintf("Import %d %s from clipboard?\n\n", n, noun))

		const previewCount = 5
		for i := 0; i < n && i < previewCount; i++ {
			line := m.importLines[i]
			if m.windowWidth > 4 {
				if wrapped := wrapText(line, m.windowWidth-4); len(wrapped) > 0 {
					line = wrapped[0]
				}
			}
			s.WriteString(fmt.Sprintf("  • %s\n", line))
		}
		if n > previewCount {
			s.WriteString(helpStyle.Render(fmt.Sprintf("  … and %d more", n-previewCount)))
			s.WriteString("\n")
		}
		s.WriteString("\n")
		s.WriteString(helpStyle.Render("y: import • n/esc: cancel"))

	default: // list mode
		if len(m.todos) == 0 {
			s.WriteString("No todos yet. Press 'a' to add one!\n\n")
		} else {
			visible := m.visibleListItems()
			end := m.offset + visible
			if end > len(m.todos) {
				end = len(m.todos)
			}

			if m.offset > 0 {
				s.WriteString(helpStyle.Render(fmt.Sprintf("  ↑ %d more", m.offset)))
				s.WriteString("\n")
			}

			for i := m.offset; i < end; i++ {
				todo := m.todos[i]
				cursor := " "
				if m.cursor == i {
					cursor = ">"
				}

				checked := " "
				if todo.Done {
					checked = "✓"
				}

				prefix := fmt.Sprintf("%s [%s] ", cursor, checked)
				// Prefix is always 6 display cells: cursor(1) + " ["(2) + checked(1) + "] "(2).
				const prefixCells = 6
				indent := strings.Repeat(" ", prefixCells)

				wrappedLines := wrapText(todo.Text, m.windowWidth-prefixCells)

				var block strings.Builder
				for j, wl := range wrappedLines {
					if todo.Done {
						wl = doneStyle.Render(wl)
					}
					if j == 0 {
						block.WriteString(prefix + wl)
					} else {
						block.WriteString(indent + wl)
					}
					if j < len(wrappedLines)-1 {
						block.WriteString("\n")
					}
				}

				line := block.String()
				if m.cursor == i {
					line = selectedStyle.Render(line)
				}

				s.WriteString(line)
				s.WriteString("\n")
			}

			if end < len(m.todos) {
				s.WriteString(helpStyle.Render(fmt.Sprintf("  ↓ %d more", len(m.todos)-end)))
				s.WriteString("\n")
			}
		}

		s.WriteString("\n")
		helpText := "a: add • e: edit • d: delete • u: undo • y: yank • Ctrl+Shift+V: import • space/enter: toggle • Ctrl+↑/K: move up • Ctrl+↓/J: move down • r: refresh • q: quit"
		if m.windowWidth > 0 {
			s.WriteString(helpStyle.Render(lipgloss.NewStyle().Width(m.windowWidth).Render(helpText)))
		} else {
			s.WriteString(helpStyle.Render(helpText))
		}
	}

	return s.String()
}

// renderInputPrompt formats the "> ..." input line, wrapping long input
// (typed or pasted) so it doesn't overflow the terminal width. Continuation
// lines are indented under the prompt.
func (m model) renderInputPrompt(input string) string {
	const promptCells = 2 // "> "
	if m.windowWidth <= promptCells {
		return "> " + input
	}
	lines := wrapText(input, m.windowWidth-promptCells)
	indent := strings.Repeat(" ", promptCells)
	var b strings.Builder
	for i, l := range lines {
		if i == 0 {
			b.WriteString("> ")
		} else {
			b.WriteString("\n")
			b.WriteString(indent)
		}
		b.WriteString(l)
	}
	return b.String()
}

// wrapText wraps text to fit within maxCellWidth display cells, preferring
// word boundaries and hard-breaking only when a token is longer than the
// width. It delegates to lipgloss/cellbuf so multi-byte runes and wide
// characters (CJK, emoji) are measured in cells, not bytes.
func wrapText(text string, maxCellWidth int) []string {
	if maxCellWidth < 1 || lipgloss.Width(text) <= maxCellWidth {
		return []string{text}
	}
	wrapped := lipgloss.NewStyle().Width(maxCellWidth).Render(text)
	lines := strings.Split(wrapped, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return lines
}

// insertableRunes returns the runes from a KeyMsg that should be inserted
// verbatim into a text field. It accepts both single typed characters and
// pasted multi-rune chunks (bracketed paste arrives as one KeyMsg with many
// runes), and rejects control keys.
func insertableRunes(msg tea.KeyMsg) string {
	switch msg.Type {
	case tea.KeyRunes, tea.KeySpace:
		return string(msg.Runes)
	}
	return ""
}

// parseImportLines turns pasted clipboard text into a list of task strings:
// it splits on newlines, trims surrounding whitespace (including CR from
// CRLF line endings), drops blank lines, and caps the result to the first
// max entries.
func parseImportLines(s string, max int) []string {
	var lines []string
	for _, raw := range strings.Split(s, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		lines = append(lines, line)
		if len(lines) >= max {
			break
		}
	}
	return lines
}

func runTUI() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v", err)
	}
}
