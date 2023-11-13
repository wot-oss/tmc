package commands

import "testing"

func TestElideString(t *testing.T) {
	testCases := []struct {
		colWidth int
		input    string
		expected string
	}{
		{
			colWidth: 10,
			input:    "",
			expected: "",
		},
		{
			colWidth: 10,
			input:    "testing",
			expected: "testing",
		},
		{
			colWidth: 4,
			input:    "testing",
			expected: "t...",
		},
	}
	for _, test := range testCases {
		out := elideString(test.input, test.colWidth)
		if test.expected != elideString(test.input, test.colWidth) {
			t.Errorf("failed eliding '%s' to %d characters:", test.input, test.colWidth)
			t.Errorf("expected '%s' got '%s'", test.expected, out)
		}
	}
}
