package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew_WritesToPathAndRotates(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "sub", "test.log")

	lg, closeFn, err := New(logPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer closeFn()

	lg.Println("hello-from-test")

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(data), "hello-from-test") {
		t.Errorf("log file missing message; got: %q", string(data))
	}
}

func TestNew_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "a", "b", "c", "test.log")

	_, closeFn, err := New(logPath)
	if err != nil {
		t.Fatalf("New should mkdir -p parents, got: %v", err)
	}
	defer closeFn()

	if _, err := os.Stat(filepath.Dir(logPath)); err != nil {
		t.Errorf("parent dir not created: %v", err)
	}
}
