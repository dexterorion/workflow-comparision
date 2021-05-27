package handlers

import (
	pb "avenuesec/workflow-poc/cadence/transfer/common/protogen"
	"avenuesec/workflow-poc/cadence/transfer/rabbitmq"
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type TransferHandler interface {
	StartTransfer() http.Handler
}

type transferHandlerImpl struct {
	router *mux.Router
	rabbit rabbitmq.AmqpConnection
}

func NewTransferHandler(router *mux.Router, rabbit rabbitmq.AmqpConnection) {
	handler := &transferHandlerImpl{router, rabbit}
	handler.buildRoutes()
}

func (p *transferHandlerImpl) buildRoutes() {
	router := p.router.PathPrefix("/transfers").Subrouter()

	router.Handle("/new", p.StartTransfer()).Methods("POST")
}

func (p *transferHandlerImpl) StartTransfer() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var body []byte
		var message *pb.NewTransferMessage

		_, err := r.Body.Read(body)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		err = json.NewDecoder(r.Body).Decode(&message)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		err = p.rabbit.ProduceStruct(context.Background(), message)

		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		w.WriteHeader(200)

		data := map[string]string{
			"status": "ok",
		}

		jso, err := json.Marshal(data)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		if data != nil {
			w.Header().Set("Content-Type", "application/json")
			w.Write(jso)
			return
		}
	})
}
