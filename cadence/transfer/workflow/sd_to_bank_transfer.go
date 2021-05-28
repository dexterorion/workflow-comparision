package workflow

import (
	"avenuesec/workflow-poc/cadence/transfer/business"
	pb "avenuesec/workflow-poc/cadence/transfer/common/protogen"
	"fmt"

	"context"
	"time"

	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

type SdToBankWorkflow struct {
	service business.SdToBankService
	balance business.BalanceService
	account business.AccountService
	logger  *zap.SugaredLogger
}

func NewSdToBankWorkflow(service business.SdToBankService, balance business.BalanceService, account business.AccountService) SdToBankWorkflow {
	return SdToBankWorkflow{
		service: service,
		account: account,
		balance: balance,
	}
}

// SdToBankWorkflow workflow decider
func (s *SdToBankWorkflow) SdToBankWorkflow(ctx workflow.Context) error {
	s.logger = workflow.GetLogger(ctx).Sugar()
	s.logger.Info("SdToBank workflow started")

	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: time.Minute,
		StartToCloseTimeout:    time.Minute,
		HeartbeatTimeout:       time.Second * 20,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	ch := workflow.GetSignalChannel(ctx, business.SdToBankSignalName)
	for {
		var signal business.SignalTrigger
		var result string
		var err error

		if more := ch.Receive(ctx, &signal); !more {
			s.logger.Info("SdToBank channel closed")
			return cadence.NewCustomError("sd_to_bank_channel_closed")
		}

		s.logger.Info("Signal received.", zap.String("signal", string(signal)))

		executionID := workflow.GetInfo(ctx).WorkflowExecution.ID
		transfer, err := s.service.GetTransferInformation(context.Background(), executionID)
		if err != nil {
			s.logger.Error("SdToBankWorkflow failed to get transfer message.", zap.Error(err))
			return err
		}

		switch signal {
		case business.SdToBankSignalStartValidate:
			err = workflow.ExecuteActivity(ctx, s.Validate, transfer).Get(ctx, &result)
		case business.SdToBankSignalStartBlock:
			err = workflow.ExecuteActivity(ctx, s.Block, transfer).Get(ctx, &result)
		case business.SdToBankSignalStartJournal:
			err = workflow.ExecuteActivity(ctx, s.JournalWithdraw, transfer).Get(ctx, &result)
		case business.SdToBankSignalStartUnblockDebit:
			err = workflow.ExecuteActivity(ctx, s.UnblockDebit, transfer).Get(ctx, &result)
		case business.SdToBankSignalStartCredit:
			err = workflow.ExecuteActivity(ctx, s.Credit, transfer).Get(ctx, &result)
		case business.SdToBankSignalDone:
			s.logger.Info("SdToBankWorkflow completed.", zap.String("Result", result))
			return nil
		}

		if err != nil {
			s.logger.Error("SdToBankWorkflow failed.", zap.Error(err))
			return err
		}
	}
}

func (s *SdToBankWorkflow) Validate(ctx context.Context, msg *pb.Transfer) (string, error) {
	s.logger.Info("Validating Transfer request")

	accInfo, err := s.account.GetAccount(msg.AccId)
	if err != nil {
		s.logger.Errorw("Error getting account", "acc_id", msg.AccId, "err", err)
		return "error_account", err
	}

	fromAccId := accInfo.AccountUsId

	balance, err := s.balance.GetBalance(fromAccId)
	if err != nil {
		s.logger.Errorw("Error getting balance", "acc_id", fromAccId, "err", err)
		return "error_balance", err
	}

	if balance.Available < msg.Amount {
		s.logger.Errorw("Balance is not enough", "required", msg.Amount, "available", balance.Available, "account id", balance.AccountId)
		return "not_enough_balance", fmt.Errorf("not enough balance")
	}

	s.logger.Infow("Account has balance to perform operation", "account", balance.AccountId, "amount", msg.Amount)

	return "has_balance", nil
}

func (s *SdToBankWorkflow) Block(ctx context.Context, msg *pb.Transfer) (string, error) {
	return "", nil
}

func (s *SdToBankWorkflow) JournalWithdraw(ctx context.Context, msg *pb.Transfer) (string, error) {
	return "", nil
}

func (s *SdToBankWorkflow) UnblockDebit(ctx context.Context, msg *pb.Transfer) (string, error) {
	return "", nil
}

func (s *SdToBankWorkflow) Credit(ctx context.Context, msg *pb.Transfer) (string, error) {
	return "", nil
}
