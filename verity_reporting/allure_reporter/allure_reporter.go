package allure_reporter

import internalallure "github.com/verity-bdd/verity-bdd/internal/reporting/allure_reporter"

type AllureReporter = internalallure.AllureReporter

func NewAllureReporterWithDir(dir string) *AllureReporter {
	return internalallure.NewAllureReporterWithDir(dir)
}
