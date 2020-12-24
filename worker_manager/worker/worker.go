package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"io/ioutil"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/mobmob912/takuhai/worker_manager/job/shell"

	"github.com/mobmob912/takuhai/domain"

	"github.com/docker/docker/client"

	"github.com/mobmob912/takuhai/worker_manager/store"

	"github.com/mobmob912/takuhai/worker_manager/job"
	"github.com/mobmob912/takuhai/worker_manager/job/container"

	"github.com/rs/xid"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/mobmob912/takuhai/master/api"
)
var nowstep *domain.Step
type MasterInfo struct {
	URL *url.URL
}
type WorkerResponse struct {
	Time time.Time
}
type ToWorker struct {
	Name string
	Time time.Duration

}
type WorkerResponse2 struct {
	FromID string
	ToWorkers []ToWorker
}


type ToNode  struct {
	ID   string
	Name string
	URL  *url.URL 
}

// TODO: interfaceにしてもイイかも
type Worker struct {
	ID            string
	Type          domain.ImageType
	Arch          domain.ArchType
	Place         domain.Place
	Labels        []string
	OtherWorkers  []ToNode
	MasterInfo    *MasterInfo
	LocalIPAddr   *net.IP
	Errors        []error
	JobStore      store.Job
	WorkflowStore store.Workflow
}

type OptionsNew struct {
	Type          domain.ImageType
	Arch          domain.ArchType
	Place         domain.Place
	Labels        []string
	MasterInfo    *MasterInfo
	IPAddr        *net.IP
	JobStore      store.Job
	WorkflowStore store.Workflow
}
type Content struct {
	Body           []byte        
	Runtime        time.Duration
	RAM         float64
	CPU         float64
	
}

func New(opts *OptionsNew) *Worker {
	return &Worker{
		ID:            "",
		Type:          opts.Type,
		Arch:          opts.Arch,
		Place:         opts.Place,
		Labels:        opts.Labels,
		OtherWorkers:   nil,
		MasterInfo:    opts.MasterInfo,
		LocalIPAddr:   opts.IPAddr,
		Errors:        nil,
		JobStore:      opts.JobStore,
		WorkflowStore: opts.WorkflowStore,
	}
}

type ResourceInfo struct {
	CPUUsagePercent float64
	CPUClockMhz     float64
	AvailableMemory uint64
}

func (w *Worker) AddError(err error) {
	if err == nil {
		return
	}
	if w.Errors == nil {
		w.Errors = make([]error, 0, 4)
	}
	log.Println(err)
	w.Errors = append(w.Errors, err)
}

func (w *Worker) PeriodicNotifyResourceInformationToMaster(ctx context.Context) {
	for {
		ri, err := GetResourceInfo(ctx)
		if err != nil {
			w.AddError(err)
		}
		reqStruct := &api.WorkerInfoRequest{
			CPUUsagePercent: ri.CPUUsagePercent,
			CPUClockMhz:     ri.CPUClockMhz,
			AvailableMemory: ri.AvailableMemory,
		}
		reqBody, err := json.Marshal(reqStruct)
		if err != nil {
			w.AddError(err)
		}
		client := http.DefaultClient
		client.Timeout = 3 * time.Second
		req, err := http.NewRequest("PUT", fmt.Sprintf("%s/workers/%s", w.MasterInfo.URL.String(), w.ID), bytes.NewReader(reqBody))
		if err != nil {
			w.AddError(err)
		}
		if _, err := client.Do(req); err != nil {
			w.AddError(errors.New(fmt.Sprintf(err.Error())))
		}
	}
}

func (w *Worker) PeriodicGetWorkflows(ctx context.Context) {
	for {
		c := http.DefaultClient
		c.Timeout = 3 * time.Second
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/workflows", w.MasterInfo.URL.String()), nil)
		if err != nil {
			w.AddError(err)
			continue
		}
		resp, err := c.Do(req)
		if err != nil {
			w.AddError(errors.New(fmt.Sprintf(err.Error())))
			continue
		}
		ws := make([]*domain.Workflow, 0)
		if err := json.NewDecoder(resp.Body).Decode(&ws); err != nil {
			w.AddError(err)
			continue
		}
		if err := w.WorkflowStore.UpdateAll(ctx, ws); err != nil {
			w.AddError(err)
			continue
		}
	}
}

func (w *Worker) PeriodicCheckErrors(ctx context.Context) {
	for {
		time.Sleep(1 * time.Second)
		if len(w.Errors) == 0 {
			continue
		}
		for _, err := range w.Errors {
			log.Println(err)
		}
		w.Errors = nil
	}
}

