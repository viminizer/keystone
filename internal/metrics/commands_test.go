package metrics

import (
	"testing"
	"time"
)

func TestCommands(t *testing.T) {
	commands := NewCommands()
	commands.Add("GET")
	commands.Add("SET")

	commands.Record("GET", time.Microsecond, false)
	commands.Record("GET", time.Microsecond, true)
	commands.Record("SET", time.Microsecond, false)
	commands.Record("FOO", time.Microsecond, true)
	commands.Record("BAR", time.Microsecond, true)

	want := []CommandSnapshot{
		{Name: "GET", Calls: 2, Errors: 1},
		{Name: "SET", Calls: 1, Errors: 0},
		{Name: UnknownCommand, Calls: 2, Errors: 2},
	}
	got := commands.Snapshot()
	if len(got) != len(want) {
		t.Fatalf("Snapshot has %d commands, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Name != want[i].Name || got[i].Calls != want[i].Calls || got[i].Errors != want[i].Errors {
			t.Errorf("command %d = %s %d calls %d errors, want %s %d calls %d errors",
				i, got[i].Name, got[i].Calls, got[i].Errors, want[i].Name, want[i].Calls, want[i].Errors)
		}
	}
}
