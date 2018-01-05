#!/bin/sh

set -e
set -x

##-------------------------------------------------------------

gocommon() {
    pushd $root/pz-gocommon

    cd gocommon
    go test -v
    cd ..

    cd elasticsearch
    go test -v
    cd systest
    #go test -v
        # systest won't run w/o an ES instance present, so just test the compile
        cp -f sys_test.go x.go
        go build x.go
        rm -f x.go x
    cd ../..

    cd kafka
    go test -v
    cd systest
    #go test -v
        # systest won't run w/o a Kafka instance present, so just test the compile
        cp -f sys_test.go x.go
        go build x.go
        rm -f x.go x
    cd ../..

    cd syslog
    go test -v
    cd ..

    popd
}

##-------------------------------------------------------------

logger() {
    pushd $root/pz-logger

    cd logger
    go test -v
    cd ..
    cd systest
    ################go test -v
    cd ..
    go build main.go
    rm -f main

    popd
}

##-------------------------------------------------------------

uuidgen() {
    pushd $root/pz-uuidgen

    cd uuidgen
    go test -v
    cd ..
    cd systest
    go test -v
    cd ..
    go build main.go
    rm -f main

    popd
}

##-------------------------------------------------------------

workflow() {
    pushd $root/pz-workflow

    cd workflow
    go test -v
    cd ..
    cd systest
    go test -v
    cd ..
    go build main.go
    rm -f main

    popd
}

##-------------------------------------------------------------

root=$GOPATH/src/github.com/venicegeo

gocommon
logger
uuidgen
workflow
