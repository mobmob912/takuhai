package domain

import (
	"errors"
	"strings"
)

type Workflow struct {
	ID      string   `json:"id"`
	Name    string   `yaml:"name" json:"name"`
	Trigger *Trigger `yaml:"trigger" json:"trigger"`
	Jobs    []*Job   `yaml:"jobs" json:"jobs"`
	Steps   []*Step  `yaml:"steps" json:"steps"`
}

func (w *Workflow) SetStepsJob() error {
	for fi, f := range w.Steps {
		matched := false
		for _, j := range w.Jobs {
			if f.JobName == j.Name {
				w.Steps[fi].Job = j
				matched = true
				break
			}
		}
		if !matched {
			return errors.New("could not find matched job name " + f.JobName)
		}
	}
	return nil
}

func (w *Workflow) NextStepsByCurrentStepID(currentStepID string) []*Step {
	ss := make([]*Step, 0)
	for _, s := range w.Steps {
		if s.AfterByID == currentStepID {
			ss = append(ss, s)
		}
	}
	return ss
}

func (w *Workflow) GetFailureStepByFailedStepID(failedStepID string) *Step {
	for _, s := range w.Steps {
		if s.ID != failedStepID {
			continue
		}
		return s.Failure
	}
	return nil
}

type TriggerType string

const (
	TriggerTypeHTTP TriggerType = "http"
	TriggerTypeCron TriggerType = "cron"
)

type Trigger struct {
	Type   TriggerType `yaml:"type" json:"type"`
	Path   string      `yaml:"path" json:"path"`
	Output string      `yaml:"output" json:"output"`
}

type ImageType string

func (t ImageType) Satisfy(at ImageType) bool {
	ts := strings.Split(string(t), ",")
	ats := strings.Split(string(at), ",")
	for _, tt := range ts {
		for _, att := range ats {
			if tt == att {
				return true
			}
		}
	}
	return false
}

const (
	ImageTypeDocker ImageType = "docker"
	ImageTypeShell  ImageType = "shell"
)

type Place string

const (
	PlaceEdge  Place = "edge"
	PlaceCloud Place = "cloud"
	PlaceAny   Place = "any"
)

type ArchType string

func (t ArchType) Satisfy(at ArchType) bool {
	ts := strings.Split(string(t), ",")
	ats := strings.Split(string(at), ",")
	for _, tt := range ts {
		for _, att := range ats {
			if tt == att {
				return true
			}
		}
	}
	return false
}

const (
	ArchTypeAMD ArchType = "amd"
	ArchTypeARM ArchType = "arm"
)

type Image struct {
	Type  ImageType `yaml:"type" json:"type"`
	Arch  ArchType  `yaml:"arch" json:"arch"`
	Image string    `yaml:"image" json:"image"`
}

type Job struct {
	Name     string   `yaml:"name" json:"name"`
	Images   []*Image `yaml:"images" json:"images"`
	Function string   `yaml:"function,omitempty" json:"function"`
	Input    string   `yaml:"input" json:"input"`
	Limits   struct {
		Memory string `yaml:"memory" json:"memory"`
		CPU    string `yaml:"cpu" json:"cpu"`
	} `yaml:"limits,omitempty" json:"limits"`
	Output string `yaml:"output,omitempty" json:"output"`
}

type Step struct {
	ID        string   `yaml:"-" json:"id"`
	Name      string   `yaml:"name" json:"name"`
	JobName   string   `yaml:"jobName" json:"job_name"`
	Place     Place    `yaml:"place" json:"place"`
	Labels    []string `yaml:"labels" json:"labels"`
	After     string   `yaml:"after" json:"after"`
	AfterByID string   `yaml:"-" json:"after_by_id"`
	Job       *Job     `yaml:"-" json:"job"`
	Failure   *Step    `yaml:"failure" json:"failure"`
}
