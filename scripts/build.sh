#!/bin/bash

SCRIPTDIR=$(dirname $0)
BASEDIR=$(dirname $SCRIPTDIR)

# making output directory
mkdir -p $BASEDIR/output/bin $BASEDIR/output/conf

BinaryName="panda"

cd $BASEDIR
echo go build -v -o ./output/bin/$BinaryName -v  ./cmd/.
go build -o ./output/bin/$BinaryName -v  ./cmd/.

cp ./output/bin/$BinaryName /usr/local/bin/

chmod +x $BASEDIR/output/bin/*
