// Package resp reads commands and writes replies in RESP, the protocol Redis clients use.
package resp

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// maxBulkLength is the biggest bulk string we accept: 512 MB, the same as
// Redis. The memory for a value is reserved before the value arrives, so
// without a limit one client could make the server reserve all memory.
const maxBulkLength = 512 * 1024 * 1024

// ReadCommand reads one command, sent as an array of bulk strings.
// For example "*1\r\n$4\r\nPING\r\n" becomes ["PING"].
func ReadCommand(reader *bufio.Reader) ([]string, error) {
	line, err := readLine(reader)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(line, "*") {
		return nil, fmt.Errorf("expected an array, got %q", line)
	}
	count, err := strconv.Atoi(line[1:])
	if err != nil || count < 0 {
		return nil, fmt.Errorf("invalid array length %q", line)
	}

	command := []string{}
	for range count {
		line, err := readLine(reader)
		if err != nil {
			return nil, err
		}
		if !strings.HasPrefix(line, "$") {
			return nil, fmt.Errorf("expected a bulk string, got %q", line)
		}
		length, err := strconv.Atoi(line[1:])
		if err != nil || length < 0 {
			return nil, fmt.Errorf("invalid bulk string length %q", line)
		}
		if length > maxBulkLength {
			return nil, fmt.Errorf("bulk string length %d is too big", length)
		}

		// The value is followed by "\r\n", so read 2 more bytes.
		value := make([]byte, length+2)
		_, err = io.ReadFull(reader, value)
		if err != nil {
			return nil, err
		}
		if string(value[length:]) != "\r\n" {
			return nil, fmt.Errorf("bulk string does not end with \\r\\n")
		}

		command = append(command, string(value[:length]))
	}

	return command, nil
}

// readLine reads one line and removes the "\r\n" at the end.
func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(line, "\r\n"), nil
}
