package master

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/mobmob912/takuhai/domain"
	"golang.org/x/sync/errgroup"

	"github.com/mobmob912/takuhai/master/master/repository"

	"github.com/mobmob912/takuhai/master/worker"

	"github.com/rs/xid"
)

type Master struct {
	workerRepository   repository.Worker
	workflowRepository repository.Workflow
	uidGenerator       repository.UID
}

func NewMaster(nr repository.Worker, wr repository.Workflow, uid repository.UID) *Master {
	return &Master{
		workerRepository:   nr,
		workflowRepository: wr,
		uidGenerator:       uid,
	}
}

func (m *Master) Init(ctx context.Context) error {
	if err := m.HealthCheckAllWorkers(context.Background()); err != nil {
		return err
	}
	ws, err := m.workerRepository.ListAll(ctx)
	if err != nil {
		return err
	}
	for _, w := range ws {
		go m.PeriodicWorkerHealthCheck(ctx, w)
	}
	return nil
}

func (m *Master) Workers(ctx context.Context) ([]*worker.Worker, error) {
	return m.workerRepository.ListAll(ctx)
}

func (m *Master) GetWorkerByID(ctx context.Context, id string) (*worker.Worker, error) {
	return m.workerRepository.Get(ctx, id)
}

func (m *Master) AddWorker(ctx context.Context, n *worker.Worker) (string, error) {
	ns, err := m.workerRepository.ListAll(ctx)
	if err != nil {
		return "", err
	}
	if err := n.Validate(ns); err != nil {
		return "", err
	}
	if exist := CheckWorkerExistByName(ns, n.Name); exist {
		return "", errors.New(fmt.Sprintf("Worker name %s is exist", n.Name))
	}
	//if err := healthCheck(*n.URL); err != nil {
	//	return "", errors.New("health check error. may be worker manager is not working or addr is invalid")
	//}
	id := xid.New().String()
	n.ID = id
	go m.PeriodicWorkerHealthCheck(context.Background(), n)
	return id, m.workerRepository.Set(ctx, id, n)
}

func CheckWorkerExistByName(ws []*worker.Worker, name string) bool {
	for _, w := range ws {
		if w.Name == name {
			return true
		}
	}
	return false
}

func healthCheck(u url.URL) error {
	c := http.DefaultClient
	c.Timeout = 3 * time.Second
	u.Path = "/check"
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return err
	}
	_, err = c.Do(req)
	return err
}

// 基本goroutineで動かす
func (m *Master) PeriodicWorkerHealthCheck(ctx context.Context, w *worker.Worker) {
	for {
		time.Sleep(1 * time.Second)
		if err := healthCheck(*w.URL); err != nil {
			log.Printf("health check failed. worker name: %s. it will delete. msg: %s", w.Name, err.Error())
			if err := m.DeleteWorker(ctx, w.ID); err != nil {
				log.Printf("worker delete failed. id: %s. msg: ", w.ID, err.Error())
				continue
			}
			break
		}
	}
}

