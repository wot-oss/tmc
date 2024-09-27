// Package cli contains implementations of CLI commands. The command code is supposed contain only logic specific to
// the CLI and delegate complex/reusable stuff to code in /internal/commands.
// Commands in cli package should print results in human-readable format to stdout.
package cli

import (
	"fmt"
	"os"

	"github.com/wot-oss/tmc/internal/repos"
)

const DefaultListSeparator = ","

var TmcVersion = "n/a"

// Stderrf prints a message to os.Stderr, followed by newline
func Stderrf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format, args...)
	_, _ = fmt.Fprintln(os.Stderr)
}

func printErrs(hdr string, errs []*repos.RepoAccessError) {
	if len(errs) == 0 {
		return
	}
	Stderrf("%s", hdr)
	for _, e := range errs {
		Stderrf("%v", e)
	}
}

const (
	opResultOK = opResultType(iota)
	opResultWarn
	opResultErr
)

type opResultType int

func (t opResultType) String() string {
	switch t {
	case opResultOK:
		return "OK"
	case opResultWarn:
		return "warning"
	case opResultErr:
		return "error"
	default:
		return "unknown"
	}
}

type operationResult struct {
	typ        opResultType
	resourceId string
	text       string
}

func (r operationResult) String() string {
	return fmt.Sprintf("%v\t %s %s", r.typ, r.resourceId, r.text)
}
