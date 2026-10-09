package core

import (
	"context"
	"fmt"
)

// ConditionalCheck evaluates a boolean question and performs its selected activities.
type ConditionalCheck interface {
	Activity
	AndIfSo(activities ...Activity) ConditionalCheck
	Otherwise(activities ...Activity) ConditionalCheck
}

type check struct {
	question  Question[bool]
	ifSo      []Activity
	otherwise []Activity
}

func (c *check) AndIfSo(activities ...Activity) ConditionalCheck {
	c.ifSo = activities
	return c
}

func (c *check) Otherwise(activities ...Activity) ConditionalCheck {
	c.otherwise = activities
	return c
}

func (c *check) Description() string {
	return fmt.Sprintf("checks whether %s", c.question.Description())
}

func (*check) FailureMode() FailureMode {
	return FailFast
}

func (c *check) PerformAs(ctx context.Context, actor Actor) error {
	matches, err := c.question.AnsweredBy(ctx, actor)
	if err != nil {
		return fmt.Errorf("answer condition question %q: %w", c.question.Description(), err)
	}

	activities := c.otherwise
	if matches {
		activities = c.ifSo
	}
	for _, activity := range activities {
		if err := performActivity(ctx, actor, activity); err != nil {
			return fmt.Errorf("condition %q failed during activity %q: %w", c.question.Description(), activity.Description(), err)
		}
	}
	return nil
}

// CheckWhether creates a conditional activity that evaluates question for its actor.
func CheckWhether(question Question[bool]) ConditionalCheck {
	return &check{question: question}
}
