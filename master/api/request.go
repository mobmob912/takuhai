package api

import (
	"errors"
	"net/url"

	"github.com/mobmob912/takuhai/domain"

	"github.com/mobmob912/takuhai/master/worker"
)

type WorkerInfoRequest struct {
	Name   string           `json:"name"`
	URL    string           `json:"url"`
	Arch   domain.ArchType  `json:"arch"`
	Type   domain.ImageType `json:"type"`
	Place  domain.Place     `json:"place"`
	Labels []string         `json:"labels"`

	CPUUsagePercent float64 `json:"cpu_usage_percent"`
	CPUClockMhz     float64 `json:"cpu_clock_mhz"`
	AvailableMemory uint64  `json:"available_memory"`
}

func (n *WorkerInfoRequest) Validate() error {
	if n.Name == "" || n.URL == "" || n.Arch == "" || n.Type == "" || n.Place == "" {
		return errors.New("name, url, arch, type and place are must not empty")
	}
	return nil
}

// Deprecated:
func (n *WorkerInfoRequest) ToWorkerInfo() (*worker.Info, error) {
	u, err := url.Parse(n.URL)
	if err != nil {
		return nil, err
	}
	return &worker.Info{
		Name:            n.Name,
		URL:             u,
		CPUUsagePercent: n.CPUUsagePercent,
		CPUClockMhz:     n.CPUClockMhz,
		AvailableMemory: n.AvailableMemory,
	}, nil
}

func (n *WorkerInfoRequest) ToWorker() (*worker.Worker, error) {
	u, err := url.Parse(n.URL)
	if err != nil {
		return nil, err
	}
	return &worker.Worker{
		Name:            n.Name,
		Type:            n.Type,
		Arch:            n.Arch,
		Place:           n.Place,
		Labels:          n.Labels,
		URL:             u,
		CPUUsagePercent: n.CPUUsagePercent,
		CPUClockMhz:     n.CPUClockMhz,
		AvailableMemory: n.AvailableMemory,
	}, nil
}

type WorkerInfoResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

func WorkerInfoResponseFromWorker(no *worker.Worker) *WorkerInfoResponse {
	return &WorkerInfoResponse{
		ID:   no.ID,
		Name: no.Name,
		URL:  no.URL.String(),
	}
}
