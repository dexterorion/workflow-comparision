package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"github.com/uber-go/tally"
	"go.uber.org/cadence/.gen/go/cadence/workflowserviceclient"
	"go.uber.org/cadence/worker"
	"go.uber.org/cadence/workflow"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/zap"

	"avenuesec/workflow-poc/cadence/transfer/business"
	pb "avenuesec/workflow-poc/cadence/transfer/common/protogen"
	"avenuesec/workflow-poc/cadence/transfer/handlers"
	"avenuesec/workflow-poc/cadence/transfer/helpers/model"
	"avenuesec/workflow-poc/cadence/transfer/helpers/security"
	"avenuesec/workflow-poc/cadence/transfer/rabbitmq"
	"avenuesec/workflow-poc/cadence/transfer/redis"
	wf "avenuesec/workflow-poc/cadence/transfer/workflow"
)

var HostPort = "cadence-frontend:7933"
var Domain = "simpledomain"
var ClientName = "simpleworker"
var CadenceService = "cadence-frontend"
var CadenceClientName = "cadence-client"

var (
	flagUser  string
	flagPwd   string
	flagVHost string
	flagHost  string
	flagPort  = 5672
	mode      string
)

func InitWithFlagSet(flagSet *flag.FlagSet) {
	flagSet.StringVar(&flagUser, "rmq_user", "guest", "")
	flagSet.StringVar(&flagPwd, "rmq_pwd", "guest", "")
	flagSet.StringVar(&flagVHost, "rmq_vhost", "avenue", "")
	flagSet.StringVar(&flagHost, "rmq_host", "localhost", "")
	flagSet.IntVar(&flagPort, "rmq_port", 5672, "")
}

func GetEnvOrDefault(key string, defaultValue string) string {
	val := os.Getenv(key)

	if val == "" {
		return defaultValue
	}

	return val
}

func AmqpConfigFromFlags() model.AmqpConfig {

	pwd := security.DecryptIf(GetEnvOrDefault("AVENUE_GCLOUD_ID", "trading-dev-201715"), flagPwd)

	return model.AmqpConfig{
		User:     flagUser,
		Password: pwd,
		Host:     flagHost,
		Port:     flagPort,
		VHost:    flagVHost,
	}
}

func InitFromEnvSet() {
	flagUser = GetEnvOrDefault("rmq_user", "guest")
	flagPwd = GetEnvOrDefault("rmq_pwd", "guest")
	flagVHost = GetEnvOrDefault("rmq_vhost", "avenue")
	flagHost = GetEnvOrDefault("rmq_host", "rabbitmq")
}

func init() {
	flag.StringVar(&mode, "m", "trigger", "Mode is worker, trigger or shadower.")
	InitWithFlagSet(flag.CommandLine)
	flag.Parse()
}

func main() {
	switch mode {
	case "worker":
		startWorker(buildLogger(), buildCadenceClient())

		// The workers are supposed to be long running process that should not exit.
		// Use select{} to block indefinitely for samples, you can quit by CMD+C.
		select {}

	case "server":
		startServer(buildCadenceClient())
	}
}

func getHandler(service workflowserviceclient.Interface, amqpConfig model.AmqpConfig) *mux.Router {
	rabbit := rabbitmq.GetConnection(amqpConfig)
	rd := redis.NewRedisConnection()

	r := handlers.NewHandler(rabbit)

	bizz := business.NewSdToBankService(rabbit, rd, service, Domain)
	handlers.NewConsumer(rabbit, bizz)

	return r.GetRouter()
}

func withCors(r *mux.Router) http.Handler {
	// For dev only - Set up CORS so React client can consume our API
	corsWrapper := cors.New(cors.Options{
		AllowedMethods: []string{"GET", "POST", "PUT", "OPTIONS", "DELETE", "HEAD"},
		AllowedHeaders: []string{"Content-Type", "Origin", "Accept", "*"},
	})

	return corsWrapper.Handler(r)
}

func startServer(service workflowserviceclient.Interface) {
	// amqpConfig := AmqpConfigFromFlags()

	logger := buildLogger()
	defer logger.Sync()

	sugar := logger.Sugar()

	sugar.Infow("Starting service...")

	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

	srv := &http.Server{
		Addr: "0.0.0.0:8080",
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler: withCors(getHandler(service, model.AmqpConfig{
			User:     "guest",
			Password: "guest",
			VHost:    "avenue",
			Host:     "rabbitmq",
			Port:     5672,
		})), // Pass our instance of gorilla/mux in.
	}

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	sugar.Infow("Listening on port :8080")

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	sugar.Infow("shutting down")
	os.Exit(0)
}

func buildLogger() *zap.Logger {
	config := zap.NewDevelopmentConfig()

	var err error
	logger, err := config.Build()
	if err != nil {
		panic("Failed to setup logger")
	}

	return logger
}

func buildCadenceClient() workflowserviceclient.Interface {
	ch, err := tchannel.NewChannelTransport(tchannel.ServiceName(ClientName))
	if err != nil {
		panic("Failed to setup tchannel")
	}
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: ClientName,
		Outbounds: yarpc.Outbounds{
			CadenceService: {Unary: ch.NewSingleOutbound(HostPort)},
		},
	})
	if err := dispatcher.Start(); err != nil {
		panic("Failed to start dispatcher")
	}

	return workflowserviceclient.New(dispatcher.ClientConfig(CadenceService))
}

func startWorker(logger *zap.Logger, service workflowserviceclient.Interface) {
	// TaskListName identifies set of client workflows, activities, and workers.
	// It could be your group or client or application name.
	workerOptions := worker.Options{
		Logger:       logger,
		MetricsScope: tally.NewTestScope(business.SdToBankApplicationName, map[string]string{}),
	}

	worker := worker.New(
		service,
		Domain,
		business.SdToBankApplicationName,
		workerOptions)

	rabbit := rabbitmq.GetConnection(model.AmqpConfig{
		User:     "guest",
		Password: "guest",
		VHost:    "avenue",
		Host:     "rabbitmq",
		Port:     5672,
	})
	rd := redis.NewRedisConnection()
	bizz := business.NewSdToBankService(rabbit, rd, service, Domain)

	accCh := make(chan *pb.AccountInformation)

	balSvc := business.NewBalanceService(rd, accCh)
	accSvc := business.NewAccountService(rd, accCh)

	sdToBankWf := wf.NewSdToBankWorkflow(bizz, balSvc, accSvc)

	worker.RegisterWorkflowWithOptions(sdToBankWf.SdToBankWorkflow, workflow.RegisterOptions{Name: business.SdToBankWorkflowName})
	worker.RegisterActivity(sdToBankWf.Block)
	worker.RegisterActivity(sdToBankWf.Credit)
	worker.RegisterActivity(sdToBankWf.JournalWithdraw)
	worker.RegisterActivity(sdToBankWf.UnblockDebit)
	worker.RegisterActivity(sdToBankWf.Validate)

	err := worker.Start()
	if err != nil {
		panic("Failed to start worker")
	}

	logger.Info("Started Worker.", zap.String("worker", business.SdToBankApplicationName))
}
