package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/viminizer/keystone/aof"
	"github.com/viminizer/keystone/resp"
	"github.com/viminizer/keystone/store"
)

// writeMutex is held while a command changes the store and writes to the
// log. So the commands in the log are always in the same order as they
// changed the store.
var writeMutex sync.Mutex

// runCommand runs one command and returns the reply to send.
// An empty command gets no reply, so it returns "".
// Changes are written to aofLog. A nil aofLog means they are not written.
func runCommand(command []string, db *store.Store, aofLog *aof.Log) string {
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
			writeMutex.Lock()
			db.Set(key, value)
			appendToLog(aofLog, []string{"SET", key, value})
			writeMutex.Unlock()
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
		writeMutex.Lock()
		db.SetWithExpiry(key, value, at)
		appendToLog(aofLog, []string{"SET", key, value})
		appendToLog(aofLog, []string{"PEXPIREAT", key, unixMilli(at)})
		writeMutex.Unlock()
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
		keys := command[1:]
		writeMutex.Lock()
		deleted := 0
		for _, key := range keys {
			if db.Delete(key) {
				deleted++
			}
		}
		if deleted > 0 {
			appendToLog(aofLog, command)
		}
		writeMutex.Unlock()
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
		return expireAt(command[1], at, db, aofLog)

	case "PEXPIREAT":
		if len(command) != 3 {
			return wrongNumberOfArguments(command[0])
		}
		// The time is in milliseconds since January 1, 1970 (Unix time).
		milliseconds, err := strconv.ParseInt(command[2], 10, 64)
		if err != nil {
			return resp.Error("ERR value is not an integer or out of range")
		}
		return expireAt(command[1], time.UnixMilli(milliseconds), db, aofLog)

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

// expireAt runs EXPIRE and PEXPIREAT. Both are written to the log as
// PEXPIREAT, because a moment in time stays correct after a restart,
// and "10 seconds from now" does not.
func expireAt(key string, at time.Time, db *store.Store, aofLog *aof.Log) string {
	writeMutex.Lock()
	defer writeMutex.Unlock()

	if !db.ExpireAt(key, at) {
		return resp.Integer(0)
	}
	appendToLog(aofLog, []string{"PEXPIREAT", key, unixMilli(at)})
	return resp.Integer(1)
}

// appendToLog writes one command to the log. If that fails, the change is
// still in memory, so the error is only printed.
func appendToLog(aofLog *aof.Log, command []string) {
	if aofLog == nil {
		return
	}
	err := aofLog.Append(command)
	if err != nil {
		fmt.Println("could not write to the append-only file:", err)
	}
}

func unixMilli(at time.Time) string {
	return strconv.FormatInt(at.UnixMilli(), 10)
}

func wrongNumberOfArguments(name string) string {
	return resp.Error("ERR wrong number of arguments for '" + strings.ToLower(name) + "' command")
}