func GetResourceInfo(ctx context.Context) (*ResourceInfo, error) {
	info := &ResourceInfo{}
	ps, err := cpu.Percent(1*time.Second, false)
	if err != nil {
		return nil, err
	}
	info.CPUUsagePercent = ps[0]
	cpuInfos, err := cpu.Info()
	if err != nil {
		return nil, err
	}
	info.CPUClockMhz = cpuInfos[0].Mhz
	m, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}
	info.AvailableMemory = m.Available
	return info, nil
}

func (w *Worker) RegisterDeployedJob(ctx context.Context, stepID string) (string, error) {
	id := xid.New().String()
	if err := w.JobStore.SetReadyFromPending(ctx, stepID); err != nil {
		return "", err
	}
	return id, nil
}

var (
	ErrNotFoundSatisfiedImage = errors.New("not found satisfied image")
)

// 時間かかるのでgoroutineで呼ぶべき
// TODO: これエラーすると結構致命的なので、イイ感じにmasterへ通知する仕組み作る
func (w *Worker) DeployJob(ctx context.Context, workflowID, stepID string) error {

	jobInfo, err := w.WorkflowStore.GetJob(ctx, workflowID, stepID)
	if err != nil {
		return err
	}

	for _, img := range jobInfo.Images {
		if img.Type == w.Type && img.Arch == w.Arch {
			return w.deployJobByType(ctx, &optionsDeployJobByType{
				imageType:  img.Type,
				stepID:     stepID,
				name:       jobInfo.Name,
				image:      img.Image,
				workflowID: workflowID,
			})
		}
	}
	return ErrNotFoundSatisfiedImage
}

type optionsDeployJobByType struct {
	imageType  domain.ImageType
	stepID     string
	name       string
	image      string
	workflowID string
}

func (w *Worker) deployJobByType(ctx context.Context, opts *optionsDeployJobByType) error {
	var j job.Job
	switch opts.imageType {
	case domain.ImageTypeDocker:
		// TODO: docker clientはここで作るべきではないと思う。もっと上で。docker動くノードですよ〜ってのがわかった時に生成して、エラーならpanicさせるとか
		cli, err := client.NewClientWithOpts()
		if err != nil {
			return err
		}
		j = container.New(cli, opts.stepID, opts.workflowID, opts.name, opts.image, w.LocalIPAddr)
		log.Println("container found")
	case domain.ImageTypeShell:
		j = shell.New(opts.stepID, opts.workflowID, opts.name, opts.image, w.LocalIPAddr)
		log.Println("shell found")
	default:
		return errors.New("invalid jobType")
	}
	log.Println("deploy start")
	go j.Deploy(context.Background())
	return w.JobStore.SetPending(ctx, opts.stepID, j)
}

func (w *Worker) RunJobAfterJobIsReady(ctx context.Context, stepID string, body []byte) error {
	for {
		log.Println("run job after job is ready...")
		time.Sleep(1 * time.Second)
		j, err := w.JobStore.GetFromReady(ctx, stepID)
		if err != nil {
			if err != store.ErrNotFound {
				log.Println(err)
				return err
			}
			continue
		}
		log.Println("deployed. do")
		// job deployed
		jobID, err := w.JobStore.SetRunningFromReady(ctx, stepID)
		if err != nil {
			return err
		}
		if err := j.Do(ctx, jobID, body); err != nil {
			log.Println(err)
			return err
		}
		return nil
	}
}

