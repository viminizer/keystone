package resp

import "strconv"

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
