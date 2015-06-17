#!/bin/sh
#
# add all ssh keys which should be allowed to set the state to
# authorized_keys file for user www-data using something like this:
#
# command="/usr/local/bin/ssh-status-json.sh",no-X11-forwarding,no-agent-forwarding,no-port-forwarding ssh-rsa <public-key> <descriptive name>
#

SHM_D="/dev/shm/"
WWW_D="$SHM_D/www"
FILENAME="status.json"

mkdir -p $WWW_D

command=`echo $SSH_ORIGINAL_COMMAND | awk '{ print $1 }'`
case $command in
  set)
    tee "$SHM_D/$FILENAME" | python -m simplejson.tool > /dev/null
    if [ $? -eq 0 ]; then
      mv "$SHM_D/$FILENAME" "$WWW_D/$FILENAME"
    else
      rm -f "$SHM_D/$FILENAME"
    fi
    ;;
  get)
    cat "$WWW_D/$FILENAME"
    ;;
  *)
    echo "unknown command: '$command'"
    ;;
esac

exit 0
