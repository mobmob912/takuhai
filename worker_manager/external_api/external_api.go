package external_api

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/mobmob912/takuhai/worker_manager/worker"

	"github.com/mobmob912/takuhai/domain"

	"github.com/mobmob912/takuhai/worker_manager/store"

	"github.com/go-chi/chi"
)

// external_apiは、ノード外から叩かれるAPI

type Server interface {
	Serve() error
	IsServing() bool
}

type server struct {
	// TODO workflowStoreやjobStoreでごにょごにょするのはworkerServiceの責務
	workflowStore store.Workflow
	jobStore      store.Job
	workerService *worker.Worker
	jobs          map[string]string
	isServing     bool
}

type Job struct {
	Name string
}

func NewServer(ws store.Workflow, as store.Job, wks *worker.Worker) Server {
	return &server{
		workflowStore: ws,
		jobStore:      as,
		workerService: wks,
	}
}

func (s *server) Serve() error {
	r := chi.NewRouter()

	r.Get("/check", s.healthCheck)

	// TriggerTypeHTTPのワークフローを開始する
	r.Post("/trigger/*", s.startWorkflow)

	//// jobをデプロイする from master
	//r.Post("/workflows/{workflowID}/steps/{stepID}/deploy", s.deployJob)

	// ワークフロー内の特定のタスクを実行する masterから叩かれる
	r.Post("/workflows/{workflowID}/steps/{stepID}", s.runJob)

	// Masterからワークフロー情報更新で叩かれる
	r.Put("/workflows", s.updateWorkflows)

	r.Get("/workflows/{workflowID}/steps/status", s.listStepStatus)

	log.SetPrefix("[External-API]: ")
	log.Println("Serving...")
	s.isServing = true
	return http.ListenAndServe(":4871", r)
}

func (s *server) IsServing() bool {
	return s.isServing
}

func (s *server) healthCheck(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("ok"))
}

func (s *server) startWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	triggerPath := "/" + chi.URLParam(r, "*")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		respondError(w, err, http.StatusBadRequest)
		return
	}
	if err := s.workerService.StartJobByTriggerHTTPPath(ctx, triggerPath, body); err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}
	respondSuccess(w, http.StatusCreated, nil)
}

func (s *server) runJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	workflowID := chi.URLParam(r, "workflowID")
	stepID := chi.URLParam(r, "stepID")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		respondError(w, err, http.StatusBadRequest)
		return
	}
	if err := s.workerService.RunJob(ctx, workflowID, stepID, body); err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}
	respondSuccess(w, http.StatusCreated, nil)
}

func (s *server) updateWorkflows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var flows []*domain.Workflow
	if err := json.NewDecoder(r.Body).Decode(&flows); err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}
	for _, f := range flows {
		s.workflowStore.Set(ctx, f.ID, f)
	}
	log.Println("Update workflows")
	log.Println(flows)
	respondSuccess(w, http.StatusNoContent, nil)
}

func (s *server) deployJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	workflowID := chi.URLParam(r, "workflowID")
	stepID := chi.URLParam(r, "stepID")
	var flow domain.Job
	if err := json.NewDecoder(r.Body).Decode(&flow); err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}
	go s.workerService.DeployJob(ctx, workflowID, stepID)
	respondSuccess(w, http.StatusNoContent, nil)
}

func (s *server) listStepStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	workflowID := chi.URLParam(r, "workflowID")
	ss, err := s.workerService.ListStepStatuses(ctx, workflowID)
	if err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}
	respondSuccess(w, http.StatusOK, ss)
}
