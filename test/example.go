package test

import (
	"context"
	"errors"
	"strings"

	"go.temporal.io/sdk/client"
	worker "go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func Main() {
	// create a temporal client
	temporalClient, err := client.NewLazyClient(client.Options{})
	if err != nil {
		panic(err)
	}

	// create a worker
	tWorker := worker.New(temporalClient, "test", worker.Options{})

	// register a workflow
	tWorker.RegisterWorkflow(HelloWorldWorkflow)

	// register an activity that's a plain function
	tWorker.RegisterActivity(HelloWorldActivity)

	// wrong registration, the activity is registered as a workflow
	tWorker.RegisterWorkflow(HelloWorldActivity)

	// or the other way around
	tWorker.RegisterActivity(HelloWorldWorkflow)

	// and an activity that's a struct with a method
	tWorker.RegisterActivity(&SophisticatedHelloWorldActivity{})

	// start a workflow
	executeWorkflow, err := temporalClient.ExecuteWorkflow(
		context.Background(),
		client.StartWorkflowOptions{},
		HelloWorldWorkflow,
		"World")
	if err != nil {
		panic(err)
	}

	// wait for the workflow to complete
	err = executeWorkflow.Get(context.Background(), nil)
	if err != nil {
		panic(err)
	}
}

func HelloWorldWorkflow(ctx workflow.Context, name string) (string, error) {
	var result string
	var errList []error

	// call the activity (plain function, string input)
	errList = append(errList, workflow.ExecuteActivity(ctx, HelloWorldActivity, name).Get(ctx, &result))

	var act *SophisticatedHelloWorldActivity
	// call the activity (struct with method, string input)
	errList = append(errList, workflow.ExecuteActivity(ctx, act.Greet, name).Get(ctx, &result))

	// now mis-call the activity passing an int instead of a struct
	errList = append(errList, workflow.ExecuteActivity(ctx, act.Greet2, 42).Get(ctx, &result))

	// a nil pointer to a struct can be untyped and it's "fine"
	errList = append(errList, workflow.ExecuteActivity(ctx, act.Greet2, nil).Get(ctx, &result))

	// too many arguments
	errList = append(errList, workflow.ExecuteActivity(ctx, act.Greet, name, "extra").Get(ctx, &result))

	// too few arguments
	errList = append(errList, workflow.ExecuteActivity(ctx, act.Greet).Get(ctx, &result))

	// calling a variadic string function, but mixing one float into the variadic list
	errList = append(errList, workflow.ExecuteActivity(ctx, HelloVariadic,
		",", "a", "b", "c", "d", 1.1, "e", "f").Get(ctx, &result))

	// correct, resolved by activity name
	errList = append(errList, workflow.ExecuteActivity(ctx, "HelloWorldActivity", name).Get(ctx, &result))

	// incorrect, resolved by activity name, too many arguments
	errList = append(errList, workflow.ExecuteActivity(ctx, "HelloWorldActivity", name, "extra").Get(ctx, &result))

	return result, errors.Join(errList...)
}

func HelloWorldActivity(ctx context.Context, name string) (string, error) {
	return "Hello " + name, nil
}

type SophisticatedHelloWorldActivity struct{}

func (s *SophisticatedHelloWorldActivity) Greet(ctx context.Context, name string) (string, error) {
	return "Hello " + name, nil
}

type Greet2Param struct {
	name string // this will fail because it's not exported
}

func (s *SophisticatedHelloWorldActivity) Greet2(ctx context.Context, param Greet2Param) (string, error) {
	return "Hello " + param.name, nil
}

func HelloVariadic(ctx context.Context, sep string, names ...string) (string, error) {
	return "Hello " + strings.Join(names, sep), nil
}
