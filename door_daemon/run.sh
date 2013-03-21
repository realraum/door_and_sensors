#!/bin/sh
clear
rm /tmp/test.sock 2>/dev/null
go build && ./door_daemon_go /dev/ttyACM0 || sleep 5
