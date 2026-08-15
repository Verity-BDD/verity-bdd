// Package log provides a reporter-visible Screenplay Log interaction.
package log

import (
	"context"
	"fmt"

	verity "github.com/verity-bdd/verity-bdd"
	"github.com/verity-bdd/verity-bdd/verity_reporting"
)

type logInteraction struct {
	values []any
}

type answerArgument struct {
	description string
	answer      func(context.Context, verity.Actor) (any, error)
}

type logSink interface {
	Log(verity_reporting.LogEntry) error
}

// The creates a fail-fast interaction that sends its resolved values to the
// performing actor's reporting sink.
func The(values ...any) verity.Interaction {
	return logInteraction{values: append([]any(nil), values...)}
}

// AnswerTo adapts a typed Question for use with The.
func AnswerTo[T any](question verity.Question[T]) any {
	return answerArgument{
		description: question.Description(),
		answer: func(ctx context.Context, actor verity.Actor) (any, error) {
			return question.AnsweredBy(ctx, actor)
		},
	}
}

func (l logInteraction) Description() string {
	return "#actor logs"
}

func (l logInteraction) FailureMode() verity.FailureMode {
	return verity.FailFast
}

func (l logInteraction) PerformAs(ctx context.Context, actor verity.Actor) error {
	sink, ok := actor.(logSink)
	if !ok {
		return fmt.Errorf("actor %q does not implement LogSink", actor.Name())
	}

	values := make([]string, 0, len(l.values))
	for _, value := range l.values {
		if question, ok := value.(answerArgument); ok {
			answer, err := question.answer(ctx, actor)
			if err != nil {
				return fmt.Errorf("answer log question %q: %w", question.description, err)
			}
			values = append(values, fmt.Sprint(answer))
			continue
		}
		values = append(values, fmt.Sprint(value))
	}

	return sink.Log(verity_reporting.LogEntry{ActorName: actor.Name(), Values: values})
}
