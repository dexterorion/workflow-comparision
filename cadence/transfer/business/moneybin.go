package business

import (
	"avenuesec/workflow-poc/cadence/transfer/rabbitmq"

	"go.uber.org/zap"
)

type MoneyBinService interface{}

type moneyBinServiceImpl struct {
	rabbit rabbitmq.AmqpConnection
	logger *zap.SugaredLogger
}

func MoneyBinBinService(rabbit rabbitmq.AmqpConnection) MoneyBinService {
	logger, _ := zap.NewProduction()
	logger = logger.Named("moneyBin_service")

	return &moneyBinServiceImpl{
		rabbit: rabbit,
		logger: logger.Sugar(),
	}
}
