//go:build !windows

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertToNativeLineEndings(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{"", ""},
		{"abc", "abc"},
		{"ab\nc", "ab\nc"},
		{"ab\rc", "ab\rc"},
		{"\r\rab\r\rc\r\r", "\r\rab\r\rc\r\r"},
		{"\n\nab\n\nc\n\n", "\n\nab\n\nc\n\n"},
		{"ab\r\nc", "ab\r\nc"},
	}

	for i, test := range tests {
		assert.Equal(t, []byte(test.out), ConvertToNativeLineEndings([]byte(test.in)), "in test %d", i)
	}

}
