package main

import (
	"bytes"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func main() {
	log.Fatal(run())
}

func run() error {

	c1 := exec.Command("cat", "")
	c2 := exec.Command("sh")
	c2.Env = append(os.Environ(), "takuhaiJobPort=1234", "managerAddr=hoge", "workflowID=hoge", "stepID=stepID")
	r, w := io.Pipe()
	c1.Stdout = w
	c2.Stdin = r
	var out, outErr bytes.Buffer
	c2.Stdout = &out
	c2.Stderr = &outErr

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
	if err := c2.Wait(); err != nil {
		log.Println(outErr.String())
		return err
	}
	log.Println(out.String())

	//stdout, err := c2.StdoutPipe()
	//if err != nil {
	//	log.Fatal(err)
	//}
	//stderr, err := c2.StderrPipe()
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//err = c2.Start()
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//streamReader := func(scanner *bufio.Scanner, outputChan chan string, doneChan chan bool) {
	//	defer close(outputChan)
	//	defer close(doneChan)
	//	for scanner.Scan() {
	//		outputChan <- scanner.Text()
	//	}
	//	doneChan <- true
	//}
	//
	//// stdout, stderrをひろうgoroutineを起動
	//stdoutScanner := bufio.NewScanner(stdout)
	//stdoutOutputChan := make(chan string)
	//stdoutDoneChan := make(chan bool)
	//stderrScanner := bufio.NewScanner(stderr)
	//stderrOutputChan := make(chan string)
	//stderrDoneChan := make(chan bool)
	//go streamReader(stdoutScanner, stdoutOutputChan, stdoutDoneChan)
	//go streamReader(stderrScanner, stderrOutputChan, stderrDoneChan)
	//
	//// channel経由でデータを引っこ抜く
	//stillGoing := true
	//for stillGoing {
	//	select {
	//	case <-stdoutDoneChan:
	//		stillGoing = false
	//	case line := <-stdoutOutputChan:
	//		log.Println(line)
	//	case line := <-stderrOutputChan:
	//		log.Println(line)
	//	}
	//}
	//if err := c2.Wait(); err != nil {
	//	log.Println(outErr.String())
	//	return err
	//}
	log.Println(out.String(), outErr.String())
	log.Println(runtime.GOARCH)
	log.Println(filepath.Abs("hoge"))
	return nil
}
