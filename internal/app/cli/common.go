// Package cli contains implementations of CLI commands. The command code is supposed contain only logic specific to
// the CLI and delegate complex/reusable stuff to code in /internal/commands.
// Commands in cli package should print results in human-readable format to stdout.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"

	"github.com/wot-oss/tmc/internal/repos"
)

const DefaultListSeparator = ","

const (
	OutputFormatJSON  = "json"
	OutputFormatPlain = "plain"
)

var ErrInvalidOutputFormat = errors.New("invalid output format")

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

func printJSON(js any) {
	v := reflect.ValueOf(js)
	if (v.Kind() == reflect.Array || v.Kind() == reflect.Slice) && v.Len() == 0 {
		fmt.Println("[]")
		return
	}
	if v.Kind() == reflect.Map && v.Len() == 0 {
		fmt.Println("{}")
		return
	}
	b, _ := json.MarshalIndent(js, "", "  ")
	fmt.Println(string(b))
}

func IsValidOutputFormat(format string) bool {
	switch format {
	case OutputFormatJSON, OutputFormatPlain:
		return true
	}
	return false
}

const (
	opResultOK = OpResultType(iota)
	opResultWarn
	opResultErr
)

type OpResultType int

func (t OpResultType) String() string {
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

func (t OpResultType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

type OperationResult struct {
	Type       OpResultType `json:"type"`
	ResourceId string       `json:"resourceId"`
	Text       string       `json:"text,omitempty"`
}

func (r OperationResult) String() string {
	return fmt.Sprintf("%v\t %s %s", r.Type, r.ResourceId, r.Text)
}
