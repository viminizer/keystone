package main

import (
	"strconv"
	"strings"
	"time"

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

	case "DEL":
		if len(command) < 2 {
			return wrongNumberOfArguments(command[0])
		}
		deleted := 0
		for _, key := range command[1:] {
			if db.Delete(key) {
				deleted++
			}
		}
		return resp.Integer(deleted)

	case "EXPIRE":
		if len(command) != 3 {
			return wrongNumberOfArguments(command[0])
		}
		seconds, err := strconv.Atoi(command[2])
		if err != nil {
			return resp.Error("ERR value is not an integer or out of range")
		}
		if db.Expire(command[1], time.Duration(seconds)*time.Second) {
			return resp.Integer(1)
		}
		return resp.Integer(0)

	case "TTL":
		if len(command) != 2 {
			return wrongNumberOfArguments(command[0])
		}
		expiresAt, found := db.ExpiresAt(command[1])
		if !found {
			return resp.Integer(-2)
		}
		if expiresAt.IsZero() {
			return resp.Integer(-1)
		}
		secondsLeft := time.Until(expiresAt).Round(time.Second).Seconds()
		return resp.Integer(int(secondsLeft))

	default:
		return resp.Error("ERR unknown command '" + command[0] + "'")
	}
}

func wrongNumberOfArguments(name string) string {
	return resp.Error("ERR wrong number of arguments for '" + strings.ToLower(name) + "' command")
}
