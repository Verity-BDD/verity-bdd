// Package allure_reporter provides an Allure 2 result-file reporter.
package allure_reporter

import internalallure "github.com/verity-bdd/verity-bdd/internal/reporting/allure_reporter"

// AllureReporter writes test results, nested steps, and callback attachments to
// an Allure results directory. The normal actor pipeline currently provides no
// step attachments; built-in actor notes can appear as a test-level attachment.
type AllureReporter = internalallure.AllureReporter

// NewAllureReporterWithDir returns a reporter that writes Allure result and
// attachment files beneath dir. SetOutput on this reporter is a no-op.
func NewAllureReporterWithDir(dir string) *AllureReporter {
	return internalallure.NewAllureReporterWithDir(dir)
}