func (w *Worker) NextJob(ctx context.Context, workflowID, currentStepID, currentJobID string, body []byte) error {
	wf, err := w.WorkflowStore.Get(ctx, workflowID)
	if err != nil {
		return err
	}

	if err := w.JobStore.DeleteRunningJob(ctx, currentJobID); err != nil {
		return err
	}
	nextSteps := wf.NextStepsByCurrentStepID(currentStepID)
	nowstep = wf.StepByCurrentStepID(currentStepID)
	// TODO workflowが終了した時
	if len(nextSteps) == 0 {	
	wkr := &Content{}
	if err = json.Unmarshal(body, wkr); err != nil {
		return err
		
	}
	rbody := wkr.Body
	log.Println(string(rbody))
	log.Printf("worker runtime")
	log.Println(wkr.Runtime)
	log.Printf("worker ram")
	log.Println(wkr.RAM)
	log.Printf("worker cpu")
	log.Println(wkr.CPU)
	
	
	var delay time.Duration 
	delay = 0
	log.Println("Come4!!")
	
    	log.Println(delay)
    	c := http.DefaultClient
	u := *w.MasterInfo.URL
    	
	info := &api.DelayInfo {
	JobName:      nowstep.Name,
	FromWorkerID: w.ID,
	ToWorkerName: "Home",
	RAM:          wkr.RAM,
	CPU:          wkr.CPU,
	Runtime:      wkr.Runtime,
	Time:         delay,
	}
		reqBody, err := json.Marshal(&info)
		
		if err != nil {
		return  err
	}

	u.Path = fmt.Sprintf("/delayinfo")
	
	req2, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(reqBody))
	if err != nil {
		return  err
	}
	_, err = c.Do(req2)
	if err != nil {
		return  err
	}
		log.Println("\n\n\nworkflow end\n\n\n")
		return nil
	}

	eg := errgroup.Group{}

	for _, s := range nextSteps {
		s := s
		eg.Go(func() error {
		
			return w.requestDoStep(ctx, workflowID, s, body)
		})
	}
	return eg.Wait()
}
func (w *Worker) PeriodicGetWorkerLatency(ctx context.Context) {
for {
	time.Sleep(10 * time.Second)
	client := http.DefaultClient
	var send WorkerResponse2
	send.FromID = w.ID
    for _ , i := range w.OtherWorkers {
	 	fmt.Println(i.Name)
		var addwo ToWorker
		var flows string ="hello"
		reqBody2, _ := json.Marshal(flows)
	
	req2, _ := http.NewRequest("POST", i.URL.String()+"/reply", bytes.NewReader(reqBody2))
	
	start := time.Now() 
	res, _ := client.Do(req2)
	
	delay := time.Since(start)
	resBody, _ := ioutil.ReadAll(res.Body)
	log.Println(string(resBody))	
	addwo.Name = i.Name
	addwo.Time = delay / 2
	send.ToWorkers = append(send.ToWorkers,addwo)
	}
	
	reqBody, err := json.Marshal(send)
	if err != nil {
		w.AddError(err)
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/workers/response", w.MasterInfo.URL.String()), bytes.NewReader(reqBody))
	if err != nil {
		w.AddError(err)
	}
	if _, err := client.Do(req); err != nil {
		w.AddError(errors.New(fmt.Sprintf(err.Error())))
	}
	}
}

func (w *Worker) requestDoStep(ctx context.Context, workflowID string,   step *domain.Step, body []byte) error {
	c := http.DefaultClient
	u := *w.MasterInfo.URL
	u.Path = fmt.Sprintf("/workflows/%s/steps/%s/worker/%s", workflowID, step.ID, w.ID)
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	var wk api.ResponseWorker
	if err := json.NewDecoder(resp.Body).Decode(&wk); err != nil {
		return err
	}
	wkr := &Content{}
	if err = json.Unmarshal(body, wkr); err != nil {
		return err
		
	}
	rbody := wkr.Body
	log.Println(string(rbody))
	
	wkURL := fmt.Sprintf("%s/workflows/%s/steps/%s", wk.URL, workflowID, step.ID)
	req, err = http.NewRequest(http.MethodPost, wkURL, bytes.NewReader(rbody))
	if err != nil {
		return err
	}
	if  wk.ID != w.ID{
	start := time.Now()
	resp2, _ := c.Do(req)
	delay := time.Since(start)
	var resBody WorkerResponse
	if err := json.NewDecoder(resp2.Body).Decode(&resBody); err != nil {
		return err
	}
    	
    	
	info := &api.DelayInfo {
	JobName:      nowstep.Name,
	FromWorkerID: w.ID,
	ToWorkerName: wk.Name,
	RAM:          wkr.RAM,
	CPU:          wkr.CPU,
	Runtime:      wkr.Runtime,
	Time:         delay,
	}
		reqBody, err := json.Marshal(&info)
		
		if err != nil {
		return  err
	}

	u.Path = fmt.Sprintf("/delayinfo")
	
	req2, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(reqBody))
	if err != nil {
		return  err
	}
	_, err = c.Do(req2)
	if err != nil {
		return  err
	}
	}
	if wk.ID == w.ID{
	var ded time.Duration = 0
    	
	info := &api.DelayInfo {
	JobName:      nowstep.Name,
	FromWorkerID: w.ID,
	ToWorkerName: wk.Name,
	RAM:          wkr.RAM,
	CPU:          wkr.CPU,
	Runtime:      wkr.Runtime,
	Time:         ded,
	}
		reqBody2, err := json.Marshal(&info)
		
		if err != nil {
		return  err
	}

	u.Path = fmt.Sprintf("/delayinfo")
	
	req2, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(reqBody2))
	if err != nil {
		return  err
	}
	_, err = c.Do(req2)
	if err != nil {
		return  err
	}
	wkURL = fmt.Sprintf("%s/workflows/%s/steps/%s", wk.URL, workflowID, step.ID)
	req, err = http.NewRequest(http.MethodPost, wkURL, bytes.NewReader(rbody))
	_, err = c.Do(req)
	if err != nil {
		return  err
	}

	}
	
	return nil
}

