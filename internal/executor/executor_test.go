package executor

import (
	"testing"

	"github.com/viminizer/keystone/internal/metrics"
)

func TestExecuteRecordsMetrics(t *testing.T) {
	executor, _ := newTestExecutor(t)
	executor.Execute([]string{"SET", "name", "kevin"})
	executor.Execute([]string{"get", "name"})
	executor.Execute([]string{"GET"})
	executor.Execute([]string{"FOO"})
	executor.Execute([]string{})

	want := map[string][2]int64{
		"GET":                  {2, 1},
		"SET":                  {1, 0},
		"PING":                 {0, 0},
		metrics.UnknownCommand: {1, 1},
	}
	got := map[string]metrics.CommandSnapshot{}
	for _, command := range executor.metrics.Snapshot() {
		got[command.Name] = command
	}
	for name, wanted := range want {
		command := got[name]
		if command.Calls != wanted[0] || command.Errors != wanted[1] {
			t.Errorf("%s: %d calls %d errors, want %d calls %d errors",
				name, command.Calls, command.Errors, wanted[0], wanted[1])
		}
	}
}
