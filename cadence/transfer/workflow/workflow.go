package workflow

import (
	"context"
	"time"

	"go.uber.org/cadence"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

/**
 * This is the hello world workflow sample.
 */

// ApplicationName is the task list for this sample
const ApplicationName = "transferGroup"

// WorkflowName ...
const WorkflowName = "transferWorkflow"

// TransferName is the transfer name that workflow is waiting for
const TransferName = "trigger-transfer"

// TransferWorkflow workflow decider
func TransferWorkflow(ctx workflow.Context, name string) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("transfer workflow started")

	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: time.Minute,
		StartToCloseTimeout:    time.Minute,
		HeartbeatTimeout:       time.Second * 20,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var transferResult string
	err := workflow.ExecuteActivity(ctx, TransferActivity, name).Get(ctx, &transferResult)
	if err != nil {
		logger.Error("Activity failed.", zap.Error(err))
		return err
	}

	ch := workflow.GetSignalChannel(ctx, TransferName)
	for {
		var transfer string
		if more := ch.Receive(ctx, &transfer); !more {
			logger.Info("Transfer channel closed")
			return cadence.NewCustomError("transfer_channel_closed")
		}

		logger.Info("Transfer received.", zap.String("transfer", transfer))

		if transfer == "exit" {
			break
		}

		logger.Sugar().Infow("Data received via transfer", "data", transfer)
	}

	logger.Info("Workflow completed.", zap.String("Result", transferResult))

	return nil
}

func TransferActivity(ctx context.Context, name string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("transfer activity started")
	return "Hello " + name + "!", nil
}