func (w *Worker) RunJob(ctx context.Context, workflowID, stepID string, body []byte) error {
	j, err := w.JobStore.GetFromReady(ctx, stepID)
	switch err {
	case store.ErrNotFound:
		_, err := w.JobStore.GetFromPending(ctx, stepID)
		if err != nil {
			if err != store.ErrNotFound {
				return err
			}
			// ジョブがデプロイされてない、かつデプロイ中でもない時
			go func(ctx context.Context) {
				if err := func(ctx context.Context) error {
					if err := w.DeployJob(ctx, workflowID, stepID); err != nil {
						return err
					}
					return w.RunJobAfterJobIsReady(ctx, stepID, body)
				}(ctx); err != nil {
					// TODO error notify
					log.Println(err)
				}
			}(context.Background())
			return nil
		}
		// ジョブがデプロイされていないが、デプロイ中で完了待ちの時
		go func() {
			if err := w.RunJobAfterJobIsReady(context.Background(), stepID, body); err != nil {
				// TODO error notify
				log.Println(err)
			}
		}()
		return nil
	case nil:
	default:
		return err
	}

	// デプロイ済みの時
	jobID, err := w.JobStore.SetRunningFromReady(ctx, stepID)
	if err != nil {
		return err
	}
	start := time.Now()
	go j.Do(context.Background(), jobID, body)
	time1 := time.Since(start)
	log.Println("run job is finished.......")
	log.Println(time.Now())
	log.Println(time1)
	
	return nil
}

func (w *Worker) StartJobByTriggerHTTPPath(ctx context.Context, triggerPath string, body []byte) error {
	wf, err := w.WorkflowStore.GetByTriggerHTTPPath(ctx, triggerPath)
	if err != nil {
		return err
	}
	eg := errgroup.Group{}
	for _, s := range wf.Steps {
		if s.After != "" {
			continue
		}
		s := s
		eg.Go(func() error {
			return w.RunJob(ctx, wf.ID, s.ID, body)
		})
	}
	return eg.Wait()
}

type WorkerStepStatus struct {
	Step       *domain.Step `json:"step"`
	IsPending  bool         `json:"is_pending"`
	IsDeployed bool         `json:"is_deployed"`
	IsRunning  bool         `json:"is_running"`
}

func (w *Worker) ListStepStatuses(ctx context.Context, workflowID string) ([]*WorkerStepStatus, error) {
	wf, err := w.WorkflowStore.Get(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	jobs, err := w.JobStore.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	eg := errgroup.Group{}
	statuses := make([]*WorkerStepStatus, 0)
	for _, j := range jobs {
		j := j
		eg.Go(func() error {
			var step *domain.Step
			for _, s := range wf.Steps {
				if s.ID == j.StepID() {
					step = s
				}
			}
			if step == nil {
				return errors.New("unknown error. step is nil...")
			}
			pending, err := w.JobStore.IsPending(ctx, j.StepID())
			if err != nil {
				return err
			}
			if pending {
				statuses = append(statuses, &WorkerStepStatus{
					Step:       step,
					IsPending:  true,
					IsDeployed: false,
					IsRunning:  false,
				})
				return nil
			}
			ready, err := w.JobStore.IsReady(ctx, j.StepID())
			if ready {
				s := &WorkerStepStatus{
					Step:       step,
					IsPending:  false,
					IsDeployed: true,
					IsRunning:  false,
				}
				running, err := w.JobStore.IsRunning(ctx, j.StepID())
				if err != nil {
					return err
				}
				s.IsRunning = running
				statuses = append(statuses, s)
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return statuses, nil
}

func (w *Worker) FailJob(ctx context.Context, workflowID, stepID, jobID string, body []byte) error {
	if err := w.JobStore.DeleteRunningJob(ctx, jobID); err != nil {
		return err
	}
	
	w.AddError(errors.New(string(body)))
	wf, err := w.WorkflowStore.Get(ctx, workflowID)
	if err != nil {
		return err
	}
	failureStep := wf.GetFailureStepByFailedStepID(stepID)
	return w.requestDoStep(ctx, workflowID, failureStep, body)
}

func (w *Worker) FinishJob(ctx context.Context, workflowID, stepID, jobID string) error {
	if err := w.JobStore.DeleteRunningJob(ctx, jobID); err != nil {
		return err
	}
	return nil
}
