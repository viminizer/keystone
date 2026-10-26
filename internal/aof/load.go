package aof

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/viminizer/keystone/internal/resp"
)

// Load reads the file and calls run for every command in it, in order.
// It returns how many commands it ran. A missing file means there is
// nothing to load yet.
//
// If the file ends in the middle of a command, the server stopped while
// writing it. Load runs everything before it and cuts the broken end off,
// so new commands are not added after it. Any other broken command is an
// error, because the file was damaged in some other way.
func Load(path string, run func(command []string)) (int, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	loaded := 0
	// goodBytes is where the last complete command ends.
	goodBytes := 0
	for {
		command, err := resp.ReadCommand(reader)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return loaded, fmt.Errorf("broken command after byte %d: %s", goodBytes, err)
		}
		run(command)
		loaded++
		// The file holds only commands written by resp.Command, so this is
		// exactly how many bytes the command took in the file.
		goodBytes += len(resp.Command(command))
	}

	info, err := file.Stat()
	if err != nil {
		return loaded, err
	}
	brokenBytes := info.Size() - int64(goodBytes)
	if brokenBytes > 0 {
		slog.Warn("the append-only file ends with a broken command, cutting it off", "bytes", brokenBytes)
		err = os.Truncate(path, int64(goodBytes))
		if err != nil {
			return loaded, err
		}
	}
	return loaded, nil
}
