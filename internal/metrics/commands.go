package metrics

import (
	"slices"
	"strings"
	"sync/atomic"
	"time"
)

// UnknownCommand is the name every unknown command is counted under. It is
// in lowercase, so it can never be the same as a real command name. Without
// it, a client could send many made-up names and fill the memory.
const UnknownCommand = "unknown"

// Commands keeps the numbers for each command: how many replies were
// errors, and how long each run took. It is safe to use from many
// goroutines.
type Commands struct {
	// stats has one entry for each command name. Add only runs before any
	// client connects. After that it is only read, so it needs no lock.
	stats map[string]*commandStats
}

type commandStats struct {
	errors  atomic.Int64
	latency *Histogram
}

// CommandSnapshot is a copy of one command's numbers at one moment.
type CommandSnapshot struct {
	Name    string
	Calls   int64
	Errors  int64
	Latency HistogramSnapshot
}

func NewCommands() *Commands {
	commands := &Commands{stats: map[string]*commandStats{}}
	commands.Add(UnknownCommand)
	return commands
}

// Add starts keeping numbers for one command name. Call it only before any
// client connects.
func (c *Commands) Add(name string) {
	c.stats[name] = &commandStats{latency: NewHistogram()}
}

// Record adds one run of a command. A name that was never added is counted
// as UnknownCommand.
func (c *Commands) Record(name string, duration time.Duration, failed bool) {
	stats, found := c.stats[name]
	if !found {
		stats = c.stats[UnknownCommand]
	}
	if failed {
		stats.errors.Add(1)
	}
	stats.latency.Observe(duration)
}

// Snapshot returns a copy of the numbers of every command, sorted by name.
func (c *Commands) Snapshot() []CommandSnapshot {
	snapshots := []CommandSnapshot{}
	for name, stats := range c.stats {
		latency := stats.latency.Snapshot()
		snapshots = append(snapshots, CommandSnapshot{
			Name:    name,
			Calls:   latency.Count,
			Errors:  stats.errors.Load(),
			Latency: latency,
		})
	}
	slices.SortFunc(snapshots, func(a CommandSnapshot, b CommandSnapshot) int {
		return strings.Compare(a.Name, b.Name)
	})
	return snapshots
}
