// Package aof writes every change to an append-only file, and loads the
// file again after a restart.
package aof

import (
	"fmt"
	"os"
	"time"

	"github.com/viminizer/keystone/resp"
)

// Log is an open append-only file.
type Log struct {
	file *os.File
}

// Open opens the file for adding to its end. It creates the file if it
// does not exist.
func Open(path string) (*Log, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	return &Log{file: file}, nil
}

// Append writes one command to the end of the file. The operating system
// keeps it in memory for a short time, until RunSync puts it on the disk.
func (l *Log) Append(command []string) error {
	_, err := l.file.WriteString(resp.Command(command))
	return err
}

// RunSync puts the written commands on the disk once a second, so a
// machine crash loses at most about one second of changes.
// It never returns, so start it with go.
func (l *Log) RunSync() {
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		err := l.file.Sync()
		if err != nil {
			fmt.Println("could not sync the append-only file:", err)
		}
	}
}
