package business

import (
	pb "avenuesec/workflow-poc/cadence/transfer/common/protogen"
	"avenuesec/workflow-poc/cadence/transfer/redis"
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type BalanceService interface {
	GetBalance(id string) (*pb.BalanceInformation, error)
}

type balanceServiceImpl struct {
	redis  redis.RedisConnection
	logger *zap.SugaredLogger
	ch     chan *pb.AccountInformation
}

func NewBalanceService(redis redis.RedisConnection, accountChannel chan *pb.AccountInformation) BalanceService {
	logger, _ := zap.NewProduction()
	logger = logger.Named("balance_service")

	svc := &balanceServiceImpl{
		redis:  redis,
		logger: logger.Sugar(),
		ch:     accountChannel,
	}

	go svc.listenAccountCreation()

	return svc
}

func (s *balanceServiceImpl) listenAccountCreation() {
	for {
		newAccount := <-s.ch
		// create balance
		err := s.createBalance(newAccount)

		if err != nil {
			s.logger.Errorw("Error creating balance", "err", err)
		}
	}
}

func (s *balanceServiceImpl) createBalance(acc *pb.AccountInformation) error {
	// balance us
	bUs := &pb.BalanceInformation{
		AccountId: acc.AccountUsId,
		Available: 10000,
	}

	strUs, err := proto.Marshal(bUs)
	if err != nil {
		return err
	}

	status := s.redis.GetConn().Set(context.Background(), fmt.Sprintf("balance_%s", acc.AccountUsId), strUs, time.Duration(1000*time.Hour))
	if status != nil && status.Err() != nil {
		return status.Err()
	}

	// balance bank
	bBank := &pb.BalanceInformation{
		AccountId: acc.AccountBankId,
		Available: 10000,
	}

	strBank, err := proto.Marshal(bBank)
	if err != nil {
		return err
	}

	status = s.redis.GetConn().Set(context.Background(), fmt.Sprintf("balance_%s", acc.AccountBankId), strBank, time.Duration(1000*time.Hour))
	if status != nil && status.Err() != nil {
		return status.Err()
	}

	return nil
}

func (s *balanceServiceImpl) GetBalance(id string) (*pb.BalanceInformation, error) {
	result, err := s.redis.GetConn().Get(context.Background(), fmt.Sprintf("balance_%s", id)).Result()
	if err != nil {
		return nil, err
	}

	var b pb.BalanceInformation
	err = proto.Unmarshal([]byte(result), &b)

	return &b, err
}
