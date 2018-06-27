#! /bin/bash

set -e
set -x

export PATH=/usr/local/go/bin:/go/bin:$PATH
. ${HOME}/.bashrc

cd /go/src/gitlab.com/crankykernel/maker
make install-deps

GOOS=linux GOARCH=amd64 make dist
CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 make dist
CC=i686-w64-mingw32-gcc GOOS=windows GOARCH=386 make dist

cp -a ./dist/* /dist/
