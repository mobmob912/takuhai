package internal_api

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/mobmob912/takuhai/worker_manager/worker"

	"github.com/go-chi/chi"
)

// internal_apiは、ノード内で稼働しているアプリケーションから叩かれるAPI

type Server interface {
	Serve() error
}

type server struct {
	workerService *worker.Worker
}

func NewServer(w *worker.Worker) Server {
	return &server{
		workerService: w,
	}
}


func (s *server) Serve() error {
	r := chi.NewRouter()
	r.Post("/workflows/{workflowID}/steps/{stepID}", s.jobDeployed)
	r.Get("/hello", s.hello)
	r.Post("/workflows/{workflowID}/steps/{stepID}/next", s.next)
	r.Post("/workflows/{workflowID}/steps/{stepID}/fail", s.fail)
	r.Post("/workflows/{workflowID}/steps/{stepID}/finish", s.finish)

	log.SetPrefix("[Internal-API]: ")
	log.Println("Serving...")
	return http.ListenAndServe(":2317", r)
}
func getBody(r *http.Request) (buf []byte, err error) {
	buf, err = ioutil.ReadAll(r.Body)
	defer func() { _ = r.Body.Close() }()
	return
}

func (s *server) next(w http.ResponseWriter, r *http.Request) {
	// workflowみて、次の処理へ飛ばす
	workflowID := chi.URLParam(r, "workflowID")
	stepID := chi.URLParam(r, "stepID")
	jobID := r.Header.Get("takuhai-job-id")
	buf, err := getBody(r)
	if err != nil{
		log.Println(err)}
	
	/*wkr := &Content{}
	if err := json.Unmarshal(buf, wkr); err != nil {
		sendResponse(w, http.StatusBadRequest, nil)
		return err
	}
	body := wkr.Body
	log.Println(body)
	log.Println(wkr.Runtime)
	log.Println(wkr.RAM)*/
	/*body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}*/
	go func() {
		if err := s.workerService.NextJob(context.Background(), workflowID, stepID, jobID, buf); err != nil {
			log.Println(err)
		}
	}()
	w.WriteHeader(200)
}
func (s *server) hello(w http.ResponseWriter, r *http.Request) {
	log.Println("hello come on!")
	
}

// jobがデプロイされたのをjob側から通知してもらう
func (s *server) jobDeployed(w http.ResponseWriter, r *http.Request) {
	log.Println("receive deployed msg")
	ctx := r.Context()
	stepID := chi.URLParam(r, "stepID")
	_, err := s.workerService.RegisterDeployedJob(ctx, stepID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *server) fail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	workflowID := chi.URLParam(r, "workflowID")
	stepID := chi.URLParam(r, "stepID")
	jobID := r.Header.Get("takuhai-job-id")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	go s.workerService.FailJob(ctx, workflowID, stepID, jobID, body)
}

func (s *server) finish(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	workflowID := chi.URLParam(r, "workflowID")
	stepID := chi.URLParam(r, "stepID")
	jobID := r.Header.Get("takuhai-job-id")
	go s.workerService.FinishJob(ctx, workflowID, stepID, jobID)
}
