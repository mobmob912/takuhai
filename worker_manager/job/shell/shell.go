package shell

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"time"

	"github.com/mobmob912/takuhai/worker_manager/job"
)

type shell struct {
	stepID      string
	workflowID  string
	jobName     string
	shell       string
	addr        *url.URL
	deployed    bool
	managerAddr *net.IP
	hostIP      net.IP
	err         error
}

func New(id, workflowID, jobName, sh string, managerAddr *net.IP) job.Job {
	return &shell{
		stepID:      id,
		workflowID:  workflowID,
		jobName:     jobName,
		shell:       sh,
		managerAddr: managerAddr,
	}
}

func (c *shell) StepID() string {
	return c.stepID
}

func (c *shell) Name() string {
	return c.jobName
}

func (c *shell) Deploy(ctx context.Context) error {
	portStr, err := getEmptyPort()
	if err != nil {
		return err
	}
	c.addr, err = url.Parse(fmt.Sprintf("http://0.0.0.0:%s", portStr))
	if err != nil {
		return err
	}
	file, err := os.Create(c.stepID)
	if err != nil {
		return err
	}
	defer file.Close()
	file.Write([]byte(c.shell))
	defer os.Remove(c.stepID)
	c1 := exec.Command("cat", c.stepID)
	c2 := exec.Command("sh")
	c2.Env = append(os.Environ(),
		"takuhaiJobPort="+portStr,
		"managerAddr="+c.managerAddr.String(),
		"workflowID="+c.workflowID,
		"stepID="+c.stepID,
	)
	r, w := io.Pipe()
	c1.Stdout = w
	c2.Stdin = r
	stdout, err := c2.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := c2.StderrPipe()
	if err != nil {
		return err
	}

	if err := c1.Start(); err != nil {
		return err
	}
	if err := c2.Start(); err != nil {
		return err
	}
	if err := c1.Wait(); err != nil {
		return err
	}
	w.Close()
	go c.Logging(ctx, stdout, stderr)

	log.Println("waiting...")
	return nil
}

// TODO いい感じの場所へログをはく
func (c *shell) Logging(ctx context.Context, stdout, stderr io.ReadCloser) error {
	streamReader := func(scanner *bufio.Scanner, outputChan chan string, doneChan chan bool) {
		defer close(outputChan)
		defer close(doneChan)
		for scanner.Scan() {
			outputChan <- scanner.Text()
		}
		doneChan <- true
	}

	stdoutScanner := bufio.NewScanner(stdout)
	stdoutOutputChan := make(chan string)
	stdoutDoneChan := make(chan bool)
	stderrScanner := bufio.NewScanner(stderr)
	stderrOutputChan := make(chan string)
	stderrDoneChan := make(chan bool)
	go streamReader(stdoutScanner, stdoutOutputChan, stdoutDoneChan)
	go streamReader(stderrScanner, stderrOutputChan, stderrDoneChan)

	stillGoing := true
	for stillGoing {
		select {
		case <-stdoutDoneChan:
			stillGoing = false
		case line := <-stdoutOutputChan:
			log.Println(line)
		case line := <-stderrOutputChan:
			log.Println(line)
		}
	}
	return nil
}

func (c *shell) WaitJobDeployed(ctx context.Context) {
	for {
		time.Sleep(1 * time.Second)
		cli := http.DefaultClient
		u := *c.addr
		u.Path = "/check"
		req, err := http.NewRequest("GET", u.String(), nil)
		if err != nil {
			continue
		}
		_, err = cli.Do(req)
		if err != nil {
			continue
		}
		c.deployed = true
		break
	}
}

func (c *shell) Do(ctx context.Context, jobID string, body []byte) error {
	cli := http.DefaultClient
	u := *c.addr
	u.Path = "/do"
	req, err := http.NewRequest("POST", u.String(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("takuhai-job-id", jobID)
	_, err = cli.Do(req)
	if err != nil {
		return err
	}
	return nil
}

func getEmptyPort() (string, error) {
	l, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		return "", err
	}
	defer l.Close()
	addr := l.Addr().String()
	_, port, err := net.SplitHostPort(addr)
	return port, nil
}
