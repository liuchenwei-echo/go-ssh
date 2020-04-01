#!/bin/bash

export GO111MODULE="on"
go mod tidy

VERSION=v0.1.0
BUILD_ON=`date +"%Y-%m-%dT%H:%M:%S"`
USER=`whoami`

mkdir -p ./bin
go build -o "./bin/s" -ldflags "-X main.Version=${VERSION} -X main.BuildOn=${BUILD_ON} -X main.User=${USER}" cmd/cmd.go
