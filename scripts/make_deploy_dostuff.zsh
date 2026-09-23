#!/bin/zsh
REMOTE_USER=realraum
REMOTE_HOST=iotstuff.mgmt.realraum.at
REMOTE_DIR=/home/realraum/bin/
FILES=(dostuff_switch_lights.py)

ping -W 1 -c 1 ${REMOTE_HOST} || { OPTIONS=(-o ProxyJump='gw3.realraum.at'); RSYNCOPTIONS=(-e 'ssh -o ProxyJump=gw3.realraum.at')}
rsync ${RSYNCOPTIONS[@]} -rvp --delay-updates --chown $REMOTE_USER:$REMOTE_USER --progress --delete ${FILES} ${REMOTE_HOST}:${REMOTE_DIR}/ \
&& {echo "Restart Daemon? [Yn]"; read -q \
&& ssh ${OPTIONS[@]} $REMOTE_HOST systemctl --machine=${REMOTE_USER}@.host --user restart ${FILES:r:s/(#e)/.service/}; return 0}

