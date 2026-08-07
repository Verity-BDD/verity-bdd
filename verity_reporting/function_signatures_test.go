package verity_reporting_test

import (
	vr "github.com/verity-bdd/verity-bdd/verity_reporting"
	"github.com/verity-bdd/verity-bdd/verity_reporting/allure_reporter"
	"github.com/verity-bdd/verity-bdd/verity_reporting/console_reporter"
)

var (
	_ func(vr.Reporter) *vr.TestRunnerAdapter               = vr.NewTestRunnerAdapter
	_ func(vr.Reporter, string) *vr.ActivityTracker         = vr.NewActivityTracker
	_ func(vr.Reporter, string, string) *vr.ActivityTracker = vr.NewActivityTrackerWithActor
	_ func() *console_reporter.ConsoleReporter              = console_reporter.NewConsoleReporter
	_ func(string) *allure_reporter.AllureReporter          = allure_reporter.NewAllureReporterWithDir
)
