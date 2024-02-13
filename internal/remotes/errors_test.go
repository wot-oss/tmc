package remotes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseErrTMIDConflict(t *testing.T) {

	t.Run("same content type", func(t *testing.T) {
		e := &ErrTMIDConflict{
			Type:       IdConflictSameContent,
			ExistingId: "samecontentid",
		}
		cErr, err := ParseErrTMIDConflict(e.Code())
		assert.NoError(t, err)
		assert.Equal(t, e, cErr)
	})
	t.Run("same timestamp type", func(t *testing.T) {
		e := &ErrTMIDConflict{
			Type:       IdConflictSameTimestamp,
			ExistingId: "sametimestampid",
		}
		cErr, err := ParseErrTMIDConflict(e.Code())
		assert.NoError(t, err)
		assert.Equal(t, e, cErr)
	})
	t.Run("invalid error code", func(t *testing.T) {
		_, err := ParseErrTMIDConflict("0:s")
		assert.ErrorIs(t, err, ErrInvalidErrorCode)
		_, err = ParseErrTMIDConflict("")
		assert.ErrorIs(t, err, ErrInvalidErrorCode)
		_, err = ParseErrTMIDConflict("id")
		assert.ErrorIs(t, err, ErrInvalidErrorCode)
	})

}
