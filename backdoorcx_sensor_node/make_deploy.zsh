#!/bin/zsh
### for 386:
#export GO386=387
#export CGO_ENABLED=1
#go-linux-386 clean
#go-linux-386 build -ldflags "-s" && rsync -v --progress --delay-updates ${PWD:t} realraum@gw.realraum.at:/flash/home/realraum/

### for arm:
#export CFLAGS="I/home/bernhard/source/zeromq/zeromq3-4.0.5+dfsg/include/ -L/home/bernhard/source/zeromq/zeromq3-4.0.5+dfsg/src/.libs/"
#export CGO_ENABLED=1
#export LDFLAGS="I/home/bernhard/source/zeromq/zeromq3-4.0.5+dfsg/include/ -L/home/bernhard/source/zeromq/zeromq3-4.0.5+dfsg/src/.libs/"
#export CC=/usr/bin/arm-linux-gnueabi-gcc-5
#export CXX=/usr/bin/arm-linux-gnueabi-g++-5
#export RANLIB_FOR_TARGET=/usr/bin/arm-linux-gnueabi-gcc-ranlib-5
export GOOS=linux
export GOARCH=arm
go build "$@" && rsync -v --progress --delay-updates ${PWD:t} realraum@smsgw.mgmt.realraum.at:

