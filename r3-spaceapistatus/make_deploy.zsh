#!/bin/zsh
export GO386=387
export CGO_ENABLED=1
go-linux-386 clean
#go-linux-386 build
#strip ${PWD:t}
ping -W 1 -c 1 zmqbroker.realraum.at || OPTIONS=(-e "ssh -o ProxyCommand='ssh gw.realraum.at exec nc %h %p'")
go-linux-386 build -ldflags "-s" && rsync ${OPTIONS[@]} -v --progress ${PWD:t} zmqbroker.realraum.at:/flash/tuer/
