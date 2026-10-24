package resp

// SimpleString returns a RESP simple string, like "+PONG\r\n".
func SimpleString(text string) string {
	return "+" + text + "\r\n"
}

// Error returns a RESP error, like "-ERR unknown command 'FOO'\r\n".
func Error(message string) string {
	return "-" + message + "\r\n"
}
