package health

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestImportMarkdownForUser_completes(t *testing.T) {
	md := filepath.Join("..", "mddata")
	if _, err := os.Stat(md); err != nil {
		t.Skip("mddata not present")
	}
	root := t.TempDir()
	done := make(chan error, 1)
	go func() {
		_, err := ImportMarkdownForUser(root, md, "pantazis")
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("import timed out (deadlock or DB lock)")
	}
}
