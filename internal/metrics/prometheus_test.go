package metrics

import (
	"strings"
	"testing"
	"time"
)

func TestWritePrometheus(t *testing.T) {
	commands := NewCommands()
	commands.Add("GET")
	commands.Record("GET", 3*time.Microsecond, false)
	commands.Record("GET", 20*time.Microsecond, true)

	var output strings.Builder
	err := commands.WritePrometheus(&output)
	if err != nil {
		t.Fatal(err)
	}

	got := output.String()
	wantLines := []string{
		"# TYPE keystone_command_duration_seconds histogram",
		`keystone_command_duration_seconds_bucket{command="get",le="1e-05"} 1`,
		`keystone_command_duration_seconds_bucket{command="get",le="5e-05"} 2`,
		`keystone_command_duration_seconds_bucket{command="get",le="1"} 2`,
		`keystone_command_duration_seconds_bucket{command="get",le="+Inf"} 2`,
		`keystone_command_duration_seconds_sum{command="get"} 2.3e-05`,
		`keystone_command_duration_seconds_count{command="get"} 2`,
		`keystone_command_duration_seconds_count{command="unknown"} 0`,
		"# TYPE keystone_command_errors_total counter",
		`keystone_command_errors_total{command="get"} 1`,
		`keystone_command_errors_total{command="unknown"} 0`,
	}
	for _, line := range wantLines {
		if !strings.Contains(got, line+"\n") {
			t.Errorf("missing line %s in:\n%s", line, got)
		}
	}
}
