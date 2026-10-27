package metrics

import (
	"strings"
	"testing"
	"time"
)

func TestWritePrometheus(t *testing.T) {
	commandStats := NewCommandStats()
	commandStats.Add("GET")
	commandStats.Record("GET", 3*time.Microsecond, false)
	commandStats.Record("GET", 20*time.Microsecond, true)

	var output strings.Builder
	err := commandStats.WritePrometheus(&output)
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

func TestWritePrometheusPersistence(t *testing.T) {
	persistenceStats := NewPersistenceStats()
	persistenceStats.RecordFsync(2 * time.Millisecond)
	persistenceStats.SetSize(1024)
	persistenceStats.RecordRewrite(3 * time.Second)
	persistenceStats.RecordRewrite(1500 * time.Millisecond)

	var output strings.Builder
	err := persistenceStats.WritePrometheus(&output)
	if err != nil {
		t.Fatal(err)
	}

	got := output.String()
	wantLines := []string{
		"# TYPE keystone_aof_fsync_duration_seconds histogram",
		`keystone_aof_fsync_duration_seconds_bucket{le="0.001"} 0`,
		`keystone_aof_fsync_duration_seconds_bucket{le="0.005"} 1`,
		`keystone_aof_fsync_duration_seconds_bucket{le="+Inf"} 1`,
		"keystone_aof_fsync_duration_seconds_sum 0.002",
		"keystone_aof_fsync_duration_seconds_count 1",
		"# TYPE keystone_aof_size_bytes gauge",
		"keystone_aof_size_bytes 1024",
		"# TYPE keystone_aof_rewrites_total counter",
		"keystone_aof_rewrites_total 2",
		"# TYPE keystone_aof_last_rewrite_duration_seconds gauge",
		"keystone_aof_last_rewrite_duration_seconds 1.5",
	}
	for _, line := range wantLines {
		if !strings.Contains(got, line+"\n") {
			t.Errorf("missing line %s in:\n%s", line, got)
		}
	}
}
