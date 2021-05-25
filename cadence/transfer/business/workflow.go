package business

import (
	"context"
	"encoding/json"
	"log"
	"time"

	wf "avenuesec/workflow-poc/cadence/signal/workflow"
	"avenuesec/workflow-poc/cadence/transfer/rabbitmq"

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
	rabbit  rabbitmq.AmqpConnection
	domain  string
}

func NewWorkflowBusiness(service workflowserviceclient.Interface, domain string) WorkflowBusiness {
	logger, _ := zap.NewProduction()
	logger = logger.Named("workflow")

	bizz := &workflowBusinessImpl{
		logger:  logger.Sugar(),
		service: service,
		domain:  domain,
		rabbit:  rabbitmq.GetConnection(),
	}

	go bizz.installHandler()

	return bizz
}

type Message struct {
	ExecutionID string
	Text        string
}

func (s *workflowBusinessImpl) installHandler() {
	consumer := s.rabbit.GetChannel()

	msgs, err := consumer.Consume(
		s.rabbit.GetQueue(), // queue
		"",                  // consumer
		true,                // auto-ack
		false,               // exclusive
		false,               // no-local
		false,               // no-wait
		nil,                 // args
	)

	if err != nil {
		s.logger.DPanicw("Deu ruim", "err", err)
	}

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)

			var msg Message
			err := json.Unmarshal(d.Body, &msg)
			if err != nil {
				s.logger.DPanicw("Error unmarshiling", "err", err)
			}

			switch msg.Text {
			case "doA":
				s.doA(msg.ExecutionID)
			case "doB":
				s.doB(msg.ExecutionID)
			case "doC":
				s.doC(msg.ExecutionID)
			case "finish":
				s.sendSignal(msg.ExecutionID, "exit")
			}
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func (s *workflowBusinessImpl) StartWorkflow(ctx context.Context) {
	workflowOptions := client.StartWorkflowOptions{
		ID:                              "transfer_" + uuid.New(),
		TaskList:                        wf.ApplicationName,
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

	s.rabbit.Produce(ctx, &Message{
		Text:        "doA",
		ExecutionID: we.ID,
	})

}

func (s *workflowBusinessImpl) doA(executionID string) {
	s.logger.Info("Sending signal to doA")
	s.sendSignal(executionID, "doA")
	s.logger.Info("Sending message to doB")
	s.rabbit.Produce(context.Background(), &Message{
		Text:        "doB",
		ExecutionID: executionID,
	})
}

func (s *workflowBusinessImpl) doB(executionID string) {
	s.logger.Info("Sending signal to doB")
	s.sendSignal(executionID, "doB")
	s.logger.Info("Sending message to doC")
	s.rabbit.Produce(context.Background(), &Message{
		Text:        "doC",
		ExecutionID: executionID,
	})
}

func (s *workflowBusinessImpl) doC(executionID string) {
	s.logger.Info("Sending signal to doC")
	s.sendSignal(executionID, "doC")
	s.logger.Info("Sending message to finish")
	s.rabbit.Produce(context.Background(), &Message{
		Text:        "finish",
		ExecutionID: executionID,
	})
}

func (s *workflowBusinessImpl) sendSignal(executionID, text string) {
	ctx := context.Background()

	var workflowClient client.Client = client.NewClient(
		s.service, s.domain, &client.Options{Identity: "local-mac-vinny", MetricsScope: tally.NoopScope, ContextPropagators: []workflow.ContextPropagator{}})

	err := workflowClient.SignalWorkflow(ctx, executionID, "", wf.SignalName, text)

	if err != nil {
		s.logger.Error("Failed to send signal", zap.Error(err))
		panic("Failed to send signal.")
	} else {
		s.logger.Info("Signal sent", zap.String("WorkflowID", executionID), zap.String("Text", text))
	}
}
