// Package cli contains implementations of CLI commands. The command code is supposed contain only logic specific to
// the CLI and delegate complex/reusable stuff to code in /internal/commands.
// Commands in cli package should print results in human-readable format to stdout.
package cli

import (
	"fmt"
	"os"
)

// Stderrf prints a message to os.Stderr, followed by newline
func Stderrf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format, args...)
	_, _ = fmt.Fprintln(os.Stderr)
}
