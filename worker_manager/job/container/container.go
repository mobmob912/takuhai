package container

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"

	"github.com/docker/docker/api/types/filters"

	"github.com/docker/go-connections/nat"

	"github.com/docker/docker/pkg/stdcopy"

	docker_container "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/mobmob912/takuhai/worker_manager/job"
)

type container struct {
	stepID           string // stepID
	workflowID       string
	client           *client.Client
	jobName          string
	image            string
	deployed         bool
	addr             *url.URL
	managerLocalAddr *net.IP
	hostIP           net.IP
	err              error
}

func New(cli *client.Client, id, workflowID, jobName, image string, managerLocalAddr *net.IP) job.Job {
	return &container{
		stepID:           id,
		workflowID:       workflowID,
		client:           cli,
		jobName:          jobName,
		image:            image,
		managerLocalAddr: managerLocalAddr,
	}
}

func (c *container) StepID() string {
	return c.stepID
}

func (c *container) Name() string {
	return c.jobName
}

func (c *container) Deploy(ctx context.Context) error {
	log.Println("pulling image...")
	resp, err := c.client.ImagePull(ctx, c.image, types.ImagePullOptions{})
	if err != nil {
		c.err = err
		return err
	}
	io.Copy(os.Stdout, resp)

	log.Println("pull success")

	portStr, err := getEmptyPort()
	if err != nil {
		return err
	}
	port := nat.Port(portStr)
	c.addr, err = url.Parse(fmt.Sprintf("http://0.0.0.0:%s", portStr))
	if err != nil {
		return err
	}

	containerName := fmt.Sprintf("%s-%s", c.jobName, c.stepID)

	cs, err := c.client.ContainerList(ctx, types.ContainerListOptions{
		Filters: filters.NewArgs(filters.Arg("name", containerName)),
	})
	if err != nil {
		return err
	}
	for _, cc := range cs {
		log.Println("previous container found. delete " + cc.ID)
		_ = c.client.ContainerRemove(ctx, cc.ID, types.ContainerRemoveOptions{Force: true})
	}

	log.Println("container creating...")

	portBindings := nat.PortMap{
		port: []nat.PortBinding{{HostIP: "", HostPort: portStr}},
	}
	body, err := c.client.ContainerCreate(ctx, &docker_container.Config{
		Image: c.image,
		Env: []string{
			"takuhaiJobPort=" + portStr,
			"managerAddr=" + c.managerLocalAddr.String(),
			"workflowID=" + c.workflowID,
			"stepID=" + c.stepID,
		},
		ExposedPorts: nat.PortSet{port: struct{}{}},
	},
		&docker_container.HostConfig{
			PortBindings: portBindings,
			ExtraHosts:   []string{fmt.Sprintf("takuhai_host:%s", c.managerLocalAddr.String())},
		},
		&network.NetworkingConfig{},
		containerName,
	)
	if err != nil {
		return err
	}
	log.Println("container creating success")
	log.Println("container starting...")
	if err := c.client.ContainerStart(ctx, body.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}
	log.Println("container starting success")

	go c.Logging(context.Background(), body.ID)

	log.Println("waiting...")
	return nil
}

// TODO いい感じの場所へログをはく
func (c *container) Logging(ctx context.Context, containerID string) error {
	reader, err := c.client.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true})
	if err != nil {
		return err
	}
	stdcopy.StdCopy(os.Stdout, os.Stderr, reader)
	return nil
}

//func (c *container) WaitJobDeployed(ctx context.Context) {
//	for {
//		time.Sleep(1 * time.Second)
//		cli := http.DefaultClient
//		u := *c.addr
//		u.Path = "/check"
//		req, err := http.NewRequest("GET", u.String(), nil)
//		if err != nil {
//			continue
//		}
//		_, err = cli.Do(req)
//		if err != nil {
//			continue
//		}
//		c.deployed = true
//		break
//	}
//}

func (c *container) Do(ctx context.Context, jobID string, body []byte) error {
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
