#!/bin/sh

export CGO_CPPFLAGS="-I/usr/include"
export CGO_LDFLAGS="-L/usr/lib -L/usr/lib/x86_64-linux-gnu -lzmq -lpthread -lsodium -lrt -lstdc++ -lm -lc -lgcc"
go build --ldflags '-extldflags "-static"' -v .
