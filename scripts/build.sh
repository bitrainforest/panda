#!/bin/bash

SCRIPTDIR=$(dirname $0)
BASEDIR=$(dirname $SCRIPTDIR)

# making output directory
mkdir -p $BASEDIR/output/bin $BASEDIR/output/conf

BinaryName="Panda"

cd $BASEDIR
echo go build -v -o ./output/bin/$BinaryName -v  ./cmd/.
go build -o ./output/bin/$BinaryName -v  ./cmd/.

chmod +x $BASEDIR/output/bin/*
