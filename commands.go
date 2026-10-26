package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/viminizer/keystone/resp"
)

// runCommand runs one command and returns the reply to send.
// An empty command gets no reply, so it returns "".
func runCommand(command []string, engine *Engine) string {
	if len(command) == 0 {
		return ""
	}

	name := strings.ToUpper(command[0])
	switch name {
	case "PING":
		return resp.SimpleString("PONG")

	case "SET":
		if len(command) < 3 {
			return wrongNumberOfArguments(command[0])
		}
		key := command[1]
		value := command[2]
		if len(command) == 3 {
			engine.Set(key, value)
			return resp.SimpleString("OK")
		}
		if len(command) != 5 || strings.ToUpper(command[3]) != "EX" {
			return resp.Error("ERR syntax error")
		}
		seconds, err := strconv.Atoi(command[4])
		if err != nil {
			return resp.Error("ERR value is not an integer or out of range")
		}
		if seconds <= 0 {
			return resp.Error("ERR invalid expire time in 'set' command")
		}
		at := time.Now().Add(time.Duration(seconds) * time.Second)
		engine.SetWithExpiry(key, value, at)
		return resp.SimpleString("OK")

	case "GET":
		if len(command) != 2 {
			return wrongNumberOfArguments(command[0])
		}
		value, found := engine.Get(command[1])
		if !found {
			return resp.Null()
		}
		return resp.BulkString(value)

	case "DEL":
		if len(command) < 2 {
			return wrongNumberOfArguments(command[0])
		}
		deleted := engine.Delete(command[1:])
		return resp.Integer(deleted)

	case "EXPIRE":
		if len(command) != 3 {
			return wrongNumberOfArguments(command[0])
		}
		seconds, err := strconv.Atoi(command[2])
		if err != nil {
			return resp.Error("ERR value is not an integer or out of range")
		}
		at := time.Now().Add(time.Duration(seconds) * time.Second)
		if !engine.ExpireAt(command[1], at) {
			return resp.Integer(0)
		}
		return resp.Integer(1)

	case "PEXPIREAT":
		if len(command) != 3 {
			return wrongNumberOfArguments(command[0])
		}
		// The time is in milliseconds since January 1, 1970 (Unix time).
		milliseconds, err := strconv.ParseInt(command[2], 10, 64)
		if err != nil {
			return resp.Error("ERR value is not an integer or out of range")
		}
		if !engine.ExpireAt(command[1], time.UnixMilli(milliseconds)) {
			return resp.Integer(0)
		}
		return resp.Integer(1)

	case "TTL":
		if len(command) != 2 {
			return wrongNumberOfArguments(command[0])
		}
		expiresAt, found := engine.ExpiresAt(command[1])
		if !found {
			return resp.Integer(-2)
		}
		if expiresAt.IsZero() {
			return resp.Integer(-1)
		}
		secondsLeft := time.Until(expiresAt).Round(time.Second).Seconds()
		return resp.Integer(int(secondsLeft))

	case "BGREWRITEAOF":
		if len(command) != 1 {
			return wrongNumberOfArguments(command[0])
		}
		if !engine.StartRewrite() {
			return resp.Error("ERR Background append only file rewriting already in progress")
		}
		return resp.SimpleString("Background append only file rewriting started")

	default:
		return resp.Error("ERR unknown command '" + command[0] + "'")
	}
}

func wrongNumberOfArguments(name string) string {
	return resp.Error("ERR wrong number of arguments for '" + strings.ToLower(name) + "' command")
}
