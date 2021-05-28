package business

import (
	pb "avenuesec/workflow-poc/cadence/transfer/common/protogen"
	"avenuesec/workflow-poc/cadence/transfer/redis"
	"context"
	"fmt"
	"time"

	"github.com/pborman/uuid"
	"google.golang.org/protobuf/proto"

	"go.uber.org/zap"
)

var accounts []string = []string{
	"8fd87578-c75e-4ae2-b7d0-f513a2325e1d",
	"f0378bd0-f123-48da-97f3-99792534da77",
	"f6d5fe4d-ee1f-4df4-9e23-10c0106e6b7d",
	"3977a8bf-73c7-4468-b4c4-7fe3f74ba3c4",
	"c523cd3f-14e8-41e8-94c9-963f57cb2e92",
	"d8bd66ba-9987-4e27-a36d-3020c525b146",
	"65496621-3b69-4af2-9f97-b853e1df5393",
	"f22e370a-09ef-4df9-836b-c710eb026f76",
	"87d7c585-e4a5-419e-b024-07f331f6e16e",
	"6ff38d11-77db-4e01-8be6-f72b6311b8ca",
}

type AccountService interface {
	GetAccount(id string) (*pb.AccountInformation, error)
}

type accountServiceImpl struct {
	redis  redis.RedisConnection
	logger *zap.SugaredLogger
	ch     chan *pb.AccountInformation
}

func NewAccountService(redis redis.RedisConnection, accountChannel chan *pb.AccountInformation) AccountService {
	logger, _ := zap.NewProduction()
	logger = logger.Named("account_service")

	svc := &accountServiceImpl{
		redis:  redis,
		logger: logger.Sugar(),
		ch:     accountChannel,
	}

	go svc.startAccounts()

	return svc
}

func (s *accountServiceImpl) startAccounts() {
	for _, a := range accounts {
		ae, err := s.accountExists(a)
		if err != nil {
			s.logger.Errorw("Error verifying account exists", "err", err)
			break
		}

		if !ae {
			s.logger.Infow("Account does not exist. Creating...", "account", a)

			err = s.createAccount(a)
			if err != nil {
				s.logger.Errorw("Error creating account", "err", err)
			} else {
				s.logger.Infow("Account created", "account", a)
			}
		}
	}
}

func (s *accountServiceImpl) accountExists(id string) (bool, error) {
	_, err := s.redis.GetConn().Get(context.Background(), fmt.Sprintf("account_%s", id)).Result()

	if s.redis.NoKeyError(err) {
		return false, nil
	}

	if err != nil {
		s.logger.Errorw("Error getting from redis", "err", err)
		return false, err
	}

	return true, nil
}

func (s *accountServiceImpl) createAccount(id string) error {
	acc := &pb.AccountInformation{
		AccountUsId:   uuid.New(),
		AccountBankId: uuid.New(),
	}

	str, err := proto.Marshal(acc)
	if err != nil {
		return err
	}

	status := s.redis.GetConn().Set(context.Background(), fmt.Sprintf("account_%s", id), str, time.Duration(1000*time.Hour))
	if status != nil && status.Err() != nil {
		return status.Err()
	}

	s.ch <- acc

	return nil
}

func (s *accountServiceImpl) GetAccount(id string) (*pb.AccountInformation, error) {
	result, err := s.redis.GetConn().Get(context.Background(), fmt.Sprintf("account_%s", id)).Result()
	if err != nil {
		return nil, err
	}

	var acc pb.AccountInformation
	err = proto.Unmarshal([]byte(result), &acc)

	return &acc, err
}
