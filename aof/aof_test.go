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

func TestRewrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "appendonly.aof")
	log, _ := Open(path)
	log.Append([]string{"SET", "name", "kevin"})
	log.Append([]string{"SET", "name", "alex"})

	err := log.Rewrite([][]string{{"SET", "name", "alex"}})
	if err != nil {
		t.Fatal(err)
	}
	// New commands must go into the new file.
	log.Append([]string{"DEL", "name"})

	got, _ := os.ReadFile(path)
	want := "*3\r\n$3\r\nSET\r\n$4\r\nname\r\n$4\r\nalex\r\n" + "*2\r\n$3\r\nDEL\r\n$4\r\nname\r\n"
	if string(got) != want {
		t.Errorf("file = %q, want %q", got, want)
	}
	if _, err := os.Stat(path + ".tmp"); err == nil {
		t.Error("the temporary file was not renamed")
	}
}
