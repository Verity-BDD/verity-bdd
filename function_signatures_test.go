package verity_test

import (
	"context"

	verity "github.com/verity-bdd/verity-bdd"
	reporting "github.com/verity-bdd/verity-bdd/verity_reporting"
)

var (
	_ func(string, func(context.Context, verity.Actor) error) verity.Interaction      = verity.Do
	_ func(string, ...verity.Activity) verity.Task                                    = verity.TaskWhere
	_ func() verity.FailureMode                                                       = verity.Critical
	_ func() verity.FailureMode                                                       = verity.NonCritical
	_ func() verity.FailureMode                                                       = verity.Optional
	_ func(verity.Ability) string                                                     = verity.AbilityName
	_ func(verity.TestContext, verity.Scene) verity.VerityTest                        = verity.NewVerityTest
	_ func(context.Context, verity.TestContext) verity.VerityTest                     = verity.NewVerityTestWithContext
	_ func(context.Context, verity.TestContext, reporting.Reporter) verity.VerityTest = verity.NewVerityTestWithReporter
)
