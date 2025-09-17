# Kaj - Simple Todo CLI

A simple, elegant CLI todo list manager with a beautiful TUI (Terminal User Interface) built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

> **Disclaimer**: This tool was created for my personal use with vibe coding. It's a quick and simple solution that works for my workflow, but may not cover all edge cases or use patterns. Feel free to fork and modify as needed!

## Features

- **CLI Commands**: Add, list, edit, toggle, and delete todos from the command line
- **Interactive TUI**: Beautiful terminal interface for managing todos
- **Persistent Storage**: SQLite database stored in `~/.todos/` directory
- **Git Integration**: Automatically ignores `.todos/` directory
- **Keyboard Navigation**: Vim-style keybindings and intuitive controls

## Installation

### From Pre-built Binaries (Recommended)

Download the latest release for your platform from [GitHub Releases](https://github.com/mdmmn378/kaj/releases):

```bash
# Linux (AMD64)
curl -L -o kaj https://github.com/mdmmn378/kaj/releases/latest/download/kaj-linux-amd64
chmod +x kaj
sudo mv kaj /usr/local/bin/

# macOS (Apple Silicon)
curl -L -o kaj https://github.com/mdmmn378/kaj/releases/latest/download/kaj-darwin-arm64
chmod +x kaj
sudo mv kaj /usr/local/bin/
```

### From Source

```bash
go install github.com/mdmmn378/kaj@latest
```

> **Note**: Make sure you have Go 1.21+ installed for the `go install` command to work.

### Build Locally

```bash
git clone https://github.com/mdmmn378/kaj.git
cd kaj
go build -o kaj
```

## Usage

### Command Line Interface

```bash
# Add a new todo
kaj add "call grandma"

# List all todos
kaj list

# Edit a todo (by index)
kaj edit 1 "call grandma at 1 pm"

# Toggle todo completion
kaj toggle 1

# Delete a todo
kaj delete 1

# Undo last deletion
kaj undo

# Initialize local project todos
kaj init

# Check which database is being used
kaj status

# Show version information
kaj version
```

### Interactive TUI

Simply run `kaj` without arguments to enter the interactive mode:

```bash
kaj
```

#### TUI Controls

- `↑/k`: Move cursor up
- `↓/j`: Move cursor down
- `Space/Enter`: Toggle todo completion
- `a`: Add new todo
- `e`: Edit selected todo
- `d`: Delete selected todo
- `u`: Undo last deletion
- `Ctrl+↑/J`: Move task up in list
- `Ctrl+↓/K`: Move task down in list
- `r`: Refresh list
- `q`: Quit

## Database

Kaj supports both **global** and **local** todo lists:

### Global Todos

- Stored in `~/.todos/todos.db` (your home directory)
- Available from anywhere on your system
- Default behavior when no local todos exist

### Local Project Todos

- Created with `kaj init` in any directory
- Stored in `.todos/todos.db` within that directory
- Takes precedence over global todos when present
- Perfect for project-specific tasks
- Automatically git-ignored

### Database Priority

1. **Local first**: If `.todos/` exists in current directory, use local database
2. **Global fallback**: Otherwise, use global database in home directory

Use `kaj status` to see which database is currently active.

## Examples

```bash
# Add some todos
kaj add "call grandma"
kaj add "buy groceries"
kaj add "finish project"

# List them
kaj list
# Output:
# 1. [ ] call grandma
# 2. [ ] buy groceries
# 3. [ ] finish project

# Edit first todo
kaj edit 1 "call grandma at 1 pm"

# Toggle completion
kaj toggle 1

kaj list
# Output:
# 1. [x] call grandma at 1 pm
# 2. [ ] buy groceries
# 3. [ ] finish project

# Launch interactive TUI
kaj
```

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [SQLite](https://github.com/mattn/go-sqlite3) - Database driver
