#!/bin/zsh
REMOTE_HOST=mqtt.realraum.at

#ping -W 1 -c 1 $REMOTE_HOST || OPTIONS=(-e "ssh -o ProxyCommand='ssh gw.realraum.at exec nc %h %p'")
export GOOS=linux
export GOARCH=arm
export CGO_ENABLED=0
go build "$@"  && rsync ${OPTIONS[@]} -v --delay-updates --progress ${PWD:t} realraum@$REMOTE_HOST:bin/
