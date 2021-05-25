package handlers

import (
	"avenuesec/workflow-poc/cadence/transfer/business"

	"github.com/gorilla/mux"
)

type Handler interface {
	GetRouter() *mux.Router
}

type handleImpl struct {
	router *mux.Router
}

func NewHandler(svc business.WorkflowBusiness) Handler {
	router := mux.NewRouter().PathPrefix("/api").Subrouter()

	NewWorkflowHandler(router, svc)

	return &handleImpl{
		router,
	}
}

func (h *handleImpl) GetRouter() *mux.Router {
	return h.router
}
