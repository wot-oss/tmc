package remotes

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	ErrAmbiguous         = errors.New("multiple remotes configured, but remote target not specified")
	ErrRemoteNotFound    = errors.New("remote not found")
	ErrInvalidRemoteName = errors.New("invalid remote remoteName")
	ErrRemoteExists      = errors.New("named remote already exists")
	ErrTmNotFound        = errors.New("TM not found")
	ErrInvalidSpec       = errors.New("illegal remote spec: both dir and remoteName given")
)

type ErrTMIDConflict struct {
	Type       IdConflictType
	ExistingId string
}

type IdConflictType int

const (
	IdConflictUnknown IdConflictType = iota
	IdConflictSameContent
	IdConflictSameTimestamp
)
const errTMIDConflictPrefix = "Thing Model id conflict"

var (
	idConflictStrings = map[IdConflictType]string{
		IdConflictSameContent:   "same content",
		IdConflictSameTimestamp: "same timestamp",
	}
	idConflictTypeRegex       = regexp.MustCompile("Type: (.+?),")
	idConflictExistingIdRegex = regexp.MustCompile(": ([^,]+?)$")
)

func (t IdConflictType) String() string {
	if s, ok := idConflictStrings[t]; ok {
		return s
	}
	return "unknown conflict type"
}

func ParseIdConflictType(s string) IdConflictType {
	for t, ts := range idConflictStrings {
		if s == ts {
			return t
		}
	}
	return 0
}

func (e *ErrTMIDConflict) Error() string {
	return fmt.Sprintf(errTMIDConflictPrefix+". Type: %v, existing id: %s", e.Type, e.ExistingId)
}

func (e *ErrTMIDConflict) FromString(s string) {
	if !strings.HasPrefix(s, errTMIDConflictPrefix) {
		return
	}

	t := IdConflictUnknown
	tMatches := idConflictTypeRegex.FindStringSubmatch(s)
	if tMatches != nil {
		t = ParseIdConflictType(tMatches[1])
	}
	e.Type = t

	iMatches := idConflictExistingIdRegex.FindStringSubmatch(s)
	if iMatches != nil {
		e.ExistingId = iMatches[1]
	}
}
