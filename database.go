package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

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

	return nil
}

func (d *Database) createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS todos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		text TEXT NOT NULL,
		done BOOLEAN DEFAULT FALSE,
		position INTEGER DEFAULT 0
	);`

	_, err := d.db.Exec(query)
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
	query := `DELETE FROM todos WHERE id = ?`
	_, err := d.db.Exec(query, id)
	return err
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
