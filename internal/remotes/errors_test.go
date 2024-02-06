package remotes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrTMIDConflict_FromString(t *testing.T) {

	t.Run("same content type", func(t *testing.T) {
		e := &ErrTMIDConflict{
			Type:       IdConflictSameContent,
			ExistingId: "samecontentid",
		}
		ne := &ErrTMIDConflict{}
		ne.FromString(e.Error())
		assert.Equal(t, e, ne)
	})
	t.Run("same timestamp type", func(t *testing.T) {
		e := &ErrTMIDConflict{
			Type:       IdConflictSameTimestamp,
			ExistingId: "sametimestampid",
		}
		ne := &ErrTMIDConflict{}
		ne.FromString(e.Error())
		assert.Equal(t, e, ne)
	})
	t.Run("broken string", func(t *testing.T) {
		e := &ErrTMIDConflict{
			Type:       IdConflictUnknown,
			ExistingId: "",
		}
		ne := &ErrTMIDConflict{}
		ne.FromString("Thing Model id conflict: Type: who knows, existing id")
		assert.Equal(t, e, ne)
	})

}
