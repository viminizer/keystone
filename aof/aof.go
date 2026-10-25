// Package aof writes every change to an append-only file, and loads the
// file again after a restart.
package aof

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/viminizer/keystone/resp"
)

// Log is an open append-only file.
type Log struct {
	path string
	// mutex protects file, because Rewrite replaces it while RunSync
	// and Append use it.
	mutex sync.Mutex
	file  *os.File
}

// Open opens the file for adding to its end. It creates the file if it
// does not exist.
func Open(path string) (*Log, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	return &Log{path: path, file: file}, nil
}

// Append writes one command to the end of the file. The operating system
// keeps it in memory for a short time, until RunSync puts it on the disk.
func (l *Log) Append(command []string) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	_, err := l.file.WriteString(resp.Command(command))
	return err
}

// RunSync puts the written commands on the disk once a second, so a
// machine crash loses at most about one second of changes.
// It never returns, so start it with go.
func (l *Log) RunSync() {
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		// The sync is slow, so it happens outside the lock. Then Append
		// does not have to wait for it.
		l.mutex.Lock()
		file := l.file
		l.mutex.Unlock()

		err := file.Sync()
		// Rewrite may have closed this file in the meantime. The new
		// file is synced by Rewrite, so that error can be ignored.
		if err != nil && !errors.Is(err, os.ErrClosed) {
			fmt.Println("could not sync the append-only file:", err)
		}
	}
}

// Rewrite replaces the whole file with the given commands.
//
// The commands are written to a temporary file first, which is then
// renamed to the real name. A rename is atomic: after a crash the file is
// always the old one or the new one, never a mix of both.
//
// The caller must make sure no Append happens while Rewrite runs, or that
// command would go into the old file and be lost.
func (l *Log) Rewrite(commands [][]string) error {
	tempPath := l.path + ".tmp"
	temp, err := os.Create(tempPath)
	if err != nil {
		return err
	}

	err = writeAndSync(temp, commands)
	if err == nil {
		err = os.Rename(tempPath, l.path)
	}
	if err != nil {
		temp.Close()
		os.Remove(tempPath)
		return err
	}

	// The temporary file now has the real name, so new commands go there.
	l.mutex.Lock()
	old := l.file
	l.file = temp
	l.mutex.Unlock()
	old.Close()
	return nil
}

// writeAndSync writes the commands to the file and puts them on the disk.
func writeAndSync(file *os.File, commands [][]string) error {
	// A bufio.Writer collects many small writes in memory and sends them
	// to the file in big pieces, which is much faster.
	writer := bufio.NewWriter(file)
	for _, command := range commands {
		writer.WriteString(resp.Command(command))
	}
	// If any write above failed, Flush returns that error.
	err := writer.Flush()
	if err != nil {
		return err
	}
	return file.Sync()
}
