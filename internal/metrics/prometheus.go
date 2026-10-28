package metrics

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
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
		label := fmt.Sprintf("command=%q", strings.ToLower(command.Name))
		writeHistogram(&text, "keystone_command_duration_seconds", label, command.Latency)
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

// WritePrometheus writes the numbers about the append-only file in the
// Prometheus text format.
func (p *PersistenceStats) WritePrometheus(writer io.Writer) error {
	var text strings.Builder

	text.WriteString("# HELP keystone_aof_fsync_duration_seconds Time it took to put the append-only file on the disk.\n")
	text.WriteString("# TYPE keystone_aof_fsync_duration_seconds histogram\n")
	writeHistogram(&text, "keystone_aof_fsync_duration_seconds", "", p.fsyncLatency.Snapshot())

	text.WriteString("# HELP keystone_aof_size_bytes Size of the append-only file.\n")
	text.WriteString("# TYPE keystone_aof_size_bytes gauge\n")
	fmt.Fprintf(&text, "keystone_aof_size_bytes %d\n", p.sizeBytes.Load())

	text.WriteString("# HELP keystone_aof_rewrites_total Rewrites of the append-only file that worked.\n")
	text.WriteString("# TYPE keystone_aof_rewrites_total counter\n")
	fmt.Fprintf(&text, "keystone_aof_rewrites_total %d\n", p.rewrites.Load())

	text.WriteString("# HELP keystone_aof_last_rewrite_duration_seconds Time the last rewrite of the append-only file took.\n")
	text.WriteString("# TYPE keystone_aof_last_rewrite_duration_seconds gauge\n")
	lastRewrite := time.Duration(p.lastRewriteDuration.Load())
	fmt.Fprintf(&text, "keystone_aof_last_rewrite_duration_seconds %s\n", formatSeconds(lastRewrite.Seconds()))

	_, err := io.WriteString(writer, text.String())
	return err
}

// WritePrometheus writes the counts of expired keys in the Prometheus text
// format, with a label for how each key was found.
func (e *ExpiryStats) WritePrometheus(writer io.Writer) error {
	var text strings.Builder

	text.WriteString("# HELP keystone_expired_keys_total Keys deleted because their time ran out.\n")
	text.WriteString("# TYPE keystone_expired_keys_total counter\n")
	fmt.Fprintf(&text, "keystone_expired_keys_total{by=\"active\"} %d\n", e.active.Load())
	fmt.Fprintf(&text, "keystone_expired_keys_total{by=\"lazy\"} %d\n", e.lazy.Load())

	text.WriteString("# HELP keystone_keys_with_expiry Keys that have an expiry right now.\n")
	text.WriteString("# TYPE keystone_keys_with_expiry gauge\n")
	fmt.Fprintf(&text, "keystone_keys_with_expiry %d\n", e.keysWithExpiry.Load())

	_, err := io.WriteString(writer, text.String())
	return err
}

// writeHistogram writes the bucket, sum and count lines of one histogram.
// labels is like command="get", or empty when there are no labels.
func writeHistogram(text *strings.Builder, name string, labels string, histogram HistogramSnapshot) {
	bucketLabels := "le="
	otherLabels := ""
	if labels != "" {
		bucketLabels = labels + ",le="
		otherLabels = "{" + labels + "}"
	}

	// Prometheus wants each bucket to count everything up to its limit,
	// so the counts are added up on the way.
	upToLimit := int64(0)
	for i, limit := range histogram.Limits {
		upToLimit += histogram.Buckets[i]
		fmt.Fprintf(text, "%s_bucket{%s%q} %d\n", name, bucketLabels, formatSeconds(limit.Seconds()), upToLimit)
	}
	fmt.Fprintf(text, "%s_bucket{%s\"+Inf\"} %d\n", name, bucketLabels, histogram.Count)
	fmt.Fprintf(text, "%s_sum%s %s\n", name, otherLabels, formatSeconds(histogram.Sum.Seconds()))
	fmt.Fprintf(text, "%s_count%s %d\n", name, otherLabels, histogram.Count)
}

// formatSeconds writes a number of seconds as short as possible, like
// 1e-05 or 0.5.
func formatSeconds(seconds float64) string {
	return strconv.FormatFloat(seconds, 'g', -1, 64)
}
