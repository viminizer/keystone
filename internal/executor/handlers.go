package executor

import (
	"strconv"
	"strings"
	"time"

	"github.com/viminizer/keystone/internal/engine"
	"github.com/viminizer/keystone/internal/resp"
)

func handlePing(db *engine.Engine, args []string) string {
	return resp.SimpleString("PONG")
}

func handleSet(db *engine.Engine, args []string) string {
	if len(args) < 2 {
		return wrongNumberOfArguments("set")
	}
	key := args[0]
	value := args[1]
	if len(args) == 2 {
		db.Set(key, value)
		return resp.SimpleString("OK")
	}
	if len(args) != 4 || strings.ToUpper(args[2]) != "EX" {
		return resp.Error("ERR syntax error")
	}
	seconds, err := strconv.Atoi(args[3])
	if err != nil {
		return resp.Error("ERR value is not an integer or out of range")
	}
	if seconds <= 0 {
		return resp.Error("ERR invalid expire time in 'set' command")
	}
	at := time.Now().Add(time.Duration(seconds) * time.Second)
	db.SetWithExpiry(key, value, at)
	return resp.SimpleString("OK")
}

func handleGet(db *engine.Engine, args []string) string {
	if len(args) != 1 {
		return wrongNumberOfArguments("get")
	}
	value, found := db.Get(args[0])
	if !found {
		return resp.Null()
	}
	return resp.BulkString(value)
}

func handleDel(db *engine.Engine, args []string) string {
	if len(args) < 1 {
		return wrongNumberOfArguments("del")
	}
	deleted := db.Delete(args)
	return resp.Integer(deleted)
}

func handleExpire(db *engine.Engine, args []string) string {
	if len(args) != 2 {
		return wrongNumberOfArguments("expire")
	}
	seconds, err := strconv.Atoi(args[1])
	if err != nil {
		return resp.Error("ERR value is not an integer or out of range")
	}
	at := time.Now().Add(time.Duration(seconds) * time.Second)
	if !db.ExpireAt(args[0], at) {
		return resp.Integer(0)
	}
	return resp.Integer(1)
}

func handlePexpireat(db *engine.Engine, args []string) string {
	if len(args) != 2 {
		return wrongNumberOfArguments("pexpireat")
	}
	// The time is in milliseconds since January 1, 1970 (Unix time).
	milliseconds, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return resp.Error("ERR value is not an integer or out of range")
	}
	if !db.ExpireAt(args[0], time.UnixMilli(milliseconds)) {
		return resp.Integer(0)
	}
	return resp.Integer(1)
}

func handleTTL(db *engine.Engine, args []string) string {
	if len(args) != 1 {
		return wrongNumberOfArguments("ttl")
	}
	expiresAt, found := db.ExpiresAt(args[0])
	if !found {
		return resp.Integer(-2)
	}
	if expiresAt.IsZero() {
		return resp.Integer(-1)
	}
	secondsLeft := time.Until(expiresAt).Round(time.Second).Seconds()
	return resp.Integer(int(secondsLeft))
}

func handleBgrewriteaof(db *engine.Engine, args []string) string {
	if len(args) != 0 {
		return wrongNumberOfArguments("bgrewriteaof")
	}
	if !db.StartRewrite() {
		return resp.Error("ERR Background append only file rewriting already in progress")
	}
	return resp.SimpleString("Background append only file rewriting started")
}

func wrongNumberOfArguments(name string) string {
	return resp.Error("ERR wrong number of arguments for '" + name + "' command")
}
