package master

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mobmob912/takuhai/domain"

	"golang.org/x/sync/errgroup"

	"github.com/mobmob912/takuhai/master/worker"
)

// 各ジョブがどこにデプロイされているか
// で、デプロイされたジョブがそのワーカーで実行されているかどうか
type WorkflowStatuses struct {
	Workflow *domain.Workflow    `json:"workflow"`
	Statuses []*StepWorkerStatus `json:"statuses"`
}

type StepWorkerStatus struct {
	Step           *domain.Step      `json:"step"`
	DeployedWorker []*DeployedWorker `json:"deployed_worker"`
}

type DeployedWorker struct {
	Worker    *worker.Worker `json:"worker"`
	IsPending bool           `json:"is_pending"`
	IsRunning bool           `json:"is_running"`
}

type ResponseStepStatus struct {
}

type WorkerStepStatus struct {
	Worker     *worker.Worker `json:"-"`
	Step       *domain.Step   `json:"step"`
	IsPending  bool           `json:"is_pending"`
	IsDeployed bool           `json:"is_deployed"`
	IsRunning  bool           `json:"is_running"`
}

// 全ワーカーに対して、デプロイされているジョブ、それを実行中かを問い合わせる

func (m *Master) ListWorkflowStatusesByWorkflowName(ctx context.Context, workflowName string) (*WorkflowStatuses, error) {
	wf, err := m.workflowRepository.GetByName(ctx, workflowName)
	if err != nil {
		return nil, err
	}
	wks, err := m.workerRepository.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	statuses := make([]*WorkerStepStatus, 0)
	eg := errgroup.Group{}
	for _, wk := range wks {
		wk := wk
		eg.Go(func() error {
			s, err := m.GetWorkerStepStatus(ctx, wk, wf.ID)
			if err != nil {
				return err
			}
			statuses = append(statuses, s...)
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	rwss := &WorkflowStatuses{
		Workflow: wf,
	}
	for _, step := range wf.Steps {
		sws := &StepWorkerStatus{Step: step}
		for _, status := range statuses {
			if step.ID != status.Step.ID {
				continue
			}
			sws.DeployedWorker = append(sws.DeployedWorker, &DeployedWorker{
				Worker:    status.Worker,
				IsPending: status.IsPending,
				IsRunning: status.IsRunning,
			})
		}
		rwss.Statuses = append(rwss.Statuses, sws)
	}
	return rwss, nil
}

func (m *Master) GetWorkerStepStatus(ctx context.Context, wk *worker.Worker, workflowID string) ([]*WorkerStepStatus, error) {
	c := http.DefaultClient
	u := *wk.URL
	u.Path = fmt.Sprintf("workflows/%s/steps/status", workflowID)
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	var wjs []*WorkerStepStatus
	if err := json.NewDecoder(resp.Body).Decode(&wjs); err != nil {
		return nil, err
	}
	for i := range wjs {
		wjs[i].Worker = wk
	}
	return wjs, nil
}
