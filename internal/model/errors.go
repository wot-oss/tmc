package model

import (
	"fmt"
	"strings"
)

var (
	ErrTMNotFound         *ErrNotFound
	ErrTMNameNotFound     *ErrNotFound
	ErrAttachmentNotFound *ErrNotFound
)

type ErrNotFound struct {
	Subject string
}

func (e *ErrNotFound) Error() string {
	return strings.TrimSpace(fmt.Sprintf("%s not found", e.Subject))
}

func (e *ErrNotFound) Code() string {
	return e.Subject
}

func NewErrNotFound(subject string) *ErrNotFound {
	return &ErrNotFound{Subject: subject}
}

func init() {
	ErrTMNotFound = NewErrNotFound("TM")
	ErrTMNameNotFound = NewErrNotFound("TM name")
	ErrAttachmentNotFound = NewErrNotFound("attachment")
}
