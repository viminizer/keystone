package main

import "testing"

func TestShouldRewrite(t *testing.T) {
	const megabyte = 1024 * 1024
	tests := []struct {
		size     int64
		baseSize int64
		want     bool
	}{
		{size: 100 * megabyte, baseSize: 50 * megabyte, want: true},
		{size: 99 * megabyte, baseSize: 50 * megabyte, want: false},
		{size: 64 * megabyte, baseSize: 0, want: true},
		{size: 63 * megabyte, baseSize: 0, want: false},
		{size: 10 * megabyte, baseSize: 1 * megabyte, want: false},
	}

	for _, test := range tests {
		got := shouldRewrite(test.size, test.baseSize)
		if got != test.want {
			t.Errorf("shouldRewrite(%d, %d) = %v, want %v", test.size, test.baseSize, got, test.want)
		}
	}
}
