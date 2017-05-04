#!/bin/zsh
REMOTE_USER=realraum
REMOTE_HOST=smsgw.realraum.at
REMOTE_DIR=/home/realraum/bin/

ping -W 1 -c 1 ${REMOTE_HOST} || { OPTIONS=(-o ProxyCommand='ssh gw.realraum.at exec nc '$REMOTE_HOST' 22000'); RSYNCOPTIONS=(-e 'ssh -o ProxyCommand="ssh gw.realraum.at exec nc '$REMOTE_HOST' 22000"')}
export GOOS=linux
export GOARCH=arm
export CGO_ENABLED=0
go build "$@"  && rsync ${RSYNCOPTIONS[@]} -rvp --delay-updates --progress --delete ${PWD:t} ${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_DIR}/ \
&& {echo "Restart Daemon? [Yn]"; read -q \
&& ssh ${OPTIONS[@]} ${REMOTE_USER}@$REMOTE_HOST systemctl --user restart ${PWD:t}.service; return 0}
