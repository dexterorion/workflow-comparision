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
	"go.uber.org/cadence/client"
	"go.uber.org/cadence/worker"
	"go.uber.org/cadence/workflow"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/zap"

	wf "avenuesec/workflow-poc/cadence/signal/workflow"
	"avenuesec/workflow-poc/cadence/transfer/business"
	"avenuesec/workflow-poc/cadence/transfer/handlers"
)

var HostPort = "127.0.0.1:7933"
var Domain = "simpledomain"
var ClientName = "simpleworker"
var CadenceService = "cadence-frontend"
var CadenceClientName = "cadence-client"

func main() {
	var mode string
	var text string
	var executionID string
	flag.StringVar(&mode, "m", "trigger", "Mode is worker, trigger or shadower.")
	flag.StringVar(&executionID, "id", "id", "Workflow exection ID.")
	flag.StringVar(&text, "text", "Anything", "Text to be shown. Exit finishes workflow")
	flag.Parse()

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

func getHandler(service workflowserviceclient.Interface) *mux.Router {
	svc := business.NewWorkflowBusiness(service, Domain)

	r := handlers.NewHandler(svc)

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
		Handler:      withCors(getHandler(service)), // Pass our instance of gorilla/mux in.
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

func sendSignal(service workflowserviceclient.Interface, executionID, text string) {
	ctx := context.Background()

	logger := buildLogger()

	var workflowClient client.Client = client.NewClient(
		service, Domain, &client.Options{Identity: "local-mac-vinny", MetricsScope: tally.NoopScope, ContextPropagators: []workflow.ContextPropagator{}})

	err := workflowClient.SignalWorkflow(ctx, executionID, "", wf.SignalName, text)

	if err != nil {
		logger.Error("Failed to send signal", zap.Error(err))
		panic("Failed to send signal.")
	} else {
		logger.Info("Signal sent", zap.String("WorkflowID", executionID), zap.String("Text", text))
	}
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
		MetricsScope: tally.NewTestScope(wf.ApplicationName, map[string]string{}),
	}

	worker := worker.New(
		service,
		Domain,
		wf.ApplicationName,
		workerOptions)

	worker.RegisterWorkflowWithOptions(wf.SignalWorkflow, workflow.RegisterOptions{Name: wf.WorkflowName})
	worker.RegisterActivity(wf.SignalActivity)

	err := worker.Start()
	if err != nil {
		panic("Failed to start worker")
	}

	logger.Info("Started Worker.", zap.String("worker", wf.ApplicationName))
}
