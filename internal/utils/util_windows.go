//go:build windows

package utils

import "bytes"

// ConvertToNativeLineEndings converts all instances of '\n' to native line endings for the platform.
// Assumes that line endings are normalized, i.e. there are no '\r' or "\r\n" line endings in the data
// See NormalizeLineEndings
func ConvertToNativeLineEndings(b []byte) []byte {
	if len(b) == 0 {
		return b
	}
	return bytes.ReplaceAll(b, []byte{'\n'}, []byte{'\r', '\n'})
}
