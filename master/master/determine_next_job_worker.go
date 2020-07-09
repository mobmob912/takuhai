package master

import (
	"context"
	"errors"
	"reflect"

	"github.com/mobmob912/takuhai/domain"

	"github.com/mobmob912/takuhai/master/worker"
)

var (
	ErrMatchedWorkerNotFound = errors.New("matched worker not found")
)

func (m *Master) DetermineNextJobWorker(ctx context.Context, opts *OptionsDetermineNextJobWorker) (*worker.Worker, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}
	step, err := m.workflowRepository.GetStep(ctx, opts.WorkflowID, opts.StepID)
	if err != nil {
		return nil, err
	}

	wks, err := m.workerRepository.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	wks, err = listWorkersFromTypeAndArch(ctx, wks, step.Job)
	if err != nil {
		return nil, err
	}

	if len(step.Labels) != 0 {
		labeledWks := make([]*worker.Worker, 0)
		for _, w := range wks {
			if reflect.DeepEqual(w.Labels, step.Labels) {
				labeledWks = append(labeledWks, w)
			}
		}
		wks = labeledWks
	}
	return determineNextWorkerFromWorkers(ctx, wks, step, opts)
}

func listWorkersFromTypeAndArch(ctx context.Context, wks []*worker.Worker, j *domain.Job) ([]*worker.Worker, error) {
	rwks := make([]*worker.Worker, 0)
	for _, w := range wks {
		for _, is := range j.Images {
			if is.Type.Satisfy(w.Type) && is.Arch.Satisfy(w.Arch) {
				rwks = append(rwks, w)
			}
		}
	}
	return rwks, nil
}

func determineNextWorkerFromWorkers(ctx context.Context, wks []*worker.Worker, step *domain.Step, opts *OptionsDetermineNextJobWorker) (*worker.Worker, error) {
	// TODO なかった時
	if step.Place == domain.PlaceEdge && opts.PreviousJobWorkerID != "" {
		for _, w := range wks {
			if w.ID == opts.PreviousJobWorkerID {
				return w, nil
			}
		}
	}

	if step.Place == domain.PlaceCloud {
		return determineNextWorkerByCloudWorkers(ctx, wks, opts)
	}
	return determineNextWorkerByAllWorkers(ctx, wks, opts)
}

func determineNextWorkerByCloudWorkers(ctx context.Context, wks []*worker.Worker, opts *OptionsDetermineNextJobWorker) (*worker.Worker, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}
	cwks := make([]*worker.Worker, 0)

	for _, w := range wks {
		if w.Place == domain.PlaceCloud {
			cwks = append(cwks, w)
		}
	}
	if len(cwks) == 0 {
		return nil, ErrMatchedWorkerNotFound
	}

	var determinedWorker *worker.Worker
	var maxRAM uint64
	for _, wk := range cwks {
		if wk.AvailableMemory > maxRAM {
			determinedWorker = wk
			maxRAM = wk.AvailableMemory
		}
	}
	if determinedWorker == nil {
		// TODO error type
		return nil, ErrMatchedWorkerNotFound
	}
	return determinedWorker, nil
}

func determineNextWorkerByAllWorkers(ctx context.Context, wks []*worker.Worker, opts *OptionsDetermineNextJobWorker) (*worker.Worker, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	var determinedWorker *worker.Worker
	var maxRAM uint64
	for _, wk := range wks {
		if wk.AvailableMemory > maxRAM {
			determinedWorker = wk
			maxRAM = wk.AvailableMemory
		}
	}
	if determinedWorker == nil {
		// TODO error type
		return nil, ErrMatchedWorkerNotFound
	}
	return determinedWorker, nil
}
