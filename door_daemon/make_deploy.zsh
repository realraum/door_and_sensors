#!/bin/zsh
REMOTE_HOST=torwaechter.mgmt.realraum.at
export GO386=387
export CGO_ENABLED=0
export GOOS=linux
export GOARCH=386

ping -W 1 -c 1 $REMOTE_HOST || OPTIONS=(-e "ssh -o ProxyCommand='ssh gw.realraum.at exec nc %h %p'")
go clean
go build "$@" -ldflags "-s" && rsync ${OPTIONS[@]} -v --delay-updates --progress ${PWD:t} root@$REMOTE_HOST:/flash/tuer/
