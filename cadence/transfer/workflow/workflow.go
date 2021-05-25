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
const ApplicationName = "helloWorldGroup"

// WorkflowName ...
const WorkflowName = "helloWordWorkflow"

// SignalName is the signal name that workflow is waiting for
const SignalName = "trigger-signal"

// SignalWorkflow workflow decider
func SignalWorkflow(ctx workflow.Context, name string) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("signal workflow started")

	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: time.Minute,
		StartToCloseTimeout:    time.Minute,
		HeartbeatTimeout:       time.Second * 20,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var signalResult string
	err := workflow.ExecuteActivity(ctx, SignalActivity, name).Get(ctx, &signalResult)
	if err != nil {
		logger.Error("Activity failed.", zap.Error(err))
		return err
	}

	ch := workflow.GetSignalChannel(ctx, SignalName)
	for {
		var signal string
		if more := ch.Receive(ctx, &signal); !more {
			logger.Info("Signal channel closed")
			return cadence.NewCustomError("signal_channel_closed")
		}

		logger.Info("Signal received.", zap.String("signal", signal))

		if signal == "exit" {
			break
		}

		logger.Sugar().Infow("Data received via signal", "data", signal)
	}

	logger.Info("Workflow completed.", zap.String("Result", signalResult))

	return nil
}

func SignalActivity(ctx context.Context, name string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("signal activity started")
	return "Hello " + name + "!", nil
}
