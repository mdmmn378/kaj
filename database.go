package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type Todo struct {
	ID       int    `json:"id"`
	Text     string `json:"text"`
	Done     bool   `json:"done"`
	Position int    `json:"position"`
}

type Database struct {
	db *sql.DB
}

func NewDatabase() (*Database, error) {
	dbPath, err := getDatabasePath()
	if err != nil {
		return nil, err
	}

	os.MkdirAll(filepath.Dir(dbPath), 0755)

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	database := &Database{db: db}
	if err := database.createTables(); err != nil {
		return nil, err
	}

	return database, nil
}

func getDatabasePath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	localTodosPath := filepath.Join(cwd, ".todos", "todos.db")

	if _, err := os.Stat(filepath.Dir(localTodosPath)); err == nil {
		return localTodosPath, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, ".todos", "todos.db"), nil
}

func InitLocalDatabase() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	localTodosDir := filepath.Join(cwd, ".todos")

	if _, err := os.Stat(localTodosDir); err == nil {
		return fmt.Errorf("local .todos directory already exists")
	}

	err = os.MkdirAll(localTodosDir, 0755)
	if err != nil {
		return err
	}

	db, err := NewDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	err = addToGitignore(cwd)
	if err != nil {
		return fmt.Errorf("failed to update .gitignore: %v", err)
	}

	return nil
}

func addToGitignore(dir string) error {
	gitignorePath := filepath.Join(dir, ".gitignore")

	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		file, err := os.Create(gitignorePath)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = file.WriteString("# Local todos\n.todos/\n")
		return err
	}

	file, err := os.Open(gitignorePath)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)
	hasTodosEntry := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == ".todos/" || line == ".todos" {
			hasTodosEntry = true
			break
		}
	}
	file.Close()

	if err := scanner.Err(); err != nil {
		return err
	}

	if !hasTodosEntry {
		file, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer file.Close()

		content, err := os.ReadFile(gitignorePath)
		if err != nil {
			return err
		}

		needsNewline := len(content) > 0 && content[len(content)-1] != '\n'

		if needsNewline {
			_, err = file.WriteString("\n")
			if err != nil {
				return err
			}
		}

		_, err = file.WriteString("\n# Local todos\n.todos/\n")
		return err
	}

	return nil
}

