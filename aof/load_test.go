package aof

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// loadAll loads the file and returns every command it ran.
func loadAll(t *testing.T, path string) [][]string {
	var commands [][]string
	_, err := Load(path, func(command []string) {
		commands = append(commands, command)
	})
	if err != nil {
		t.Fatalf("Load returned error: %s", err)
	}
	return commands
}

func TestLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "appendonly.aof")
	log, _ := Open(path)
	log.Append([]string{"SET", "name", "kevin"})
	log.Append([]string{"DEL", "name"})

	got := loadAll(t, path)
	if len(got) != 2 || !slices.Equal(got[0], []string{"SET", "name", "kevin"}) || !slices.Equal(got[1], []string{"DEL", "name"}) {
		t.Errorf("loaded %q", got)
	}
}

func TestLoadMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "appendonly.aof")

	loaded, err := Load(path, func(command []string) {})
	if loaded != 0 || err != nil {
		t.Errorf("Load = %d, %v, want 0 and no error", loaded, err)
	}
}

func TestLoadCutsBrokenEnd(t *testing.T) {
	path := filepath.Join(t.TempDir(), "appendonly.aof")
	good := "*3\r\n$3\r\nSET\r\n$4\r\nname\r\n$5\r\nkevin\r\n"
	// The server stopped in the middle of writing the second command.
	os.WriteFile(path, []byte(good+"*2\r\n$3\r\nDE"), 0644)

	got := loadAll(t, path)
	if len(got) != 1 {
		t.Fatalf("loaded %q, want only the complete command", got)
	}
	file, _ := os.ReadFile(path)
	if string(file) != good {
		t.Fatalf("file after Load = %q, want the broken end cut off", file)
	}

	// New commands go right after the good part, so the file is still valid.
	log, _ := Open(path)
	log.Append([]string{"DEL", "name"})
	if got := loadAll(t, path); len(got) != 2 {
		t.Errorf("loaded %q after adding a command, want 2 commands", got)
	}
}

func TestLoadBrokenMiddle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "appendonly.aof")
	os.WriteFile(path, []byte("hello\r\n*1\r\n$4\r\nPING\r\n"), 0644)

	_, err := Load(path, func(command []string) {})
	if err == nil {
		t.Error("Load on a damaged file returned no error")
	}
}
