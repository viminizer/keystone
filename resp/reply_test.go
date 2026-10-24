package resp

import "testing"

func TestSimpleString(t *testing.T) {
	got := SimpleString("PONG")
	if got != "+PONG\r\n" {
		t.Errorf("SimpleString(\"PONG\") = %q", got)
	}
}

func TestError(t *testing.T) {
	got := Error("ERR unknown command 'FOO'")
	if got != "-ERR unknown command 'FOO'\r\n" {
		t.Errorf("Error(...) = %q", got)
	}
}
