package store

import (
	"context"
	"errors"
	"sync"

	"github.com/rs/xid"

	"github.com/mobmob912/takuhai/worker_manager/job"
)

var (
	ErrNotFound = errors.New("not found")
)

// そのノードで配置されているアプリケーション
// アプリケーションにはpending, ready, workingの３つの状態がある。
// pending - デプロイ中のjob
// ready - デプロイ完了し、待機中
// working - 稼働中
type Job interface {
	ListAll(ctx context.Context) ([]job.Job, error)
	GetFromReady(ctx context.Context, stepID string) (job.Job, error)
	GetFromPending(ctx context.Context, stepID string) (job.Job, error)
	SetPending(ctx context.Context, stepID string, job job.Job) error
	SetReadyFromPending(ctx context.Context, stepID string) error
	SetRunningFromReady(ctx context.Context, stepID string) (jobID string, err error)
	DeleteRunningJob(ctx context.Context, jobID string) error
	IsReady(ctx context.Context, stepID string) (bool, error)
	IsPending(ctx context.Context, stepID string) (bool, error)
	IsRunning(ctx context.Context, stepID string) (bool, error)
}

type jobStore struct {
	mutex       *sync.Mutex
	allJobs     []job.Job
	runningJobs map[string]job.Job
	readyJobs   map[string]job.Job

	// k=jobName
	pendingJobs map[string]job.Job
}

func NewJob() Job {
	return &jobStore{
		mutex:       new(sync.Mutex),
		allJobs:     make([]job.Job, 0),
		runningJobs: make(map[string]job.Job),
		readyJobs:   make(map[string]job.Job),
		pendingJobs: make(map[string]job.Job),
	}
}

func (a *jobStore) ListAll(ctx context.Context) ([]job.Job, error) {
	return a.allJobs, nil
}

func (a *jobStore) GetFromReady(ctx context.Context, stepID string) (job.Job, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	j, ok := a.readyJobs[stepID]
	if !ok {
		return nil, ErrNotFound
	}
	return j, nil
}

func (a *jobStore) GetFromPending(ctx context.Context, stepID string) (job.Job, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	j, ok := a.pendingJobs[stepID]
	if !ok {
		return nil, ErrNotFound
	}
	return j, nil
}

func (a *jobStore) SetPending(ctx context.Context, stepID string, job job.Job) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.allJobs = append(a.allJobs, job)
	a.pendingJobs[stepID] = job
	return nil
}

func (a *jobStore) SetReadyFromPending(ctx context.Context, stepID string) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	j, ok := a.pendingJobs[stepID]
	if !ok {
		return ErrNotFound
	}
	a.readyJobs[stepID] = j
	delete(a.pendingJobs, stepID)
	return nil
}

func (a *jobStore) SetRunningFromReady(ctx context.Context, stepID string) (string, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	j, ok := a.readyJobs[stepID]
	if !ok {
		return "", ErrNotFound
	}
	jobID := xid.New().String()
	a.runningJobs[jobID] = j
	return jobID, nil
}

func (a *jobStore) IsPending(ctx context.Context, stepID string) (bool, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	_, ok := a.pendingJobs[stepID]
	return ok, nil
}
func (a *jobStore) IsReady(ctx context.Context, stepID string) (bool, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	_, ok := a.readyJobs[stepID]
	return ok, nil
}
func (a *jobStore) IsRunning(ctx context.Context, stepID string) (bool, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	for _, rj := range a.runningJobs {
		if rj.StepID() == stepID {
			return true, nil
		}
	}
	return false, nil
}

func (a *jobStore) DeleteRunningJob(ctx context.Context, jobID string) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	delete(a.runningJobs, jobID)
	return nil
}
