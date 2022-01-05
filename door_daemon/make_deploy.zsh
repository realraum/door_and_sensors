#!/bin/zsh
REMOTE_USER=root
REMOTE_HOST=torwaechter.mgmt.realraum.at
REMOTE_DIR=/usr/local/bin/

export GO386=softfloat
export CGO_ENABLED=0
export GOOS=linux
export GOARCH=386

ping -W 1 -c 1 ${REMOTE_HOST} || { OPTIONS=(-o ProxyJump='gw.realraum.at:22000'); RSYNCOPTIONS=(-e "ssh $OPTIONS")}
go build "$@" -ldflags "-s" && rsync ${RSYNCOPTIONS[@]} -rvp --delay-updates --progress --delete ${PWD:t} ${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_DIR}/   && {echo "Restart Daemon? [Yn]"; read -q && ssh ${OPTIONS[@]} ${REMOTE_USER}@$REMOTE_HOST /etc/init.d/doord restart; return 0}
