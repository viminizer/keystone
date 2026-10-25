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

func TestBulkString(t *testing.T) {
	tests := []struct {
		text string
		want string
	}{
		{"kevin", "$5\r\nkevin\r\n"},
		{"", "$0\r\n\r\n"},
		{"a\r\nb", "$4\r\na\r\nb\r\n"},
	}

	for _, test := range tests {
		got := BulkString(test.text)
		if got != test.want {
			t.Errorf("BulkString(%q) = %q, want %q", test.text, got, test.want)
		}
	}
}

func TestNull(t *testing.T) {
	got := Null()
	if got != "$-1\r\n" {
		t.Errorf("Null() = %q", got)
	}
}

func TestInteger(t *testing.T) {
	got := Integer(2)
	if got != ":2\r\n" {
		t.Errorf("Integer(2) = %q", got)
	}
}

func TestCommand(t *testing.T) {
	got := Command([]string{"SET", "name", "kevin"})
	want := "*3\r\n$3\r\nSET\r\n$4\r\nname\r\n$5\r\nkevin\r\n"
	if got != want {
		t.Errorf("Command(...) = %q, want %q", got, want)
	}
}
