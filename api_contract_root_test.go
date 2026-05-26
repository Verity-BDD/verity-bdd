package verity_test

import (
	"context"
	"testing"

	verity "github.com/verity-bdd/verity-bdd"
)

func TestRootAPIContractCompiles(t *testing.T) {
	t.Parallel()
	var _ verity.Activity = verity.Do("noop", func(ctx context.Context, actor verity.Actor) error {
		return nil
	})

	_ = verity.TaskWhere("empty")

	q := verity.QuestionAbout("answer", func(ctx context.Context, actor verity.Actor) (int, error) {
		return 42, nil
	})

	_, _ = q.AnsweredBy(context.Background(), nil)
}
