package api

import (
	"github.com/mobmob912/takuhai/master/worker"
)

type AddWorkerResponse struct {
	ID string `json:"id"`
}

type ResponseWorker struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

func NewResponseWorker(w *worker.Worker) *ResponseWorker {
	return &ResponseWorker{
		ID:   w.ID,
		Name: w.Name,
		URL:  w.URL.String(),
	}
}
