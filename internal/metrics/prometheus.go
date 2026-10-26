package metrics

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// WritePrometheus writes the numbers of every command in the Prometheus
// text format. Times are written in seconds, and the command name is a
// label in lowercase, like command="get".
func (c *CommandStats) WritePrometheus(writer io.Writer) error {
	snapshots := c.Snapshot()
	var text strings.Builder

	text.WriteString("# HELP keystone_command_duration_seconds Time it took to run each command.\n")
	text.WriteString("# TYPE keystone_command_duration_seconds histogram\n")
	for _, command := range snapshots {
		name := strings.ToLower(command.Name)
		// Prometheus wants each bucket to count everything up to its limit,
		// so the counts are added up on the way.
		upToLimit := int64(0)
		for i, limit := range command.Latency.Limits {
			upToLimit += command.Latency.Buckets[i]
			fmt.Fprintf(&text, "keystone_command_duration_seconds_bucket{command=%q,le=%q} %d\n",
				name, formatSeconds(limit.Seconds()), upToLimit)
		}
		fmt.Fprintf(&text, "keystone_command_duration_seconds_bucket{command=%q,le=\"+Inf\"} %d\n",
			name, command.Latency.Count)
		fmt.Fprintf(&text, "keystone_command_duration_seconds_sum{command=%q} %s\n",
			name, formatSeconds(command.Latency.Sum.Seconds()))
		fmt.Fprintf(&text, "keystone_command_duration_seconds_count{command=%q} %d\n",
			name, command.Latency.Count)
	}

	text.WriteString("# HELP keystone_command_errors_total Commands that replied with an error.\n")
	text.WriteString("# TYPE keystone_command_errors_total counter\n")
	for _, command := range snapshots {
		fmt.Fprintf(&text, "keystone_command_errors_total{command=%q} %d\n",
			strings.ToLower(command.Name), command.Errors)
	}

	_, err := io.WriteString(writer, text.String())
	return err
}

// formatSeconds writes a number of seconds as short as possible, like
// 1e-05 or 0.5.
func formatSeconds(seconds float64) string {
	return strconv.FormatFloat(seconds, 'g', -1, 64)
}
