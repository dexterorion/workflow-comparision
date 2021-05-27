package handlers

import (
	"avenuesec/workflow-poc/cadence/transfer/rabbitmq"

	"github.com/gorilla/mux"
)

type Handler interface {
	GetRouter() *mux.Router
}

type handleImpl struct {
	router *mux.Router
}

func NewHandler(rabbit rabbitmq.AmqpConnection) Handler {
	router := mux.NewRouter().PathPrefix("/api").Subrouter()

	NewTransferHandler(router, rabbit)

	return &handleImpl{
		router,
	}
}

func (h *handleImpl) GetRouter() *mux.Router {
	return h.router
}
