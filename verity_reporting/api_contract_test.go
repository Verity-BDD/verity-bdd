package verity_reporting_test

import (
	"testing"

	vr "github.com/verity-bdd/verity-bdd/verity_reporting"
	"github.com/verity-bdd/verity-bdd/verity_reporting/console_reporter"
)

func TestReportingAPIContractCompiles(t *testing.T) {
	t.Parallel()
	var reporter vr.Reporter = console_reporter.NewConsoleReporter()
	_ = reporter
}
