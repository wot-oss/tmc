package commands

import (
	"crypto/sha1"
	"fmt"

	"github.com/buger/jsonparser"
	"github.com/wot-oss/tmc/internal/utils"
)

// CalculateFileDigest calculates the hash string for TM version. Returns the 12-char hash string, the file contents
// that were hashed, and an error. The contents that were hashed may differ from the input.
// The changes to the contents are made to make the hashing reliable and idempotent: normalizing line endings,
// and setting 'id' to empty string
// If the file is not a valid json, the function is not guaranteed to return with an error
func CalculateFileDigest(raw []byte) (string, []byte, error) {
	raw = utils.NormalizeLineEndings(raw)
	fileForHashing, err := jsonparser.Set(raw, []byte("\"\""), "id")
	if err != nil {
		return "", raw, err
	}
	hasher := sha1.New()
	hasher.Write(fileForHashing)
	hash := hasher.Sum(nil)
	hashStr := fmt.Sprintf("%x", hash[:6])
	return hashStr, fileForHashing, nil
}
