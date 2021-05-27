package business

import (
	pb "avenuesec/workflow-poc/cadence/transfer/common/protogen"
	"avenuesec/workflow-poc/cadence/transfer/rabbitmq"
	"avenuesec/workflow-poc/cadence/transfer/redis"
	"context"
	"fmt"
	"time"

	"github.com/pborman/uuid"
	"github.com/uber-go/tally"
	"go.uber.org/cadence/.gen/go/cadence/workflowserviceclient"
	"go.uber.org/cadence/client"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type SignalTrigger string

const (
	SdToBankSignalStartValidate     SignalTrigger = "trigger-sdtobank-start-validate"
	SdToBankSignalStartBlock        SignalTrigger = "trigger-sdtobank-start-block"
	SdToBankSignalStartJournal      SignalTrigger = "trigger-sdtobank-start-journal"
	SdToBankSignalStartUnblockDebit SignalTrigger = "trigger-sdtobank-start-unblock-debit"
	SdToBankSignalStartCredit       SignalTrigger = "trigger-sdtobank-start-credit"
	SdToBankSignalDone              SignalTrigger = "trigger-sdtobank-done"

	SdToBankApplicationName = "sdToBankTransferGroup"
	SdToBankWorkflowName    = "sdToBankTransferWorkflow"
	SdToBankSignalName      = "sdToBankSignal"
)

type SdToBankService interface {
	StartTransfer(ctx context.Context, message *pb.NewTransferMessage) error
	GetTransferInformation(ctx context.Context, workflowID string) (*pb.Transfer, error)
}

type sdToBankServiceImpl struct {
	rabbit rabbitmq.AmqpConnection
	redis  redis.RedisConnection
	wf     workflowserviceclient.Interface
	logger *zap.SugaredLogger
	domain string
}

func NewSdToBankService(rabbit rabbitmq.AmqpConnection, redis redis.RedisConnection, wf workflowserviceclient.Interface, domain string) SdToBankService {
	logger, _ := zap.NewProduction()
	logger = logger.Named("sdtobank_service")

	return &sdToBankServiceImpl{
		rabbit: rabbit,
		redis:  redis,
		wf:     wf,
		domain: domain,
		logger: logger.Sugar(),
	}
}

func (s *sdToBankServiceImpl) StartTransfer(ctx context.Context, message *pb.NewTransferMessage) error {
	workflowOptions := client.StartWorkflowOptions{
		ID:                              "sdtobank_" + uuid.New(),
		TaskList:                        SdToBankApplicationName,
		ExecutionStartToCloseTimeout:    time.Minute,
		DecisionTaskStartToCloseTimeout: time.Minute,
	}

	var workflowClient client.Client = client.NewClient(
		s.wf, s.domain, &client.Options{Identity: "local-mac-vinny", MetricsScope: tally.NoopScope, ContextPropagators: []workflow.ContextPropagator{}})

	we, err := workflowClient.StartWorkflow(ctx, workflowOptions, SdToBankWorkflowName, "SdToBank")
	if err != nil {
		s.logger.Error("Failed to create SdToBankWorkflow", zap.Error(err))
		panic("Failed to create SdToBankWorkflow.")
	} else {
		s.logger.Info("Started SdToBankWorkflow", zap.String("WorkflowID", we.ID), zap.String("RunID", we.RunID))
	}

	// store transfer message information
	err = s.newTransfer(ctx, message, we.ID)
	if err != nil {
		s.logger.Error("Failed to store transfer", zap.Error(err))
		return workflowClient.CancelWorkflow(ctx, we.ID, we.RunID)
	}

	return s.sendSignal(ctx, we.ID, string(SdToBankSignalStartValidate))
}

func (s *sdToBankServiceImpl) GetTransferInformation(ctx context.Context, workflowID string) (*pb.Transfer, error) {
	result, err := s.redis.Conn.Get(ctx, fmt.Sprintf("sdtobank_%s", workflowID)).Result()
	if err != nil {
		return nil, err
	}

	var msg pb.Transfer
	err = proto.Unmarshal([]byte(result), &msg)

	return &msg, err
}

func (s *sdToBankServiceImpl) newTransfer(ctx context.Context, message *pb.NewTransferMessage, workflowID string) error {
	transfer := &pb.Transfer{
		Amount:      message.Amount,
		FromAccId:   message.FromAccId,
		ToAccId:     message.ToAccId,
		ExecutionId: message.ExecutionId,
		Direction:   message.Direction,
		Status:      "starting",
	}

	str, err := proto.Marshal(transfer)
	if err != nil {
		return err
	}

	status := s.redis.Conn.Set(ctx, fmt.Sprintf("sdtobank_%s", workflowID), str, time.Duration(1000*time.Hour))
	if status != nil && status.Err() != nil {
		return status.Err()
	}

	return nil
}

func (s *sdToBankServiceImpl) sendSignal(ctx context.Context, executionID, text string) error {
	var workflowClient client.Client = client.NewClient(
		s.wf, s.domain, &client.Options{Identity: "local-mac-vinny", MetricsScope: tally.NoopScope, ContextPropagators: []workflow.ContextPropagator{}})

	err := workflowClient.SignalWorkflow(ctx, executionID, "", SdToBankSignalName, text)

	if err != nil {
		s.logger.Error("Failed to send signal", zap.Error(err))
		return err
	}

	s.logger.Info("Signal sent", zap.String("WorkflowID", executionID), zap.String("Text", text))
	return nil
}
