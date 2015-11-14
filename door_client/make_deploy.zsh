#!/bin/zsh
export GO386=387
export CGO_ENABLED=1
go-linux-386 clean
#go-linux-386 build
#strip ${PWD:t}
go-linux-386 build -ldflags "-s" && rsync -v --progress ${PWD:t} zmqbroker.realraum.at:/flash/tuer/
