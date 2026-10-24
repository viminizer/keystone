package main

import (
	"strings"

	"github.com/viminizer/keystone/resp"
	"github.com/viminizer/keystone/store"
)

// runCommand runs one command and returns the reply to send.
// An empty command gets no reply, so it returns "".
func runCommand(command []string, db *store.Store) string {
	if len(command) == 0 {
		return ""
	}

	name := strings.ToUpper(command[0])
	switch name {
	case "PING":
		return resp.SimpleString("PONG")

	case "SET":
		if len(command) != 3 {
			return wrongNumberOfArguments(command[0])
		}
		db.Set(command[1], command[2])
		return resp.SimpleString("OK")

	case "GET":
		if len(command) != 2 {
			return wrongNumberOfArguments(command[0])
		}
		value, found := db.Get(command[1])
		if !found {
			return resp.Null()
		}
		return resp.BulkString(value)

	default:
		return resp.Error("ERR unknown command '" + command[0] + "'")
	}
}

func wrongNumberOfArguments(name string) string {
	return resp.Error("ERR wrong number of arguments for '" + strings.ToLower(name) + "' command")
}
