package business

import (
	"context"
	"time"

	wf "avenuesec/workflow-poc/cadence/signal/workflow"

	"go.uber.org/cadence/.gen/go/cadence/workflowserviceclient"

	"github.com/pborman/uuid"
	"github.com/uber-go/tally"
	"go.uber.org/cadence/client"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

// WorkflowBusiness represents the workflow business
type WorkflowBusiness interface {
	StartWorkflow(ctx context.Context)
}

type workflowBusinessImpl struct {
	logger  *zap.SugaredLogger
	service workflowserviceclient.Interface
	domain  string
}

func NewWorkflowBusiness(service workflowserviceclient.Interface, domain string) WorkflowBusiness {
	logger, _ := zap.NewProduction()
	logger = logger.Named("workflow")

	return &workflowBusinessImpl{
		logger:  logger.Sugar(),
		service: service,
		domain:  domain,
	}
}

func (s *workflowBusinessImpl) StartWorkflow(ctx context.Context) {
	workflowOptions := client.StartWorkflowOptions{
		ID:                              "transfer_" + uuid.New(),
		ExecutionStartToCloseTimeout:    time.Minute,
		DecisionTaskStartToCloseTimeout: time.Minute,
	}

	var workflowClient client.Client = client.NewClient(
		s.service, s.domain, &client.Options{Identity: "local-mac-vinny", MetricsScope: tally.NoopScope, ContextPropagators: []workflow.ContextPropagator{}})

	we, err := workflowClient.StartWorkflow(ctx, workflowOptions, wf.WorkflowName, "Cadence")
	if err != nil {
		s.logger.Error("Failed to create workflow", zap.Error(err))
		panic("Failed to create workflow.")
	} else {
		s.logger.Info("Started Workflow", zap.String("WorkflowID", we.ID), zap.String("RunID", we.RunID))
	}
}
