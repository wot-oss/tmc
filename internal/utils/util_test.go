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

func TestParseAsList(t *testing.T) {

	tests := []struct {
		in   string
		sep  string
		trim bool
		out  []string
	}{
		{"1,2,3,4,5", ",", true, []string{"1", "2", "3", "4", "5"}},
		{"1,2,3,4,5,4,3,1", ",", true, []string{"1", "2", "3", "4", "5", "4", "3", "1"}},
		{"1,2,3,,4,,5", ",", true, []string{"1", "2", "3", "4", "5"}},
		{"1, 2 ,3 ,4,5 ", ",", true, []string{"1", "2", "3", "4", "5"}},
		{"1, 2 ,3 ,4,5 ", ",", false, []string{"1", " 2 ", "3 ", "4", "5 "}},
		{"1,2,3,4,5", "/", true, []string{"1,2,3,4,5"}},
	}

	for i, test := range tests {
		assert.Equal(t, test.out, ParseAsList(test.in, test.sep, test.trim), "in test %d", i)
	}
}
