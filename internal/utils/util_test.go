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

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		in  string
		exp string
	}{
		{in: "", exp: ""},
		{in: " ", exp: ""},
		{in: "&", exp: "-"},
		{in: "_", exp: "-"},
		{in: "=", exp: "-"},
		{in: "+", exp: "-"},
		{in: ":", exp: "-"},
		{in: "/", exp: "-"},
		{in: "a b&c=D+e:f/G", exp: "a-b-c-d-e-f-g"},
		{in: "a++:b", exp: "a-b"},
		{in: "a//b", exp: "a-b"},
		{in: "//a/b", exp: "-a-b"},
		{in: "a\\b", exp: "ab"},
		{in: "a#b", exp: "ab"},
		{in: " a b ", exp: "a-b"},
		{in: "äö/ôm/før mи", exp: "aeoe-om-foer-m"},
		{in: "a_b 123c", exp: "a-b-123c"},
		{in: "a\r\nb", exp: "ab"},
		{in: "Ñ-É-Þ", exp: "n-e-th"},
	}

	for i, test := range tests {
		out := SanitizeName(test.in)
		assert.Equal(t, test.exp, out, "failed for %s (test %d)", test.in, i)
	}
}
