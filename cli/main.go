package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"

	"github.com/mobmob912/takuhai/master/master"

	"github.com/mobmob912/takuhai/domain"

	"gopkg.in/yaml.v2"
)

// const URL = "http://34.85.80.13:3000"

const URL = "http://localhost:3000"

// const URL = "http://10.31.22.29:3000"

func main() {

	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	args := os.Args

	if len(args) < 3 {
		return errors.New("missing required arguments")
	}

	cmd := args[1]

	switch cmd {
	case "worker":
		return worker(args)
	case "workflow":
		return workflow(args)
	}
	return errors.New("no commands matched")
}

func worker(args []string) error {
	cmd := args[2]

	switch cmd {
	case "list":
		return workerList(args)
	}
	return nil
}

func workerList(args []string) error {
	client := http.DefaultClient
	req, err := http.NewRequest("GET", URL+"/workers", nil)
	if err != nil {
		return err
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	log.Println(string(body))
	return nil
}

func workflow(args []string) error {
	cmd := args[2]

	switch cmd {
	case "add":
		return addWorkflow(args)
	case "status":
		return workflowStatus(args)
	}
	return nil
}

func addWorkflow(args []string) error {

	path := args[3]
	ss := strings.Split(path, string(os.PathSeparator))
	workDir := strings.Join(ss[:len(ss)-1], string(os.PathSeparator))

	file, err := os.Open(path)
	if err != nil {
		return err
	}

	var wf domain.Workflow
	if err := yaml.NewDecoder(file).Decode(&wf); err != nil {
		return err
	}

	for jI, j := range wf.Jobs {
		for imgI, img := range j.Images {
			if img.Type == domain.ImageTypeShell {
				shellFile, err := os.Open(fmt.Sprintf("%s/%s", workDir, img.Image))
				if err != nil {
					return err
				}
				shell, err := ioutil.ReadAll(shellFile)
				if err != nil {
					return err
				}
				// TODO いい感じにパース
				wf.Jobs[jI].Images[imgI].Image = string(shell)
			}
		}
	}
	body, err := json.Marshal(wf)
	if err != nil {
		return err
	}
	log.Println(string(body))
	client := http.DefaultClient
	req, err := http.NewRequest("POST", URL+"/workflows", bytes.NewReader(body))
	if err != nil {
		return err
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	log.Println(string(resBody))
	return nil
}

func workflowStatus(args []string) error {
	log.Println("called")
	workflowName := args[3]
	c := http.DefaultClient
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/workflows/%s/status", URL, workflowName), nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	var status master.WorkflowStatuses
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return err
	}
	// header := []string{"", "IS PENDING", "IS READY", "IS RUNNING"}

	header := []string{"STEP"}
	ps := []string{"IS PENDING"}
	rs := []string{"IS READY"}
	us := []string{"IS RUNNING"}
	for _, s := range status.Statuses {
		header = append(header, s.Step.Name)
		pp := []string{}
		rr := []string{}
		uu := []string{}
		for _, wk := range s.DeployedWorker {
			name := wk.Worker.Name
			if wk.IsPending {
				pp = append(pp, name)
			} else {
				rr = append(rr, name)
			}
			if wk.IsRunning {
				uu = append(uu, name)
			}
		}
		ps = append(ps, strings.Join(pp, ", "))
		rs = append(rs, strings.Join(rr, ", "))
		us = append(us, strings.Join(uu, ", "))
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.AppendBulk([][]string{header, ps, rs, us})

	// rows := [][]string{}
	// for _, s := range status.Statuses {
	// 	ps := []string{}
	// 	rs := []string{}
	// 	us := []string{}
	// 	for _, wk := range s.DeployedWorker {
	// 		name := wk.Worker.Name
	// 		if wk.IsPending {
	// 			ps = append(ps, name)
	// 		} else {
	// 			rs = append(rs, name)
	// 		}
	// 		if wk.IsRunning {
	// 			us = append(us, name)
	// 		}
	// 	}
	// 	rows = append(rows, []string{s.Step.Name, strings.Join(ps, ", "), strings.Join(rs, ", "), strings.Join(us, ", ")})
	// }
	// table := tablewriter.NewWriter(os.Stdout)
	// table.SetHeader(header)
	// for _, row := range rows {
	// 	table.Append(row)
	// }
	table.Render()
	return nil
}
