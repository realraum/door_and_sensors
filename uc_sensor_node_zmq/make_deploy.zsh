#!/bin/zsh
export GO386=387
export CGO_ENABLED=1
go-linux-386 clean
go-linux-386 build -ldflags "-s" && rsync -v --progress ${PWD:t} gw.realraum.at:/flash/home/realraum/
#go-linux-386 build -ldflags " --linkmode external -extldflags \"-L . -Bstatic ./libc.so.6 ./libglib-2.0.so.0 ./libstdc++.so.6 ./libm.so.6 ./libpthread.a ./librt.a\" "
