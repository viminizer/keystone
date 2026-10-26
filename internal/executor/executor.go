// Package executor finds the handler for each command and runs it.
package executor

import (
	"strings"
	"time"

	"github.com/viminizer/keystone/internal/engine"
	"github.com/viminizer/keystone/internal/metrics"
	"github.com/viminizer/keystone/internal/resp"
)

// Handler runs one command.
type Handler struct {
	// Run gets the arguments without the command name, and returns the
	// reply to send.
	Run func(db *engine.Engine, args []string) string
}

// Executor finds the handler for a command and runs it.
type Executor struct {
	db *engine.Engine
	// commands is only changed by New, before any client connects.
	// After that it is only read, so it needs no lock.
	commands map[string]Handler
	metrics  *metrics.Commands
}

func New(db *engine.Engine, commandMetrics *metrics.Commands) *Executor {
	executor := &Executor{
		db:       db,
		commands: map[string]Handler{},
		metrics:  commandMetrics,
	}

	executor.Register("PING", handlePing)
	executor.Register("SET", handleSet)
	executor.Register("GET", handleGet)
	executor.Register("DEL", handleDel)
	executor.Register("EXPIRE", handleExpire)
	executor.Register("PEXPIREAT", handlePexpireat)
	executor.Register("TTL", handleTTL)
	executor.Register("BGREWRITEAOF", handleBgrewriteaof)

	return executor
}

// Register adds the handler for one command, and starts keeping metrics for
// it. The name must be in capitals.
func (e *Executor) Register(name string, run func(db *engine.Engine, args []string) string) {
	e.commands[name] = Handler{Run: run}
	e.metrics.Add(name)
}

// Execute runs one command and returns the reply to send. It records how
// long the command took and if the reply was an error.
// An empty command gets no reply, so it returns "".
func (e *Executor) Execute(command []string) string {
	if len(command) == 0 {
		return ""
	}

	name := strings.ToUpper(command[0])
	start := time.Now()
	reply := e.run(name, command)
	// Error replies in RESP start with "-".
	e.metrics.Record(name, time.Since(start), strings.HasPrefix(reply, "-"))
	return reply
}

// run finds the handler for the command and runs it.
func (e *Executor) run(name string, command []string) string {
	handler, found := e.commands[name]
	if !found {
		return resp.Error("ERR unknown command '" + command[0] + "'")
	}
	return handler.Run(e.db, command[1:])
}
