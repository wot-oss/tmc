package repos

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

var (
	ErrAmbiguous               = errors.New("multiple repos configured, but target repo not specified")
	ErrRepoNotFound            = errors.New("repo not found")
	ErrInvalidRepoName         = errors.New("invalid repo name")
	ErrRepoExists              = errors.New("named repo already exists")
	ErrAttachmentExists        = errors.New("attachment already exists")
	ErrInvalidErrorCode        = errors.New("invalid error code")
	ErrInvalidCompletionParams = errors.New("invalid completion parameters")
	ErrNotSupported            = errors.New("method not supported")
	ErrResourceAccess          = errors.New("cannot access resource")
	ErrResourceInvalid         = errors.New("invalid resource name")
	ErrResourceNotExists       = errors.New("resource does not exist")
	ErrIndexMismatch           = errors.New("index does not reflect repository content, maybe needs rebuild")
	ErrNoIndex                 = errors.New("no table of contents found. Run `index` for this repo")
)

type CodedError interface {
	Code() string
}

type ErrTMIDConflict struct {
	Type       IdConflictType
	ExistingId string
}

type IdConflictType int

const (
	IdConflictSameContent = iota + 1
	IdConflictSameTimestamp
)

var (
	idConflictStrings = map[IdConflictType]string{
		IdConflictSameContent:   "same content",
		IdConflictSameTimestamp: "same timestamp",
	}
	idConflictCodeRegex = regexp.MustCompile("^([12]):(.+?)$")
)

func (t IdConflictType) String() string {
	if s, ok := idConflictStrings[t]; ok {
		return s
	}
	return fmt.Sprintf("unknown conflict type: %d", t)
}

func stringToIdConflictType(s string) IdConflictType {
	i, _ := strconv.Atoi(s)
	return IdConflictType(i)
}

func (e *ErrTMIDConflict) Error() string {
	return fmt.Sprintf("Thing Model id conflict. Type: %v, existing id: %s", e.Type, e.ExistingId)
}

// Code returns a machine-readable string error code, which can be parsed by ParseErrTMIDConflict
func (e *ErrTMIDConflict) Code() string {
	return fmt.Sprintf("%d:%s", int(e.Type), e.ExistingId)
}

func ParseErrTMIDConflict(errCode string) (*ErrTMIDConflict, error) {
	matches := idConflictCodeRegex.FindStringSubmatch(errCode)
	if len(matches) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrInvalidErrorCode, errCode)
	}
	return &ErrTMIDConflict{
		Type:       stringToIdConflictType(matches[1]), // invalid conflict type would not match the regex
		ExistingId: matches[2],
	}, nil
}
