package external_api

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"time"
	"bytes"
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
type WorkerResponse struct {
	Time time.Time
}
type AddWorkerReply struct {
	Time time.Duration `json:"time"`
}

func sendResponse(w http.ResponseWriter, status int, body []byte) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
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

	r.Post("/reply", s.replyToWorkerWorker)
	// ワークフロー内の特定のタスクを実行する masterから叩かれる
	r.Post("/workflows/{workflowID}/steps/{stepID}", s.runJob)
	
	r.Post("/register", s.registerWorker)

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
	//attime := time.Now() 
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
	/*res := WorkerResponse{Time: attime}
	reBody, err := json.Marshal(res)
	 if err != nil {
		sendResponse(w, http.StatusInternalServerError, nil)
	}
	sendResponse(w, http.StatusCreated, reBody)*/
	respondSuccess(w, http.StatusNoContent, nil)
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
func (s *server) registerWorker(w http.ResponseWriter, r *http.Request) {
	//ctx := r.Context()
	var flows worker.ToNode
	if err := json.NewDecoder(r.Body).Decode(&flows); err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}
	s.workerService.OtherWorkers = append(s.workerService.OtherWorkers,flows)
	for _, f :=  range s.workerService.OtherWorkers {
		log.Println(f.ID)	
		log.Println(f.Name)	
	}
	log.Println("Register worker")
	reqBody2, err := json.Marshal(flows)
	if err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}
	req2, err := http.NewRequest("POST", flows.URL.String()+"/reply", bytes.NewReader(reqBody2))
	if err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}
	client := http.DefaultClient
	start := time.Now() 
	
	_, err = client.Do(req2)
	if err != nil {
	respondError(w, err, http.StatusInternalServerError)
		return 
	}
	delay := time.Since(start)
	res3 := AddWorkerReply{Time: delay}
	resBody3, err := json.Marshal(res3)
	log.Println(string(resBody3))
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, nil)
	}
	sendResponse(w, http.StatusCreated, resBody3)
	
}

func (s *server)replyToWorkerWorker(w http.ResponseWriter, r *http.Request) {
        rebody :="hello" 
        reBody, err := json.Marshal(rebody)
        if err != nil {
		sendResponse(w, http.StatusInternalServerError, nil)
	}
	sendResponse(w, http.StatusCreated, reBody)
	return 

}
