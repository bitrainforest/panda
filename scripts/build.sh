#!/bin/bash

SCRIPTDIR=$(dirname $0)
BASEDIR=$(dirname $SCRIPTDIR)

# making output directory
mkdir -p $BASEDIR/output/bin $BASEDIR/output/conf

# copy scripts, bootstrap.sh and settings.py
cp $BASEDIR/scripts/bootstrap.sh $BASEDIR/scripts/settings.py $BASEDIR/output/


# read settings
PRODUCT=$(cd $SCRIPTDIR; python -c "import settings; print(settings.PRODUCT)")
SUBSYS=$(cd $SCRIPTDIR; python -c "import settings; print(settings.SUBSYS)")
MODULE=$(cd $SCRIPTDIR; python -c "import settings; print(settings.MODULE)")
if [ -z "$PRODUCT" ] || [ -z "$SUBSYS" ] || [ -z "$MODULE" ]; then
    echo "Support PRODUCT SUBSYS MODULE PORT in settings.py"
    exit -1
fi

# copy configuration files
cp -r $BASEDIR/conf/${PRODUCT}_${SUBSYS}_${MODULE}.yaml $BASEDIR/output/conf/

BinaryName=${PRODUCT}.${SUBSYS}.${MODULE}

# BUILD_VERSION 是 SCM 服务在构建时自动加上的，
VERSION=$(git rev-parse --short HEAD 2>/dev/null || echo ${BUILD_VERSION:-"unkown"})
TIMESTAMP=$(date '+%Y-%m-%dT%H:%M:%S')
GITHASH=$(git rev-parse --short HEAD 2>/dev/null || echo "unkown")
buildFlags="-X main.version=${VERSION} -X main.buildstamp=${TIMESTAMP} -X main.githash=${GITHASH}"

cd $BASEDIR
echo go build -v -o ./output/bin/$BinaryName -v -ldflags "${buildFlags}" ./cmd/.
go build -o ./output/bin/$BinaryName -v -ldflags "${buildFlags}" ./cmd/.

chmod +x $BASEDIR/output/bin/* $BASEDIR/output/bootstrap.sh
