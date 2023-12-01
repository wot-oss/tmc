package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeLineEndings(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{"", ""},
		{"abc", "abc"},
		{"ab\nc", "ab\nc"},
		{"ab\rc", "ab\nc"},
		{"\r\rab\r\rc\r\r", "\n\nab\n\nc\n\n"},
		{"\n\nab\n\nc\n\n", "\n\nab\n\nc\n\n"},
		{"ab\r\nc", "ab\nc"},
		{"\r\n\r\nab\r\n\r\nc\r\n\r\n", "\n\nab\n\nc\n\n"},
		{"\r\rab\r\r\nc\r\n\n", "\n\nab\n\nc\n\n"},
	}

	for i, test := range tests {
		assert.Equal(t, []byte(test.out), NormalizeLineEndings([]byte(test.in)), "in test %d", i)
	}
}
