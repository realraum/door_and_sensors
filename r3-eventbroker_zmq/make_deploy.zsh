#!/bin/zsh
export GO386=387
export CGO_ENABLED=1
go clean
#go-linux-386 build
#strip ${PWD:t}
go build -ldflags "-s" && rsync -v ${PWD:t} wuzzler.realraum.at:/flash/tuer/
