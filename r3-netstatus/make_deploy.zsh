#!/bin/zsh
export GO386=387
go-linux-386 clean
#go-linux-386 build
#strip ${PWD:t}
go-linux-386 build -ldflags "-s" && rsync --progress -v ${PWD:t} wuzzler.realraum.at:/flash/tuer/