func (d *Database) createTables() error {
	todosQuery := `
	CREATE TABLE IF NOT EXISTS todos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		text TEXT NOT NULL,
		done BOOLEAN DEFAULT FALSE,
		position INTEGER DEFAULT 0
	);`

	deletedQuery := `
	CREATE TABLE IF NOT EXISTS deleted_todos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		original_id INTEGER NOT NULL,
		text TEXT NOT NULL,
		done BOOLEAN DEFAULT FALSE,
		position INTEGER DEFAULT 0,
		deleted_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := d.db.Exec(todosQuery)
	if err != nil {
		return err
	}

	_, err = d.db.Exec(deletedQuery)
	return err
}

func (d *Database) AddTodo(text string) error {
	maxPosition, err := d.getMaxPosition()
	if err != nil {
		return err
	}

	query := `INSERT INTO todos (text, done, position) VALUES (?, FALSE, ?)`
	_, err = d.db.Exec(query, text, maxPosition+1)
	return err
}

func (d *Database) GetTodos() ([]Todo, error) {
	query := `SELECT id, text, done, position FROM todos ORDER BY position ASC`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var todo Todo
		err := rows.Scan(&todo.ID, &todo.Text, &todo.Done, &todo.Position)
		if err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}

	return todos, nil
}

func (d *Database) UpdateTodo(id int, text string) error {
	query := `UPDATE todos SET text = ? WHERE id = ?`
	_, err := d.db.Exec(query, text, id)
	return err
}

func (d *Database) ToggleTodo(id int) error {
	query := `UPDATE todos SET done = NOT done WHERE id = ?`
	_, err := d.db.Exec(query, id)
	return err
}

func (d *Database) DeleteTodo(id int) error {
	todo, err := d.getTodoByID(id)
	if err != nil {
		return err
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	insertQuery := `INSERT INTO deleted_todos (original_id, text, done, position) VALUES (?, ?, ?, ?)`
	_, err = tx.Exec(insertQuery, todo.ID, todo.Text, todo.Done, todo.Position)
	if err != nil {
		return err
	}

	deleteQuery := `DELETE FROM todos WHERE id = ?`
	_, err = tx.Exec(deleteQuery, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (d *Database) getTodoByID(id int) (*Todo, error) {
	query := `SELECT id, text, done, position FROM todos WHERE id = ?`
	row := d.db.QueryRow(query, id)

	var todo Todo
	err := row.Scan(&todo.ID, &todo.Text, &todo.Done, &todo.Position)
	if err != nil {
		return nil, err
	}

	return &todo, nil
}

func (d *Database) UndoLastDelete() (*Todo, error) {
	query := `SELECT original_id, text, done, position FROM deleted_todos ORDER BY deleted_at DESC LIMIT 1`
	row := d.db.QueryRow(query)

	var originalID int
	var text string
	var done bool
	var position int

	err := row.Scan(&originalID, &text, &done, &position)
	if err != nil {
		return nil, err
	}

	tx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	maxPosition, err := d.getMaxPosition()
	if err != nil {
		return nil, err
	}

	insertQuery := `INSERT INTO todos (text, done, position) VALUES (?, ?, ?)`
	result, err := tx.Exec(insertQuery, text, done, maxPosition+1)
	if err != nil {
		return nil, err
	}

	newID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	deleteQuery := `DELETE FROM deleted_todos WHERE id = (SELECT id FROM deleted_todos WHERE original_id = ? ORDER BY deleted_at DESC LIMIT 1)`
	_, err = tx.Exec(deleteQuery, originalID)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &Todo{
		ID:       int(newID),
		Text:     text,
		Done:     done,
		Position: maxPosition + 1,
	}, nil
}

func (d *Database) GetRecentlyDeleted(limit int) ([]Todo, error) {
	query := `SELECT original_id, text, done, position FROM deleted_todos ORDER BY deleted_at DESC LIMIT ?`
	rows, err := d.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deletedTodos []Todo
	for rows.Next() {
		var todo Todo
		err := rows.Scan(&todo.ID, &todo.Text, &todo.Done, &todo.Position)
		if err != nil {
			return nil, err
		}
		deletedTodos = append(deletedTodos, todo)
	}

	return deletedTodos, nil
}

func (d *Database) getMaxPosition() (int, error) {
	var maxPos sql.NullInt64
	query := `SELECT MAX(position) FROM todos`
	err := d.db.QueryRow(query).Scan(&maxPos)
	if err != nil {
		return 0, err
	}

	if maxPos.Valid {
		return int(maxPos.Int64), nil
	}
	return 0, nil
}

func (d *Database) MoveTodoUp(id int) error {
	todos, err := d.GetTodos()
	if err != nil {
		return err
	}

	var currentIndex = -1
	for i, todo := range todos {
		if todo.ID == id {
			currentIndex = i
			break
		}
	}

	if currentIndex <= 0 {
		return nil
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	currentTodo := todos[currentIndex]
	aboveTodo := todos[currentIndex-1]

	_, err = tx.Exec("UPDATE todos SET position = ? WHERE id = ?", aboveTodo.Position, currentTodo.ID)
	if err != nil {
		return err
	}

	_, err = tx.Exec("UPDATE todos SET position = ? WHERE id = ?", currentTodo.Position, aboveTodo.ID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (d *Database) MoveTodoDown(id int) error {
	todos, err := d.GetTodos()
	if err != nil {
		return err
	}

	var currentIndex = -1
	for i, todo := range todos {
		if todo.ID == id {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 || currentIndex >= len(todos)-1 {
		return nil
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	currentTodo := todos[currentIndex]
	belowTodo := todos[currentIndex+1]

	_, err = tx.Exec("UPDATE todos SET position = ? WHERE id = ?", belowTodo.Position, currentTodo.ID)
	if err != nil {
		return err
	}

	_, err = tx.Exec("UPDATE todos SET position = ? WHERE id = ?", currentTodo.Position, belowTodo.ID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (d *Database) Close() error {
	return d.db.Close()
}
