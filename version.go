package main

import "fmt"

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func getVersionInfo() string {
	return fmt.Sprintf("kaj version %s (commit: %s, built: %s)", Version, Commit, Date)
}
