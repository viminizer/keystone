package resp

import (
	"bufio"
	"slices"
	"strings"
	"testing"
	"testing/iotest"
)

func TestReadCommand(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"*1\r\n$4\r\nPING\r\n", []string{"PING"}},
		{"*3\r\n$3\r\nSET\r\n$4\r\nname\r\n$5\r\nkevin\r\n", []string{"SET", "name", "kevin"}},
		{"*2\r\n$4\r\nECHO\r\n$0\r\n\r\n", []string{"ECHO", ""}},
		{"*2\r\n$4\r\nECHO\r\n$4\r\na\r\nb\r\n", []string{"ECHO", "a\r\nb"}},
	}

	for _, test := range tests {
		reader := bufio.NewReader(strings.NewReader(test.input))
		got, err := ReadCommand(reader)
		if err != nil {
			t.Errorf("ReadCommand(%q) returned error: %s", test.input, err)
			continue
		}
		if !slices.Equal(got, test.want) {
			t.Errorf("ReadCommand(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestReadCommandTwoCommandsInOneInput(t *testing.T) {
	input := "*1\r\n$4\r\nPING\r\n*2\r\n$3\r\nGET\r\n$4\r\nname\r\n"
	reader := bufio.NewReader(strings.NewReader(input))

	first, err := ReadCommand(reader)
	if err != nil || !slices.Equal(first, []string{"PING"}) {
		t.Fatalf("first command = %q, %v", first, err)
	}
	second, err := ReadCommand(reader)
	if err != nil || !slices.Equal(second, []string{"GET", "name"}) {
		t.Fatalf("second command = %q, %v", second, err)
	}
}

func TestReadCommandOneByteAtATime(t *testing.T) {
	input := "*3\r\n$3\r\nSET\r\n$4\r\nname\r\n$5\r\nkevin\r\n"
	// Each Read gives only 1 byte, like a command split over many chunks.
	reader := bufio.NewReader(iotest.OneByteReader(strings.NewReader(input)))

	got, err := ReadCommand(reader)
	if err != nil || !slices.Equal(got, []string{"SET", "name", "kevin"}) {
		t.Fatalf("ReadCommand = %q, %v", got, err)
	}
}

func TestReadCommandErrors(t *testing.T) {
	inputs := []string{
		"PING\r\n",
		"*x\r\n",
		"*-1\r\n",
		"*1\r\nPING\r\n",
		"*1\r\n$x\r\nPING\r\n",
		"*1\r\n$-1\r\n",
		"*1\r\n$4\r\nPINGxx",
		"*1\r\n$4\r\nPI",
	}

	for _, input := range inputs {
		reader := bufio.NewReader(strings.NewReader(input))
		_, err := ReadCommand(reader)
		if err == nil {
			t.Errorf("ReadCommand(%q) returned no error", input)
		}
	}
}

func TestReadCommandBulkStringTooBig(t *testing.T) {
	// 536870913 is one byte more than 512 MB.
	input := "*1\r\n$536870913\r\n"
	reader := bufio.NewReader(strings.NewReader(input))
	_, err := ReadCommand(reader)
	if err == nil || !strings.Contains(err.Error(), "too big") {
		t.Errorf("ReadCommand(%q) returned %v, want a too big error", input, err)
	}
}
