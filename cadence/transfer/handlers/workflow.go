package handlers

import (
	"avenuesec/workflow-poc/cadence/transfer/business"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type WorkflowHandler interface {
	StartWorkflow() http.Handler
}

type workflowHandlerImpl struct {
	router   *mux.Router
	business business.WorkflowBusiness
}

func NewWorkflowHandler(router *mux.Router, business business.WorkflowBusiness) {
	handler := &workflowHandlerImpl{router, business}
	handler.buildRoutes()
}

func (p *workflowHandlerImpl) buildRoutes() {
	router := p.router.PathPrefix("/workflows").Subrouter()

	router.Handle("/start", p.StartWorkflow()).Methods("POST")
}

func (p *workflowHandlerImpl) StartWorkflow() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.business.StartWorkflow(r.Context())

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
