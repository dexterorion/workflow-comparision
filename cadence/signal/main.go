package main

import (
	"context"
	"flag"
	"time"

	"github.com/pborman/uuid"

	"github.com/uber-go/tally"
	"go.uber.org/cadence/.gen/go/cadence/workflowserviceclient"
	"go.uber.org/cadence/client"
	"go.uber.org/cadence/worker"
	"go.uber.org/cadence/workflow"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/zap"

	wf "avenuesec/workflow-poc/cadence/signal/workflow"
)

var HostPort = "127.0.0.1:7933"
var Domain = "simpledomain"
var ClientName = "simpleworker"
var CadenceService = "cadence-frontend"
var CadenceClientName = "cadence-client"

func main() {
	var mode string
	var text string
	var executionID string
	flag.StringVar(&mode, "m", "trigger", "Mode is worker, trigger or shadower.")
	flag.StringVar(&executionID, "id", "id", "Workflow exection ID.")
	flag.StringVar(&text, "text", "Anything", "Text to be shown. Exit finishes workflow")
	flag.Parse()

	switch mode {
	case "worker":
		startWorker(buildLogger(), buildCadenceClient())

		// The workers are supposed to be long running process that should not exit.
		// Use select{} to block indefinitely for samples, you can quit by CMD+C.
		select {}

	case "trigger":
		startWorkflow(buildCadenceClient())

	case "signal":
		sendSignal(buildCadenceClient(), executionID, text)
	}

}

func sendSignal(service workflowserviceclient.Interface, executionID, text string) {
	ctx := context.Background()

	logger := buildLogger()

	var workflowClient client.Client = client.NewClient(
		service, Domain, &client.Options{Identity: "local-mac-vinny", MetricsScope: tally.NoopScope, ContextPropagators: []workflow.ContextPropagator{}})

	err := workflowClient.SignalWorkflow(ctx, executionID, "", wf.SignalName, text)

	if err != nil {
		logger.Error("Failed to send signal", zap.Error(err))
		panic("Failed to send signal.")
	} else {
		logger.Info("Signal sent", zap.String("WorkflowID", executionID), zap.String("Text", text))
	}
}

func startWorkflow(service workflowserviceclient.Interface) {
	ctx := context.Background()

	workflowOptions := client.StartWorkflowOptions{
		ID:                              "signalworld_" + uuid.New(),
		TaskList:                        wf.ApplicationName,
		ExecutionStartToCloseTimeout:    time.Minute,
		DecisionTaskStartToCloseTimeout: time.Minute,
	}

	logger := buildLogger()

	var workflowClient client.Client = client.NewClient(
		service, Domain, &client.Options{Identity: "local-mac-vinny", MetricsScope: tally.NoopScope, ContextPropagators: []workflow.ContextPropagator{}})

	we, err := workflowClient.StartWorkflow(ctx, workflowOptions, wf.WorkflowName, "Cadence")
	if err != nil {
		logger.Error("Failed to create workflow", zap.Error(err))
		panic("Failed to create workflow.")
	} else {
		logger.Info("Started Workflow", zap.String("WorkflowID", we.ID), zap.String("RunID", we.RunID))
	}
}

func buildLogger() *zap.Logger {
	config := zap.NewDevelopmentConfig()

	var err error
	logger, err := config.Build()
	if err != nil {
		panic("Failed to setup logger")
	}

	return logger
}

func buildCadenceClient() workflowserviceclient.Interface {
	ch, err := tchannel.NewChannelTransport(tchannel.ServiceName(ClientName))
	if err != nil {
		panic("Failed to setup tchannel")
	}
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: ClientName,
		Outbounds: yarpc.Outbounds{
			CadenceService: {Unary: ch.NewSingleOutbound(HostPort)},
		},
	})
	if err := dispatcher.Start(); err != nil {
		panic("Failed to start dispatcher")
	}

	return workflowserviceclient.New(dispatcher.ClientConfig(CadenceService))
}

func startWorker(logger *zap.Logger, service workflowserviceclient.Interface) {
	// TaskListName identifies set of client workflows, activities, and workers.
	// It could be your group or client or application name.
	workerOptions := worker.Options{
		Logger:       logger,
		MetricsScope: tally.NewTestScope(wf.ApplicationName, map[string]string{}),
	}

	worker := worker.New(
		service,
		Domain,
		wf.ApplicationName,
		workerOptions)

	worker.RegisterWorkflowWithOptions(wf.SignalWorkflow, workflow.RegisterOptions{Name: wf.WorkflowName})
	worker.RegisterActivity(wf.SignalActivity)

	err := worker.Start()
	if err != nil {
		panic("Failed to start worker")
	}

	logger.Info("Started Worker.", zap.String("worker", wf.ApplicationName))
}
