package main

import (
	"fmt"
	"strings"

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
	mode         string // "list", "add", "edit"
	input        string
	inputCursor  int
	editID       int
	windowHeight int
	offset       int // scroll offset for the list viewport
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
		if len(msg.String()) == 1 {
			m.input = m.input[:m.inputCursor] + msg.String() + m.input[m.inputCursor:]
			m.inputCursor++
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
		if len(msg.String()) == 1 {
			m.input = m.input[:m.inputCursor] + msg.String() + m.input[m.inputCursor:]
			m.inputCursor++
		}
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
		s.WriteString(fmt.Sprintf("> %s", inputWithCursor))
		s.WriteString("\n\n")
		s.WriteString(helpStyle.Render("Enter to save • Esc to cancel • ←/→ to move cursor"))

	case "edit":
		s.WriteString("Edit todo:\n")
		inputWithCursor := m.input[:m.inputCursor] + "│" + m.input[m.inputCursor:]
		s.WriteString(fmt.Sprintf("> %s", inputWithCursor))
		s.WriteString("\n\n")
		s.WriteString(helpStyle.Render("Enter to save • Esc to cancel • ←/→ to move cursor"))

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

				text := todo.Text
				if todo.Done {
					text = doneStyle.Render(text)
				}

				line := fmt.Sprintf("%s [%s] %s", cursor, checked, text)
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
		s.WriteString(helpStyle.Render("a: add • e: edit • d: delete • u: undo • space/enter: toggle • Ctrl+↑/K: move up • Ctrl+↓/J: move down • r: refresh • q: quit"))
	}

	return s.String()
}

func runTUI() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v", err)
	}
}
