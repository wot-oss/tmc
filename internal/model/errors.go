package model

import (
	"fmt"
	"strings"
)

var (
	ErrAuthorNotFound           *ErrNotFound
	ErrManufacturerNotFound     *ErrNotFound
	ErrTMNotFound               *ErrNotFound
	ErrTMNameNotFound           *ErrNotFound
	ErrAttachmentNotFound       *ErrNotFound
	ErrVariantNestingNotAllowed *ErrVariant
)

type ErrNotFound struct {
	Subject string
}

type ErrVariant struct {
	Subject string
}

func (e *ErrNotFound) Error() string {
	return strings.TrimSpace(fmt.Sprintf("%s not found", e.Subject))
}

func (e *ErrVariant) Error() string {
	return strings.TrimSpace(fmt.Sprintf("adding a nested variant to a tmid: %s as it is already a variant of %s", e.Subject, e.Subject))
}

func (e *ErrNotFound) Code() string {
	return e.Subject
}

func NewErrNotFound(subject string) *ErrNotFound {
	return &ErrNotFound{Subject: subject}
}

func NewErrVariant(subject string) *ErrVariant {
	return &ErrVariant{Subject: subject}
}

func init() {
	ErrAuthorNotFound = NewErrNotFound("author")
	ErrManufacturerNotFound = NewErrNotFound("manufacturer")
	ErrTMNotFound = NewErrNotFound("TM")
	ErrTMNameNotFound = NewErrNotFound("TM name")
	ErrAttachmentNotFound = NewErrNotFound("attachment")
	ErrVariantNestingNotAllowed = NewErrVariant("variant nesting is not allowed")
}
