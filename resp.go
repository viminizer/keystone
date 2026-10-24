package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// readCommand reads one command, sent as an array of bulk strings.
// For example "*1\r\n$4\r\nPING\r\n" becomes ["PING"].
func readCommand(reader *bufio.Reader) ([]string, error) {
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
