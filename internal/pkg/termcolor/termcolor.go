// Package termcolor decides whether ANSI colors may be written to an output.
package termcolor

import (
	"os"

	"golang.org/x/term"
)

// Enabled reports whether colors should be written to f: only when f is a terminal
// and NO_COLOR is not set (https://no-color.org). Logs redirected to a file or read
// by a log collector therefore never contain escape codes.
func Enabled(f *os.File) bool {
	if _, noColor := os.LookupEnv("NO_COLOR"); noColor {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}
