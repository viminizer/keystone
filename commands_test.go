package main

import "testing"

func TestRunCommand(t *testing.T) {
	tests := []struct {
		command []string
		want    string
	}{
		{[]string{"PING"}, "+PONG\r\n"},
		{[]string{"ping"}, "+PONG\r\n"},
		{[]string{"foo"}, "-ERR unknown command 'foo'\r\n"},
		{[]string{}, ""},
	}

	for _, test := range tests {
		got := runCommand(test.command)
		if got != test.want {
			t.Errorf("runCommand(%q) = %q, want %q", test.command, got, test.want)
		}
	}
}
