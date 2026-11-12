package main

import (
	"fmt"
	"os"
	"path/filepath"

	"earapi/health"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	path := filepath.Join(root, "mddata", "Health Tracker - Food Data.md")
	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read: %v\n", err)
		os.Exit(1)
	}
	out := health.RewriteFoodDataMarkdown(string(b))
	if err := os.WriteFile(path, []byte(out), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("rewrote", path)
}
