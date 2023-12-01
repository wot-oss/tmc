//go:build windows

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
		{"ab\nc", "ab\r\nc"},
		{"ab\rc", "ab\rc"},
		{"\r\rab\r\rc\r\r", "\r\rab\r\rc\r\r"},
		{"\n\nab\n\nc\n\n", "\r\n\r\nab\r\n\r\nc\r\n\r\n"},
		{"ab\r\nc", "ab\r\r\nc"},
	}

	for i, test := range tests {
		assert.Equal(t, []byte(test.out), ConvertToNativeLineEndings([]byte(test.in)), "in test %d", i)
	}

}