func (m *Master) HealthCheckAllWorkers(ctx context.Context) error {
	ws, err := m.workerRepository.ListAll(ctx)
	if err != nil {
		return err
	}
	eg := &errgroup.Group{}
	for _, w := range ws {
		w := w
		eg.Go(func() error {
			if err := healthCheck(*w.URL); err != nil {
				if err := m.DeleteWorker(ctx, w.ID); err != nil {
					return err
				}
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}

func (m *Master) UpdateWorkerResource(ctx context.Context, w *worker.Worker) error {
	pw, err := m.workerRepository.Get(ctx, w.ID)
	if err != nil {
		return err
	}
	pw.CPUUsagePercent = w.CPUUsagePercent
	pw.CPUClockMhz = w.CPUClockMhz
	pw.AvailableMemory = w.AvailableMemory
	if err := m.workerRepository.Update(ctx, pw.ID, pw); err != nil {
		return err
	}
	return nil
}

func (m *Master) DeleteWorker(ctx context.Context, id string) error {
	return m.workerRepository.Delete(ctx, id)
}

var (
	ErrAlreadyRegistered = errors.New("workflow already exist")
)

func (m *Master) ListWorkflows(ctx context.Context) ([]*domain.Workflow, error) {
	return m.workflowRepository.ListAll(ctx)
}

func (m *Master) AddWorkflow(ctx context.Context, wf *domain.Workflow) (string, error) {
	exist, err := m.workflowRepository.CheckExistByByName(ctx, wf.Name)
	if err != nil {
		return "", err
	}
	if exist {
		return "", ErrAlreadyRegistered
	}

	for i := range wf.Steps {
		wf.Steps[i].ID = m.uidGenerator.New()
	}

	// set after by id
	for i, w := range wf.Steps {
		if w.After == "" {
			continue
		}
		found := false
		for _, aw := range wf.Steps {
			if aw.Name != w.After {
				continue
			}
			wf.Steps[i].AfterByID = aw.ID
			found = true
		}
		if !found {
			return "", errors.New("after step name is not found. " + w.After)
		}
	}

	if err := wf.SetStepsJob(); err != nil {
		return "", err
	}

	id := m.uidGenerator.New()
	if err := m.workflowRepository.Set(ctx, id, wf); err != nil {
		return "", err
	}
	// 各Workerが定期的にworkflow更新を問い合わせる形も考えたが、
	// workflow更新頻度の少なさを考えるとそれじゃトラフィックを圧迫しそうなので
	// Masterから通知する形にする
	if err := m.NotifyWorkflowsToAllWorkers(ctx); err != nil {
		return "", err
	}
	return id, nil
}

func (m *Master) ApplyWorkflow(ctx context.Context, wf *domain.Workflow) error {
	switch wf.Trigger.Type {
	// TODO cron type実装
	case domain.TriggerTypeCron:
	case domain.TriggerTypeHTTP:
	}
	return nil
}

func (m *Master) NotifyWorkflowsToAllWorkers(ctx context.Context) error {
	wfs, err := m.workflowRepository.ListAll(ctx)
	if err != nil {
		return err
	}
	ws, err := m.workerRepository.ListAll(ctx)
	if err != nil {
		return err
	}
	reqBody, err := json.Marshal(wfs)
	if err != nil {
		return err
	}
	eg := errgroup.Group{}
	for _, w := range ws {
		w := w
		eg.Go(func() error {
			w := w
			c := http.DefaultClient
			req, err := http.NewRequest("PUT", w.URL.String()+"/workflows", bytes.NewReader(reqBody))
			if err != nil {
				return err
			}
			res, err := c.Do(req)
			if err != nil {
				return err
			}
			resBody, err := ioutil.ReadAll(res.Body)
			if err != nil {
				return err
			}
			log.Println(w.URL.String())
			log.Println(string(resBody))
			return nil
		})
	}
	// TODO 通知エラーした場合の制御も考える
	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}

func (m *Master) FailJob(ctx context.Context, workflowID, stepID string) error {
	return nil
}

// JobIDの整合性取れないので無理
//func (m *Master) UpdateWorkflow(ctx context.Context, name string, wf *domain.Workflow) error {
//	pwf, err := m.workflowRepository.GetByName(ctx, name)
//	if err != nil {
//		return err
//	}
//	wf.StepID = pwf.StepID
//	if err := m.workflowRepository.Set(ctx, wf.StepID, wf); err != nil {
//		return err
//	}
//	if err := m.NotifyWorkflowsToAllWorkers(ctx); err != nil {
//		return err
//	}
//	return nil
//}

type OptionsDetermineNextJobWorker struct {
	WorkflowID          string
	StepID              string
	PreviousJobWorkerID string
}

func (opt *OptionsDetermineNextJobWorker) Validate() error {
	if opt.WorkflowID == "" {
		return errors.New("workflow id is missing")
	}
	if opt.StepID == "" {
		return errors.New("step id is missing")
	}
	return nil
}
