package repository

import (
	"context"
	"errors"

	"github.com/mobmob912/takuhai/domain"
	"github.com/mobmob912/takuhai/master/worker"
)

var (
	ErrNotFound = errors.New("record not found")
)

type Workflow interface {
	Get(ctx context.Context, id string) (*domain.Workflow, error)
	GetByName(ctx context.Context, name string) (*domain.Workflow, error)
	GetStep(ctx context.Context, workflowID, stepID string) (*domain.Step, error)
	ListAll(ctx context.Context) ([]*domain.Workflow, error)
	CheckExistByByName(ctx context.Context, name string) (bool, error)
	Set(ctx context.Context, id string, workflow *domain.Workflow) error
}

type Worker interface {
	Get(ctx context.Context, id string) (*worker.Worker, error)
	ListAll(ctx context.Context) ([]*worker.Worker, error)
	ListClouds(ctx context.Context) ([]*worker.Worker, error)
	Set(ctx context.Context, id string, worker *worker.Worker) error
	Update(ctx context.Context, id string, worker *worker.Worker) error
	Delete(ctx context.Context, id string) error
}

// FlowAppが稼働しているWorkerを管理
type Application interface {
	FindDeployedWorker(ctx context.Context, flowID string) (*worker.Worker, error)
}

type UID interface {
	New() string
}
