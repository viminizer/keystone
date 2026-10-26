// Package aof writes every change to an append-only file, and loads the
// file again after a restart.
package aof

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/viminizer/keystone/internal/resp"
)

// Log is an open append-only file.
type Log struct {
	path string
	// mutex protects file, size and baseSize, because Rewrite changes them
	// while Sync, Append and Sizes use them.
	mutex sync.Mutex
	file  *os.File
	// size is the size of the file now, in bytes.
	size int64
	// baseSize is the size after the last rewrite, or at Open.
	baseSize int64
}

// Open opens the file for adding to its end. It creates the file if it
// does not exist.
func Open(path string) (*Log, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	size := info.Size()
	return &Log{path: path, file: file, size: size, baseSize: size}, nil
}

// Append writes one command to the end of the file. The operating system
// keeps it in memory for a short time, until Sync puts it on the disk.
func (l *Log) Append(command []string) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	written, err := l.file.WriteString(resp.Command(command))
	l.size += int64(written)
	return err
}

// Sizes returns the size of the file now, and its size after the last
// rewrite (or at Open), both in bytes.
func (l *Log) Sizes() (int64, int64) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	return l.size, l.baseSize
}

// Sync puts the written commands on the disk.
func (l *Log) Sync() error {
	// The sync is slow, so it happens outside the lock. Then Append does
	// not have to wait for it.
	l.mutex.Lock()
	file := l.file
	l.mutex.Unlock()

	err := file.Sync()
	// Rewrite may have closed this file in the meantime. The new file is
	// synced by Rewrite, so that error can be ignored.
	if errors.Is(err, os.ErrClosed) {
		return nil
	}
	return err
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

	size, err := writeAndSync(temp, commands)
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
	l.size = size
	l.baseSize = size
	l.mutex.Unlock()
	old.Close()

	// The rename changed the folder, not the file. The folder has to be
	// synced too, or a power loss can undo the rename.
	folder, err := os.Open(filepath.Dir(l.path))
	if err != nil {
		return err
	}
	err = folder.Sync()
	folder.Close()
	return err
}

// writeAndSync writes the commands to the file and puts them on the disk.
// It returns the number of bytes written.
func writeAndSync(file *os.File, commands [][]string) (int64, error) {
	// A bufio.Writer collects many small writes in memory and sends them
	// to the file in big pieces, which is much faster.
	writer := bufio.NewWriter(file)
	var size int64
	for _, command := range commands {
		written, _ := writer.WriteString(resp.Command(command))
		size += int64(written)
	}
	// If any write above failed, Flush returns that error.
	err := writer.Flush()
	if err != nil {
		return 0, err
	}
	return size, file.Sync()
}
