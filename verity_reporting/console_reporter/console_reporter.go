package console_reporter

import internalconsole "github.com/verity-bdd/verity-bdd/internal/reporting/console_reporter"

type ConsoleReporter = internalconsole.ConsoleReporter

func NewConsoleReporter() *ConsoleReporter {
	return internalconsole.NewConsoleReporter()
}
