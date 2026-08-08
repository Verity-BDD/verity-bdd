// Package console_reporter provides the built-in human-readable reporter.
package console_reporter

import internalconsole "github.com/verity-bdd/verity-bdd/internal/reporting/console_reporter"

// ConsoleReporter writes test start, completed-step, and test finish lines.
// OnStepStart tracks nesting but does not write a separate progress line.
type ConsoleReporter = internalconsole.ConsoleReporter

// NewConsoleReporter returns a reporter that writes to os.Stdout by default.
// Use SetOutput to select another io.Writer.
func NewConsoleReporter() *ConsoleReporter {
	return internalconsole.NewConsoleReporter()
}
