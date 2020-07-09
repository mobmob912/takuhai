package worker

import (
	"errors"
	"net/url"
	"time"

	"github.com/mobmob912/takuhai/domain"
)

type Worker struct {
	ID     string           `json:"id"`
	Name   string           `json:"name"`
	Type   domain.ImageType `json:"type"`
	Arch   domain.ArchType  `json:"arch"`
	Labels []string         `json:"labels"`
	URL    *url.URL         `json:"url"`

	Errors []error `json:"-"`

	Place domain.Place `json:"place"`

	CPUUsagePercent float64       `json:"cpu_usage_percent"`
	CPUClockMhz     float64       `json:"cpu_clock_mhz"`
	AvailableMemory uint64        `json:"available_memory"`
	Latency         time.Duration `json:"latency"`
}

func (n *Worker) Validate(ns []*Worker) error {
	if n.Name == "" {
		return errors.New("name is required")
	}
	for i := range ns {
		if n.URL == ns[i].URL {
			return errors.New("duplicate worker URL")
		}
	}
	return nil
}

type Info struct {
	Name string
	URL  *url.URL

	CPUUsagePercent float64
	CPUClockMhz     float64
	AvailableMemory uint64
}

func New(info *Info) *Worker {
	n := &Worker{
		Name:            info.Name,
		URL:             info.URL,
		CPUUsagePercent: info.CPUUsagePercent,
		CPUClockMhz:     info.CPUClockMhz,
		AvailableMemory: info.AvailableMemory,
	}
	return n
}
