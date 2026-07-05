package main

import (
	"testing"
)

// newTestDatabase returns a Database backed by a throwaway location. It points
// both the working directory and HOME at temp dirs so getDatabasePath resolves
// to a fresh ~/.todos/todos.db and never touches the repo's real database.
func newTestDatabase(t *testing.T) *Database {
	t.Helper()
	t.Chdir(t.TempDir())
	t.Setenv("HOME", t.TempDir())

	db, err := NewDatabase()
	if err != nil {
		t.Fatalf("NewDatabase: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestAddTodos(t *testing.T) {
	db := newTestDatabase(t)

	texts := []string{"first", "second", "third"}
	if err := db.AddTodos(texts); err != nil {
		t.Fatalf("AddTodos: %v", err)
	}

	todos, err := db.GetTodos()
	if err != nil {
		t.Fatalf("GetTodos: %v", err)
	}

	if len(todos) != len(texts) {
		t.Fatalf("expected %d todos, got %d", len(texts), len(todos))
	}
	for i, todo := range todos {
		if todo.Text != texts[i] {
			t.Errorf("todo %d: expected text %q, got %q", i, texts[i], todo.Text)
		}
		if todo.Done {
			t.Errorf("todo %d: expected not done", i)
		}
	}

	// Positions should be contiguous and increasing.
	for i := 1; i < len(todos); i++ {
		if todos[i].Position != todos[i-1].Position+1 {
			t.Errorf("expected contiguous positions, got %d then %d",
				todos[i-1].Position, todos[i].Position)
		}
	}
}

func TestAddTodosAppendsAfterExisting(t *testing.T) {
	db := newTestDatabase(t)

	if err := db.AddTodo("existing"); err != nil {
		t.Fatalf("AddTodo: %v", err)
	}
	if err := db.AddTodos([]string{"new one", "new two"}); err != nil {
		t.Fatalf("AddTodos: %v", err)
	}

	todos, err := db.GetTodos()
	if err != nil {
		t.Fatalf("GetTodos: %v", err)
	}

	want := []string{"existing", "new one", "new two"}
	if len(todos) != len(want) {
		t.Fatalf("expected %d todos, got %d", len(want), len(todos))
	}
	for i, todo := range todos {
		if todo.Text != want[i] {
			t.Errorf("todo %d: expected %q, got %q", i, want[i], todo.Text)
		}
	}
}

func TestAddTodosEmptyIsNoop(t *testing.T) {
	db := newTestDatabase(t)

	if err := db.AddTodos(nil); err != nil {
		t.Fatalf("AddTodos(nil): %v", err)
	}

	todos, err := db.GetTodos()
	if err != nil {
		t.Fatalf("GetTodos: %v", err)
	}
	if len(todos) != 0 {
		t.Fatalf("expected no todos, got %d", len(todos))
	}
}
