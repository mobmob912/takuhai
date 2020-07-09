package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/mobmob912/takuhai/master/api"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

type Stats struct {
	CPU float64
	RAM int64
}

func getStats() (*Stats, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}
	cs, err := cpu.Info()
	if err != nil {
		return nil, err
	}
	return &Stats{
		CPU: cs[0].Mhz,
		RAM: int64(v.Available),
	}, nil
}

func run(name, workerAddr, masterAddr string) error {
	getStats()
	stats, err := getStats()
	if err != nil {
		return err
	}

	workerInfo := &api.WorkerInfoRequest{
		Name: name,
		URL:  workerAddr,
		CPU:  stats.CPU,
		RAM:  stats.RAM,
	}
	body, err := json.Marshal(&workerInfo)
	if err != nil {
		return err
	}

	client := http.DefaultClient
	req, err := http.NewRequest("POST", masterAddr+"/workers", bytes.NewReader(body))
	if err != nil {
		return err
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	log.Println(ioutil.ReadAll(res.Body))
	return nil
}

func main() {

	var name, workerAddr, masterAddr string
	flag.StringVar(&name, "name", "worker name", "worker name")
	flag.StringVar(&workerAddr, "workerAddr", "", "worker addr")
	flag.StringVar(&masterAddr, "masterAddr", "", "master addr")
	flag.Parse()

	//if addr == "" {
	//	panic("addr is missing")
	//}

	if err := run(name, workerAddr, masterAddr); err != nil {
		panic(err)
	}
}
