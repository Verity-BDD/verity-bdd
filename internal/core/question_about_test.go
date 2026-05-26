package core_test

import (
	"context"
	stdtesting "testing"

	"github.com/verity-bdd/verity-bdd/internal/core"
	veritytesting "github.com/verity-bdd/verity-bdd/internal/testing"
)

func TestQuestionAboutCreatesQuestion(t *stdtesting.T) {
	t.Parallel()
	ctx := context.Background()
	test := veritytesting.NewVerityTestWithContext(ctx, t)

	question := core.QuestionAbout[int]("number", func(ctx context.Context, actor core.Actor) (int, error) {
		return 7, nil
	})

	actor := test.ActorCalled("Questioner")
	value, err := question.AnsweredBy(ctx, actor)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if value != 7 {
		t.Fatalf("expected value 7, got %d", value)
	}

	if description := question.Description(); description != "number" {
		t.Fatalf("expected description 'number', got %q", description)
	}
}

func TestNewQuestionDescriptionIsNotPrefixed(t *stdtesting.T) {
	t.Parallel()
	question := core.NewQuestion[string]("client record of company", func(ctx context.Context, actor core.Actor) (string, error) {
		return "ok", nil
	})

	if description := question.Description(); description != "client record of company" {
		t.Fatalf("expected original description, got %q", description)
	}
}
