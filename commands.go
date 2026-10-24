package main

import (
	"strings"

	"github.com/viminizer/keystone/resp"
)

// runCommand runs one command and returns the reply to send.
// An empty command gets no reply, so it returns "".
func runCommand(command []string) string {
	if len(command) == 0 {
		return ""
	}

	name := strings.ToUpper(command[0])
	if name == "PING" {
		return resp.SimpleString("PONG")
	}

	return resp.Error("ERR unknown command '" + command[0] + "'")
}
