package main

import (
	"strings"

	"github.com/viminizer/keystone/internal/resp"
)

// Handler runs one command.
type Handler struct {
	// Run gets the arguments without the command name, and returns the
	// reply to send.
	Run func(engine *Engine, args []string) string
}

// Executor finds the handler for a command and runs it.
type Executor struct {
	engine *Engine
	// commands is only changed by NewExecutor, before any client connects.
	// After that it is only read, so it needs no lock.
	commands map[string]Handler
}

func NewExecutor(engine *Engine) *Executor {
	executor := &Executor{
		engine:   engine,
		commands: map[string]Handler{},
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

// Register adds the handler for one command. The name must be in capitals.
func (e *Executor) Register(name string, run func(engine *Engine, args []string) string) {
	e.commands[name] = Handler{Run: run}
}

// Execute runs one command and returns the reply to send.
// An empty command gets no reply, so it returns "".
func (e *Executor) Execute(command []string) string {
	if len(command) == 0 {
		return ""
	}

	handler, found := e.commands[strings.ToUpper(command[0])]
	if !found {
		return resp.Error("ERR unknown command '" + command[0] + "'")
	}
	return handler.Run(e.engine, command[1:])
}
