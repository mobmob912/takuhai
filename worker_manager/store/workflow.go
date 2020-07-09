package store

import (
	"context"
	"sync"

	"github.com/mobmob912/takuhai/domain"
)

// workflowの保存（オンメモリキャッシュ）
type Workflow interface {
	Get(ctx context.Context, id string) (*domain.Workflow, error)
	GetJob(ctx context.Context, workflowID, stepID string) (*domain.Job, error)
	GetByTriggerHTTPPath(ctx context.Context, triggerPath string) (*domain.Workflow, error)
	Set(ctx context.Context, id string, workflow *domain.Workflow) error
	UpdateAll(ctx context.Context, ws []*domain.Workflow) error
}

type workflow struct {
	mutex     *sync.Mutex
	workflows map[string]*domain.Workflow
}

func (s *workflow) Get(ctx context.Context, id string) (*domain.Workflow, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.workflows[id], nil
}

func (s *workflow) GetJob(ctx context.Context, workflowID, stepID string) (*domain.Job, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	wk := s.workflows[workflowID]
	if wk == nil {
		return nil, ErrNotFound
	}
	for _, f := range wk.Steps {
		if f.ID == stepID {
			return f.Job, nil
		}
	}
	return nil, ErrNotFound
}

func (s *workflow) GetByTriggerHTTPPath(ctx context.Context, triggerPath string) (*domain.Workflow, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, wf := range s.workflows {
		if wf.Trigger.Type == domain.TriggerTypeHTTP && wf.Trigger.Path == triggerPath {
			return wf, nil
		}
	}
	return nil, ErrNotFound
}

func (s *workflow) Set(ctx context.Context, id string, v *domain.Workflow) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.workflows[id] = v
	return nil
}

func (s *workflow) UpdateAll(ctx context.Context, ws []*domain.Workflow) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.workflows = make(map[string]*domain.Workflow, len(ws))
	for _, w := range ws {
		s.workflows[w.ID] = w
	}
	return nil
}

func NewWorkflow() Workflow {
	return &workflow{
		mutex:     new(sync.Mutex),
		workflows: make(map[string]*domain.Workflow),
	}
}

// Masterとの通信
