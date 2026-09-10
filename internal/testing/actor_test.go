package testing

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/verity-bdd/verity-bdd/internal/abilities"
	"github.com/verity-bdd/verity-bdd/internal/core"
	coreMocks "github.com/verity-bdd/verity-bdd/internal/core/testing/mocks"
	"github.com/verity-bdd/verity-bdd/internal/reporting"
	reportingMocks "github.com/verity-bdd/verity-bdd/internal/reporting/mocks"
	testingMocks "github.com/verity-bdd/verity-bdd/internal/testing/mocks"
)

type dummyAbility struct{ id string }

type ifaceAbility interface {
	abilities.Ability
	Foo() string
}

type ifaceImpl struct{ id string }

func (i *ifaceImpl) Foo() string { return i.id }

type callbackFact struct {
	description string
	setup       func(context.Context, core.Actor) error
	teardown    func(context.Context, core.Actor) error
}

func (f *callbackFact) Description() string { return f.description }

func (f *callbackFact) Setup(ctx context.Context, actor core.Actor) error {
	return f.setup(ctx, actor)
}

func (f *callbackFact) Teardown(ctx context.Context, actor core.Actor) error {
	return f.teardown(ctx, actor)
}

func TestTestActorAttemptsToWithReporting(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mocks
	mockReporter := reportingMocks.NewMockReporter(ctrl)
	mockTestContext := testingMocks.NewMockTestContext(ctrl)

	// Expect OnStepStart and OnStepFinish for activity
	mockReporter.EXPECT().OnStepStart("Send GET request to /posts").Times(1)
	mockReporter.EXPECT().OnStepFinish(gomock.Any()).Times(1)

	// Expect no error from test context
	mockTestContext.EXPECT().Failed().Return(false).AnyTimes()

	// Create test actor with mock reporter
	adapter := reporting.NewTestRunnerAdapter(mockReporter)
	testCtx := context.Background()
	test := &verityTest{
		testCtx: mockTestContext,
		ctx:     testCtx,
		actors:  make(map[string]core.Actor),
		adapter: adapter,
	}

	actor := test.ActorCalled("TestActor")

	// Create mock activity
	mockActivity := coreMocks.NewMockActivity(ctrl)
	mockActivity.EXPECT().PerformAs(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	mockActivity.EXPECT().Description().Return("Send GET request to /posts").Times(1)
	mockActivity.EXPECT().FailureMode().Return(core.FailFast).AnyTimes()

	// Execute activity
	actor.AttemptsTo(mockActivity)
}

func TestTestActorAttemptsToWithNestedTaskReporting(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockReporter := reportingMocks.NewMockReporter(ctrl)
	mockTestContext := testingMocks.NewMockTestContext(ctrl)

	gomock.InOrder(
		mockReporter.EXPECT().OnStepStart("Sam creates an order"),
		mockReporter.EXPECT().OnStepStart("Sam opens order page"),
		mockReporter.EXPECT().OnStepFinish(gomock.Any()).Do(func(result reporting.TestResult) {
			if result.Name() != "Sam opens order page" {
				t.Fatalf("unexpected child step name: %s", result.Name())
			}
		}),
		mockReporter.EXPECT().OnStepStart("Sam saves order"),
		mockReporter.EXPECT().OnStepFinish(gomock.Any()).Do(func(result reporting.TestResult) {
			if result.Name() != "Sam saves order" {
				t.Fatalf("unexpected child step name: %s", result.Name())
			}
		}),
		mockReporter.EXPECT().OnStepFinish(gomock.Any()).Do(func(result reporting.TestResult) {
			if result.Name() != "Sam creates an order" {
				t.Fatalf("unexpected task step name: %s", result.Name())
			}
		}),
	)

	mockTestContext.EXPECT().Failed().Return(false).AnyTimes()

	adapter := reporting.NewTestRunnerAdapter(mockReporter)
	test := &verityTest{
		testCtx: mockTestContext,
		ctx:     context.Background(),
		actors:  make(map[string]core.Actor),
		adapter: adapter,
	}

	actor := test.ActorCalled("Sam")
	actor.AttemptsTo(
		core.TaskWhere("#actor creates an order",
			core.Do("#actor opens order page", func(ctx context.Context, actor core.Actor) error {
				return nil
			}),
			core.Do("#actor saves order", func(ctx context.Context, actor core.Actor) error {
				return nil
			}),
		),
	)
}

func TestTestActorPerformActivityReportsNestedTaskHierarchy(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockReporter := reportingMocks.NewMockReporter(ctrl)
	mockTestContext := testingMocks.NewMockTestContext(ctrl)

	gomock.InOrder(
		mockReporter.EXPECT().OnStepStart("Sam creates an order"),
		mockReporter.EXPECT().OnStepStart("Sam submits order details"),
		mockReporter.EXPECT().OnStepStart("Sam opens order page"),
		mockReporter.EXPECT().OnStepFinish(gomock.Any()).Do(func(result reporting.TestResult) {
			if result.Name() != "Sam opens order page" {
				t.Fatalf("unexpected leaf step name: %s", result.Name())
			}
		}),
		mockReporter.EXPECT().OnStepFinish(gomock.Any()).Do(func(result reporting.TestResult) {
			if result.Name() != "Sam submits order details" {
				t.Fatalf("unexpected nested task name: %s", result.Name())
			}
		}),
		mockReporter.EXPECT().OnStepFinish(gomock.Any()).Do(func(result reporting.TestResult) {
			if result.Name() != "Sam creates an order" {
				t.Fatalf("unexpected root task name: %s", result.Name())
			}
		}),
	)

	actor := &testActor{
		name:        "Sam",
		testContext: mockTestContext,
		reporter:    reporting.NewTestRunnerAdapter(mockReporter),
		ctx:         context.Background(),
	}

	err := actor.PerformActivity(context.Background(),
		core.TaskWhere("#actor creates an order",
			core.TaskWhere("#actor submits order details",
				core.Do("#actor opens order page", func(ctx context.Context, actor core.Actor) error {
					return nil
				}),
			),
		),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestAbilityToReturnsFriendlyError(t *testing.T) {
	t.Parallel()
	actor := &testActor{
		name:      "TestActor",
		ctx:       context.Background(),
		abilities: []abilities.Ability{},
	}

	ability, err := actor.AbilityTo(&dummyAbility{})
	if err == nil {
		t.Fatalf("expected error, got nil and ability %v", ability)
	}

	expected := "actor 'TestActor' can't testing.dummyAbility. Did you give them the ability?"
	if err.Error() != expected {
		t.Fatalf("expected error %q, got %q", expected, err.Error())
	}
}

func TestAbilityOfReturnsConcreteAbility(t *testing.T) {
	t.Parallel()
	first := &dummyAbility{id: "first"}
	actor := &testActor{
		name:      "TestActor",
		ctx:       context.Background(),
		abilities: []abilities.Ability{first},
	}

	ability, err := core.AbilityOf[*dummyAbility](actor)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if ability != first {
		t.Fatalf("expected first ability, got %+v", ability)
	}
}

func TestAbilityOfReturnsFriendlyErrorWhenMissing(t *testing.T) {
	t.Parallel()
	actor := &testActor{
		name:      "TestActor",
		ctx:       context.Background(),
		abilities: []abilities.Ability{},
	}

	ability, err := core.AbilityOf[*dummyAbility](actor)
	if err == nil {
		t.Fatalf("expected error, got ability %+v", ability)
	}

	expected := "actor 'TestActor' can't testing.dummyAbility. Did you give them the ability?"
	if err.Error() != expected {
		t.Fatalf("expected error %q, got %q", expected, err.Error())
	}
}

func TestAbilityOfReturnsFirstMatchingAbility(t *testing.T) {
	t.Parallel()
	first := &dummyAbility{id: "first"}
	second := &dummyAbility{id: "second"}
	actor := &testActor{
		name:      "TestActor",
		ctx:       context.Background(),
		abilities: []abilities.Ability{first, second},
	}

	ability, err := core.AbilityOf[*dummyAbility](actor)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if ability != first {
		t.Fatalf("expected first ability, got %+v", ability)
	}
}

func TestAbilityOfHandlesNilActor(t *testing.T) {
	t.Parallel()
	ability, err := core.AbilityOf[*dummyAbility](nil)
	if err == nil {
		t.Fatalf("expected error, got ability %+v", ability)
	}

	expected := "actor is nil; cannot get testing.dummyAbility ability"
	if err.Error() != expected {
		t.Fatalf("expected error %q, got %q", expected, err.Error())
	}
}

func TestAbilityOfSupportsInterfaceAbility(t *testing.T) {
	t.Parallel()
	impl := &ifaceImpl{id: "ok"}
	actor := &testActor{
		name:      "TestActor",
		ctx:       context.Background(),
		abilities: []abilities.Ability{impl},
	}

	ability, err := core.AbilityOf[ifaceAbility](actor)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if ability.Foo() != "ok" {
		t.Fatalf("expected Foo to return ok, got %s", ability.Foo())
	}
}

func TestActorHasStopsAfterSetupError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	testContext := testingMocks.NewMockTestContext(ctrl)
	actor := &testActor{
		name:        "Sam",
		testContext: testContext,
		ctx:         context.Background(),
	}
	setupErr := errors.New("account service unavailable")
	var setupOrder []string
	var teardownOrder []string

	testContext.EXPECT().Errorf("Fact %q setup failed for actor %q: %v", "savings account", "Sam", setupErr)
	testContext.EXPECT().FailNow()

	actor.Has(
		&callbackFact{
			description: "registered account",
			setup: func(context.Context, core.Actor) error {
				setupOrder = append(setupOrder, "A")
				return nil
			},
			teardown: func(context.Context, core.Actor) error {
				teardownOrder = append(teardownOrder, "A")
				return nil
			},
		},
		&callbackFact{
			description: "savings account",
			setup: func(context.Context, core.Actor) error {
				setupOrder = append(setupOrder, "B")
				return setupErr
			},
			teardown: func(context.Context, core.Actor) error {
				teardownOrder = append(teardownOrder, "B")
				return nil
			},
		},
		&callbackFact{
			description: "credit card",
			setup: func(context.Context, core.Actor) error {
				setupOrder = append(setupOrder, "C")
				return nil
			},
			teardown: func(context.Context, core.Actor) error {
				teardownOrder = append(teardownOrder, "C")
				return nil
			},
		},
	)
	actor.teardownFacts()

	if got, want := setupOrder, []string{"A", "B"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("setup order = %v, want %v", got, want)
	}
	if got, want := teardownOrder, []string{"A"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("teardown order = %v, want %v", got, want)
	}
}

func TestAbilityOfReturnsFirstMatchingInterfaceAbility(t *testing.T) {
	t.Parallel()
	first := &ifaceImpl{id: "first"}
	second := &ifaceImpl{id: "second"}
	actor := &testActor{
		name:      "TestActor",
		ctx:       context.Background(),
		abilities: []abilities.Ability{first, second},
	}

	ability, err := core.AbilityOf[ifaceAbility](actor)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if ability != first {
		t.Fatalf("expected first interface ability, got %+v", ability)
	}
}
