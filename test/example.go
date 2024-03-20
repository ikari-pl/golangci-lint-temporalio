package test

import (
	"context"

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

	// and an activity that's a struct with a method
	tWorker.RegisterActivity(&SophisticatedHelloWorldActivity{})

	// start a workflow
	executeWorkflow, err := temporalClient.ExecuteWorkflow(context.Background(), client.StartWorkflowOptions{}, "World")
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
	err := workflow.ExecuteActivity(ctx, HelloWorldActivity, name).Get(ctx, &result)
	if err != nil {
		return "", err
	}

	var act *SophisticatedHelloWorldActivity
	err = workflow.ExecuteActivity(ctx, act.Greet2, Greet2Param{name: name}).Get(ctx, &result)
	return result, err
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
