package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/mobmob912/takuhai/domain"

	"github.com/mobmob912/takuhai/master/api"
	"github.com/mobmob912/takuhai/worker_manager/external_api"
	"github.com/mobmob912/takuhai/worker_manager/internal_api"
	"github.com/mobmob912/takuhai/worker_manager/store"
	"github.com/mobmob912/takuhai/worker_manager/worker"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {

	log.SetFlags(0)

	log.Println("==============================================================================")
	log.Println("|                                                                            |")
	log.Println("|                        takuhai worker manager                              |")
	log.Println("|                                                                            |")
	log.Println("==============================================================================\n")

	var name, argWorkerGlobalIP, argWorkerLocalIP, workerPort, argMasterIP, masterPort, workerType, place, labelsStr string
	flag.StringVar(&name, "name", "", "worker name")
	flag.StringVar(&argWorkerGlobalIP, "workerGlobalIP", "", "worker global ip addr")
	flag.StringVar(&argWorkerLocalIP, "workerLocalIP", "", "worker local ip addr")
	flag.StringVar(&workerPort, "workerPort", "4871", "worker manager port")
	flag.StringVar(&argMasterIP, "masterIP", "", "master ip addr")
	flag.StringVar(&masterPort, "masterPort", "3000", "master server port")
	flag.StringVar(&workerType, "workerType", "docker", "worker type (ex: docker, shell")
	flag.StringVar(&place, "place", "edge", "worker place (edge or cloud or device)")
	flag.StringVar(&labelsStr, "labels", "", "worker labels. comma split")
	flag.Parse()

	if name == "" {
		return errors.New("name is missing")
	}

	if argMasterIP == "" {
		return errors.New("master ip addr is missing")
	}
	masterAddr := fmt.Sprintf("http://%s:%s", argMasterIP, masterPort)

	if argWorkerGlobalIP == "" {
		return errors.New("worker global ip addr is missing")
	}
	workerGlobalAddr := fmt.Sprintf("http://%s:%s", argWorkerGlobalIP, workerPort)

	workerLocalIP := net.ParseIP(argWorkerLocalIP)
	if argWorkerLocalIP == "" {
		selectedIP, err := getWorkerIP()
		if err != nil {
			return err
		}
		workerLocalIP = *selectedIP
	}
	workerLocalAddr := fmt.Sprintf("http://%s:%s", workerLocalIP.String(), workerPort)

	u, err := url.Parse(masterAddr)
	if err != nil {
		return err
	}

	log.Printf("masterAddr: %s, workerLocalAddr: %s", masterAddr, workerLocalAddr)

	labels := strings.Split(labelsStr, ",")

	m := &worker.MasterInfo{
		URL: u,
	}
	js := store.NewJob()
	ws := store.NewWorkflow()
	w := worker.New(&worker.OptionsNew{
		Type:          domain.ImageType(workerType),
		Arch:          domain.ArchType(runtime.GOARCH),
		Place:         domain.Place(place),
		Labels:        labels,
		MasterInfo:    m,
		IPAddr:        &workerLocalIP,
		JobStore:      js,
		WorkflowStore: ws,
	})

	ctx := context.Background()

	internalServer := internal_api.NewServer(w)
	externalServer := external_api.NewServer(ws, js, w)

	go externalServer.Serve()

	for {
		if externalServer.IsServing() {
			break
		}
		log.Println("check external server is serving")
		time.Sleep(1 * time.Second)
	}

	// TODO: k8s対応のために、複数worker登録するようにする。普通なら一個
	id, err := registerWorkerToMaster(name, workerGlobalAddr, masterAddr, w)
	if err != nil {
		log.Println(err)
		return err
	}
	w.ID = id

	if place != "device" {
		// TODO: 定期的にワーカーのリソース情報送る ↑と同じく、複数情報送れるように
		go w.PeriodicNotifyResourceInformationToMaster(ctx)
	}

	go w.PeriodicGetWorkflows(ctx)
	go w.PeriodicCheckErrors(ctx)

	return internalServer.Serve()
}

func getWorkerIP() (*net.IP, error) {
	ifes, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	log.Println("Which interface will you use?")
	for i, ife := range ifes {
		msg := fmt.Sprintf("INDEX: %d NAME: %s, ADDRS: ", i, ife.Name)
		addrs, err := ife.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			msg += fmt.Sprintf("%s, ", addr.String())
		}
		log.Printf("%s\n", msg)
	}
	log.Print("Select INDEX: ")
	for {
		selectedInterfaceIdx, err := getIdx(len(ifes))
		if err != nil {
			log.Println(err.Error())
			continue
		}
		addrs, err := ifes[selectedInterfaceIdx].Addrs()
		if err != nil {
			return nil, err
		}
		if len(addrs) == 0 {
			log.Println("Error: no addr found")
			continue
		}
		ip, _, err := net.ParseCIDR(addrs[0].String())
		if len(addrs) == 1 {
			return &ip, nil
		}
		for {
			log.Println("Choose addr")
			for i, addr := range addrs {
				log.Printf("%d: %s", i, addr.String())
			}
			selectedAddrIdx, err := getIdx(len(addrs))
			if err != nil {
				log.Println(err.Error())
				continue
			}
			ip, _, err := net.ParseCIDR(addrs[selectedAddrIdx].String())
			if err != nil {
				return nil, err
			}
			return &ip, nil
		}
	}
}

func getIdx(upperLimit int) (int, error) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	parsed, err := strconv.ParseInt(scanner.Text(), 10, 64)
	if err != nil {
		return 0, errors.New("index parse error")
	}
	if parsed < 0 || parsed > int64(upperLimit) {
		return 0, errors.New("select a correct index number")
	}
	return int(parsed), nil
}

//
//type Stats struct {
//	CPU float64
//	RAM int64
//}
//
//func getStats() (*Stats, error) {
//	v, err := mem.VirtualMemory()
//	if err != nil {
//		return nil, err
//	}
//	cs, err := cpu.Info()
//	if err != nil {
//		return nil, err
//	}
//	return &Stats{
//		CPU: cs[0].Mhz,
//		RAM: int64(v.Available),
//	}, nil
//}
//
// return id, error
func registerWorkerToMaster(name, workerAddr, masterAddr string, wk *worker.Worker) (string, error) {
	workerInfo := &api.WorkerInfoRequest{
		Name:   name,
		URL:    workerAddr,
		Type:   wk.Type,
		Arch:   wk.Arch,
		Place:  wk.Place,
		Labels: wk.Labels,
	}
	body, err := json.Marshal(&workerInfo)
	if err != nil {
		return "", err
	}

	client := http.DefaultClient
	req, err := http.NewRequest("POST", masterAddr+"/workers", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	if res.StatusCode >= 400 {
		resBody, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return "", err
		}
		return "", errors.New("add worker error: " + string(resBody))
	}
	resBody := &api.AddWorkerResponse{}
	if err := json.NewDecoder(res.Body).Decode(resBody); err != nil {
		return "", err
	}
	return resBody.ID, nil
}
