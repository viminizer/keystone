package aof

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "appendonly.aof")
	log, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	log.Append([]string{"SET", "name", "kevin"})
	log.Append([]string{"DEL", "name"})

	got, _ := os.ReadFile(path)
	want := "*3\r\n$3\r\nSET\r\n$4\r\nname\r\n$5\r\nkevin\r\n" + "*2\r\n$3\r\nDEL\r\n$4\r\nname\r\n"
	if string(got) != want {
		t.Errorf("file = %q, want %q", got, want)
	}
}

func TestOpenKeepsOldCommands(t *testing.T) {
	path := filepath.Join(t.TempDir(), "appendonly.aof")

	first, _ := Open(path)
	first.Append([]string{"PING"})
	second, _ := Open(path)
	second.Append([]string{"PING"})

	got, _ := os.ReadFile(path)
	want := "*1\r\n$4\r\nPING\r\n*1\r\n$4\r\nPING\r\n"
	if string(got) != want {
		t.Errorf("file = %q, want %q", got, want)
	}
}
