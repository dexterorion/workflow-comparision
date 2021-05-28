package business

import (
	"avenuesec/workflow-poc/cadence/transfer/rabbitmq"

	"go.uber.org/zap"
)

type ApexService interface{}

type apexServiceImpl struct {
	rabbit rabbitmq.AmqpConnection
	logger *zap.SugaredLogger
}

func ApexBinService(rabbit rabbitmq.AmqpConnection) ApexService {
	logger, _ := zap.NewProduction()
	logger = logger.Named("apex_service")

	return &apexServiceImpl{
		rabbit: rabbit,
		logger: logger.Sugar(),
	}
}
