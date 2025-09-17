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
	todos  []Todo
	cursor int
	db     *Database
	err    error
	mode   string // "list", "add", "edit"
	input  string
	editID int
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
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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

	case "e":
		if len(m.todos) > 0 {
			m.mode = "edit"
			m.editID = m.todos[m.cursor].ID
			m.input = m.todos[m.cursor].Text
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

	case "backspace":
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}

	default:
		if len(msg.String()) == 1 {
			m.input += msg.String()
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

	case "backspace":
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}

	default:
		if len(msg.String()) == 1 {
			m.input += msg.String()
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
		s.WriteString(fmt.Sprintf("> %s", m.input))
		s.WriteString("\n\n")
		s.WriteString(helpStyle.Render("Enter to save • Esc to cancel"))

	case "edit":
		s.WriteString("Edit todo:\n")
		s.WriteString(fmt.Sprintf("> %s", m.input))
		s.WriteString("\n\n")
		s.WriteString(helpStyle.Render("Enter to save • Esc to cancel"))

	default: // list mode
		if len(m.todos) == 0 {
			s.WriteString("No todos yet. Press 'a' to add one!\n\n")
		} else {
			for i, todo := range m.todos {
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
		}

		s.WriteString("\n")
		s.WriteString(helpStyle.Render("a: add • e: edit • d: delete • space/enter: toggle • Ctrl+↑/J: move up • Ctrl+↓/K: move down • r: refresh • q: quit"))
	}

	return s.String()
}

func runTUI() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v", err)
	}
}
