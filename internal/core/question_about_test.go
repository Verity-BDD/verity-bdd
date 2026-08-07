package core_test

import (
	"context"
	"errors"
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

func TestQuestionAboutDescriptionIsNotPrefixed(t *stdtesting.T) {
	t.Parallel()
	question := core.QuestionAbout[string]("client record of company", func(ctx context.Context, actor core.Actor) (string, error) {
		return "ok", nil
	})

	if description := question.Description(); description != "client record of company" {
		t.Fatalf("expected original description, got %q", description)
	}
}

func TestQuestionAboutPanicsWhenAskFunctionIsNil(t *stdtesting.T) {
	t.Parallel()

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected QuestionAbout to panic for a nil ask function")
		}
		if recovered != "QuestionAbout: function parameter cannot be nil" {
			t.Fatalf("unexpected panic: %v", recovered)
		}
	}()

	var ask func(context.Context, core.Actor) (int, error)
	core.QuestionAbout("nil ask function", ask)
}

func TestQuestionAboutPropagatesAskFunctionError(t *stdtesting.T) {
	t.Parallel()
	ctx := context.Background()
	test := veritytesting.NewVerityTestWithContext(ctx, t)
	expectedErr := errors.New("question failed")

	question := core.QuestionAbout[int]("failing question", func(ctx context.Context, actor core.Actor) (int, error) {
		return 0, expectedErr
	})

	value, err := question.AnsweredBy(ctx, test.ActorCalled("Questioner"))
	if err != expectedErr {
		t.Fatalf("expected exact error %v, got %v", expectedErr, err)
	}
	if value != 0 {
		t.Fatalf("expected zero value, got %d", value)
	}
}
