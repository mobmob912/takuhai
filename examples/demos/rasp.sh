#!/bin/sh

go get github.com/tockn/takuhai-sdk-go
cd $GOPATH/src/github.com/tockn/takuhai-sdk-go/examples/echo
GO111MODULE=on go run main.go