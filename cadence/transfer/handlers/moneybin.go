package handlers

import (
	"avenuesec/workflow-poc/cadence/transfer/business"
	pb "avenuesec/workflow-poc/cadence/transfer/common/protogen"
	"avenuesec/workflow-poc/cadence/transfer/rabbitmq"
	"fmt"
	"log"
	"reflect"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
)

type MoneyBinConsumer interface {
}

type moneyBinConsumerImpl struct {
	rabbit  rabbitmq.AmqpConnection
	service business.MoneyBinService
	logger  *zap.SugaredLogger
}

func NewMoneyBinConsumer(rabbit rabbitmq.AmqpConnection, service business.AccountService) MoneyBinConsumer {
	logger, _ := zap.NewProduction()
	logger = logger.Named("moneybin_consumer")

	consumer := &moneyBinConsumerImpl{
		rabbit:  rabbit,
		service: service,
		logger:  logger.Sugar(),
	}

	go consumer.installHandlers()

	return consumer
}

func (c *moneyBinConsumerImpl) installHandlers() {
	consumer := c.rabbit.GetChannel()

	msgs, err := consumer.Consume(
		c.rabbit.GetQueue(), // queue
		"",                  // consumer
		true,                // auto-ack
		false,               // exclusive
		false,               // no-local
		false,               // no-wait
		nil,                 // args
	)

	if err != nil {
		c.logger.DPanicw("Deu ruim", "err", err)
	}

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s. Type: %s", d.Body, d.Type)

			messageType := proto.MessageType(d.Type)

			if messageType == nil {
				errorMsg := fmt.Sprintf("Unknown message type: %s - did you initialize protobuf?", d.Type)

				c.logger.Errorw("error deserializing proto message", "err", errorMsg)

				return
			}

			v := reflect.New(messageType.Elem())
			message := v.Interface().(proto.Message)

			err = proto.Unmarshal(d.Body, message)
			if err != nil {
				c.logger.Errorw("error unmarshalling message", "err", err)
				return
			}

			switch messageType {
			case reflect.TypeOf(&pb.AddEntry{}):

			}
		}
	}()

	c.logger.Info(" [*] MoneyBin Waiting for messages. To exit press CTRL+C")
	<-forever
}
