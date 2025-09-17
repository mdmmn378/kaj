package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "kaj",
	Short:   "A simple todo list manager",
	Long:    "A simple CLI todo list manager with TUI interface built with Bubble Tea",
	Version: Version,
	Run: func(cmd *cobra.Command, args []string) {
		runTUI()
	},
}

var addCmd = &cobra.Command{
	Use:   "add [todo text]",
	Short: "Add a new todo item",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		db, err := NewDatabase()
		if err != nil {
			fmt.Printf("Error opening database: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		todoText := ""
		for i, arg := range args {
			if i > 0 {
				todoText += " "
			}
			todoText += arg
		}

		err = db.AddTodo(todoText)
		if err != nil {
			fmt.Printf("Error adding todo: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Added: %s\n", todoText)
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all todo items",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := NewDatabase()
		if err != nil {
			fmt.Printf("Error opening database: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		todos, err := db.GetTodos()
		if err != nil {
			fmt.Printf("Error getting todos: %v\n", err)
			os.Exit(1)
		}

		if len(todos) == 0 {
			fmt.Println("No todos found")
			return
		}

		for i, todo := range todos {
			status := " "
			if todo.Done {
				status = "x"
			}
			fmt.Printf("%d. [%s] %s\n", i+1, status, todo.Text)
		}
	},
}

var editCmd = &cobra.Command{
	Use:   "edit [index] [new text]",
	Short: "Edit a todo item",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		db, err := NewDatabase()
		if err != nil {
			fmt.Printf("Error opening database: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		index, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Printf("Invalid index: %s\n", args[0])
			os.Exit(1)
		}

		todos, err := db.GetTodos()
		if err != nil {
			fmt.Printf("Error getting todos: %v\n", err)
			os.Exit(1)
		}

		if index < 1 || index > len(todos) {
			fmt.Printf("Index out of range: %d\n", index)
			os.Exit(1)
		}

		todo := todos[index-1]

		newText := ""
		for i, arg := range args[1:] {
			if i > 0 {
				newText += " "
			}
			newText += arg
		}

		err = db.UpdateTodo(todo.ID, newText)
		if err != nil {
			fmt.Printf("Error updating todo: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Updated: %s\n", newText)
	},
}

var toggleCmd = &cobra.Command{
	Use:   "toggle [index]",
	Short: "Toggle a todo item as done/undone",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		db, err := NewDatabase()
		if err != nil {
			fmt.Printf("Error opening database: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		index, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Printf("Invalid index: %s\n", args[0])
			os.Exit(1)
		}

		todos, err := db.GetTodos()
		if err != nil {
			fmt.Printf("Error getting todos: %v\n", err)
			os.Exit(1)
		}

		if index < 1 || index > len(todos) {
			fmt.Printf("Index out of range: %d\n", index)
			os.Exit(1)
		}

		todo := todos[index-1]
		err = db.ToggleTodo(todo.ID)
		if err != nil {
			fmt.Printf("Error toggling todo: %v\n", err)
			os.Exit(1)
		}

		status := "done"
		if todo.Done {
			status = "undone"
		}
		fmt.Printf("Marked '%s' as %s\n", todo.Text, status)
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete [index]",
	Short: "Delete a todo item",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		db, err := NewDatabase()
		if err != nil {
			fmt.Printf("Error opening database: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		index, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Printf("Invalid index: %s\n", args[0])
			os.Exit(1)
		}

		todos, err := db.GetTodos()
		if err != nil {
			fmt.Printf("Error getting todos: %v\n", err)
			os.Exit(1)
		}

		if index < 1 || index > len(todos) {
			fmt.Printf("Index out of range: %d\n", index)
			os.Exit(1)
		}

		todo := todos[index-1]
		err = db.DeleteTodo(todo.ID)
		if err != nil {
			fmt.Printf("Error deleting todo: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Deleted: %s\n", todo.Text)
	},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a local todo database in current directory",
	Long:  "Creates a .todos directory in the current directory for project-specific todos",
	Run: func(cmd *cobra.Command, args []string) {
		err := InitLocalDatabase()
		if err != nil {
			fmt.Printf("Error initializing local database: %v\n", err)
			os.Exit(1)
		}

		cwd, _ := os.Getwd()
		fmt.Printf("Initialized local todo database in %s/.todos\n", cwd)
		fmt.Println("Local todos will now take precedence over global todos in this directory.")
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show which todo database is currently being used",
	Run: func(cmd *cobra.Command, args []string) {
		dbPath, err := getDatabasePath()
		if err != nil {
			fmt.Printf("Error getting database path: %v\n", err)
			os.Exit(1)
		}

		cwd, _ := os.Getwd()
		if filepath.Dir(dbPath) == filepath.Join(cwd, ".todos") {
			fmt.Printf("Using LOCAL todo database: %s\n", dbPath)
		} else {
			fmt.Printf("Using GLOBAL todo database: %s\n", dbPath)
		}

		if _, err := os.Stat(dbPath); err == nil {
			db, err := NewDatabase()
			if err != nil {
				fmt.Printf("Error opening database: %v\n", err)
				return
			}
			defer db.Close()

			todos, err := db.GetTodos()
			if err != nil {
				fmt.Printf("Error getting todos: %v\n", err)
				return
			}

			fmt.Printf("Total todos: %d\n", len(todos))
		} else {
			fmt.Println("Database file does not exist yet.")
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(getVersionInfo())
	},
}

var undoCmd = &cobra.Command{
	Use:   "undo",
	Short: "Undo the last deleted todo",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := NewDatabase()
		if err != nil {
			fmt.Printf("Error opening database: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		todo, err := db.UndoLastDelete()
		if err != nil {
			if err.Error() == "sql: no rows in result set" {
				fmt.Println("No recently deleted todos to restore")
			} else {
				fmt.Printf("Error undoing delete: %v\n", err)
			}
			os.Exit(1)
		}

		fmt.Printf("Restored: %s\n", todo.Text)
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(toggleCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(undoCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(versionCmd)
}

func Execute() error {
	return rootCmd.Execute()
}
