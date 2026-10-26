package main

import (
	"strings"

	"github.com/viminizer/keystone/resp"
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
	// commands is only read after NewExecutor, so it needs no lock.
	commands map[string]Handler
}

func NewExecutor(engine *Engine) *Executor {
	return &Executor{
		engine: engine,
		commands: map[string]Handler{
			"PING":         {Run: handlePing},
			"SET":          {Run: handleSet},
			"GET":          {Run: handleGet},
			"DEL":          {Run: handleDel},
			"EXPIRE":       {Run: handleExpire},
			"PEXPIREAT":    {Run: handlePexpireat},
			"TTL":          {Run: handleTTL},
			"BGREWRITEAOF": {Run: handleBgrewriteaof},
		},
	}
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
