package api

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/mobmob912/takuhai/domain"

	"github.com/mobmob912/takuhai/master/master"

	"github.com/go-chi/chi"
)

type Server struct {
	master *master.Master
}

func NewServer(ma *master.Master) *Server {
	return &Server{
		master: ma,
	}
}

const (
	GET    = "GET"
	POST   = "POST"
	PUT    = "PUT"
	DELETE = "DELETE"
)

func (s *Server) Serve() error {
	r := chi.NewRouter()

	r.Method(GET, "/check", handler(s.check))

	r.Method(GET, "/workers", handler(s.listWorkers))
	r.Method(POST, "/workers", handler(s.addWorker))
	r.Method(PUT, "/workers/{workerID}", handler(s.updateWorkerResource))

	r.Method(GET, "/workflows", handler(s.listWorkflows))
	r.Method(POST, "/workflows", handler(s.addWorkflow))
	r.Method(GET, "/workflows/{workflowID}/steps/{stepID}/worker", handler(s.nextJobWorker))

	// とりま何もしない. ログ集めとかする
	r.Method(POST, "/workflows/{workflowID}/steps/{stepID}/fail", handler(s.fail))
	r.Method(GET, "/workflows/{workflowName}/status", handler(s.getStatusOfWorkflow))

	return http.ListenAndServe(":3000", r)
}

func sendResponse(w http.ResponseWriter, status int, body []byte) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func (s *Server) listWorkers(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	ns, err := s.master.Workers(ctx)
	nrs := make([]*WorkerInfoResponse, len(ns))
	for i, n := range ns {
		nrs[i] = WorkerInfoResponseFromWorker(n)
	}
	nsBuf, err := json.Marshal(nrs)
	if err != nil {
		return err
	}
	sendResponse(w, http.StatusOK, nsBuf)
	return nil
}

func getBody(r *http.Request) (buf []byte, err error) {
	buf, err = ioutil.ReadAll(r.Body)
	defer func() { _ = r.Body.Close() }()
	return
}

func (s *Server) addWorker(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	log.Println("add worker")
	buf, err := getBody(r)
	if err != nil {
		sendResponse(w, http.StatusBadRequest, nil)
		return err
	}
	nij := &WorkerInfoRequest{}
	if err := json.Unmarshal(buf, nij); err != nil {
		sendResponse(w, http.StatusBadRequest, nil)
		return err
	}
	if err := nij.Validate(); err != nil {
		sendResponse(w, http.StatusBadRequest, nil)
		return err
	}

	wk, err := nij.ToWorker()
	if err != nil {
		sendResponse(w, http.StatusBadRequest, nil)
		return err
	}
	id, err := s.master.AddWorker(ctx, wk)
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, []byte(err.Error()))
		return err
	}
	//if err := s.master.NotifyWorkflowsToAllWorkers(ctx); err != nil {
	//	sendResponse(w, http.StatusInternalServerError, []byte(err.Error()))
	//	return err
	//}
	ns, err := s.master.Workers(ctx)
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, nil)
		return err
	}
	for _, n := range ns {
		log.Printf("%#v\n", n)
	}
	res := AddWorkerResponse{ID: id}
	resBody, err := json.Marshal(res)
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, nil)
	}
	sendResponse(w, http.StatusCreated, resBody)
	return nil
}

func (s *Server) updateWorkerResource(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	workerID := chi.URLParam(r, "workerID")
	buf, err := getBody(r)
	if err != nil {
		sendResponse(w, http.StatusBadRequest, nil)
		return err
	}
	wkr := &WorkerInfoRequest{}
	if err := json.Unmarshal(buf, wkr); err != nil {
		sendResponse(w, http.StatusBadRequest, nil)
		return err
	}

	wk, err := wkr.ToWorker()
	if err != nil {
		sendResponse(w, http.StatusBadRequest, nil)
		return err
	}
	wk.ID = workerID
	if err := s.master.UpdateWorkerResource(ctx, wk); err != nil {
		sendResponse(w, http.StatusInternalServerError, nil)
		return err
	}
	sendResponse(w, http.StatusNoContent, nil)
	return nil
}

func (s *Server) check(w http.ResponseWriter, r *http.Request) error {
	sendResponse(w, http.StatusOK, nil)
	return nil
}

func (s *Server) listWorkflows(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	ws, err := s.master.ListWorkflows(ctx)
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, []byte(err.Error()))
		return err
	}
	respBody, err := json.Marshal(ws)
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, []byte(err.Error()))
		return err
	}
	sendResponse(w, http.StatusOK, respBody)
	return nil
}

func (s *Server) addWorkflow(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var wf domain.Workflow
	if err := json.NewDecoder(r.Body).Decode(&wf); err != nil {
		return err
	}
	id, err := s.master.AddWorkflow(ctx, &wf)
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, nil)
		return err
	}
	wf.ID = id
	respBody, err := json.Marshal(wf)
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, []byte(err.Error()))
		return err
	}
	sendResponse(w, http.StatusCreated, respBody)
	return nil
}

// JobIDの整合性取れないので無理
//func (s *Server) updateWorkflow(w http.ResponseWriter, r *http.Request) error {
//	ctx := r.Context()
//	var wf domain.Workflow
//	if err := json.NewDecoder(r.Body).Decode(&wf); err != nil {
//		sendResponse(w, http.StatusInternalServerError, nil)
//		return err
//	}
//	if err := s.master.UpdateWorkflow(ctx, wf.JobName, &wf); err != nil {
//		sendResponse(w, http.StatusInternalServerError, nil)
//		return err
//	}
//	respBody, err := json.Marshal(wf)
//	if err != nil {
//		sendResponse(w, http.StatusInternalServerError, []byte(err.Error()))
//		return err
//	}
//	sendResponse(w, http.StatusCreated, respBody)
//	return nil
//}

func (s *Server) nextJobWorker(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	workflowID := chi.URLParam(r, "workflowID")
	stepID := chi.URLParam(r, "stepID")
	preWorkerID := r.URL.Query().Get("previousJobWorkerID")
	wk, err := s.master.DetermineNextJobWorker(ctx, &master.OptionsDetermineNextJobWorker{
		WorkflowID:          workflowID,
		StepID:              stepID,
		PreviousJobWorkerID: preWorkerID,
	})
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, []byte(err.Error()))
		log.Println(err)
		return err
	}
	log.Printf("determined. workerID=%s, addr=%s", wk.ID, wk.URL.String())
	respBody, err := json.Marshal(NewResponseWorker(wk))
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, []byte(err.Error()))
		return err
	}
	sendResponse(w, http.StatusOK, respBody)
	return nil
}

func (s *Server) fail(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	workflowID := chi.URLParam(r, "workflowID")
	stepID := chi.URLParam(r, "stepID")
	return s.master.FailJob(ctx, workflowID, stepID)
}

func (s *Server) getStatusOfWorkflow(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	workflowName := chi.URLParam(r, "workflowName")
	statuses, err := s.master.ListWorkflowStatusesByWorkflowName(ctx, workflowName)
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, nil)
		return err
	}
	respBody, err := json.Marshal(statuses)
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, nil)
		return err
	}
	sendResponse(w, http.StatusOK, respBody)
	return nil
}
