package resp

import (
	"strconv"
	"strings"
)

// SimpleString returns a RESP simple string, like "+PONG\r\n".
func SimpleString(text string) string {
	return "+" + text + "\r\n"
}

// Error returns a RESP error, like "-ERR unknown command 'FOO'\r\n".
func Error(message string) string {
	return "-" + message + "\r\n"
}

// BulkString returns a RESP bulk string, like "$5\r\nkevin\r\n".
func BulkString(text string) string {
	return "$" + strconv.Itoa(len(text)) + "\r\n" + text + "\r\n"
}

// Null returns the RESP null bulk string, which means "no value".
func Null() string {
	return "$-1\r\n"
}

// Integer returns a RESP integer, like ":1\r\n".
func Integer(number int) string {
	return ":" + strconv.Itoa(number) + "\r\n"
}

// Command returns a command as an array of bulk strings, the same format
// ReadCommand reads. For example ["PING"] becomes "*1\r\n$4\r\nPING\r\n".
func Command(parts []string) string {
	var builder strings.Builder
	builder.WriteString("*" + strconv.Itoa(len(parts)) + "\r\n")
	for _, part := range parts {
		builder.WriteString(BulkString(part))
	}
	return builder.String()
}
