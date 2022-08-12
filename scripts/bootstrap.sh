#!/bin/bash

CURDIR=$(dirname $0)
if [ "X$1" != "X" ]; then
    RUNTIME_ROOT=$1
else
    RUNTIME_ROOT=${CURDIR}
fi
PRODUCT=$(cd $CURDIR; python -c "import settings; print(settings.PRODUCT)")
SUBSYS=$(cd $CURDIR; python -c "import settings; print(settings.SUBSYS)")
MODULE=$(cd $CURDIR; python -c "import settings; print(settings.MODULE)")
if [ -z "$PRODUCT" ] || [ -z "$SUBSYS" ] || [ -z "$MODULE" ]; then
    echo "Support PRODUCT SUBSYS MODULE PORT in settings.py"
    exit -1
fi

CONF_FILE=$CURDIR/conf
LOG_DIR=$RUNTIME_ROOT/log
BinaryName=${PRODUCT}.${SUBSYS}.${MODULE}

mkdir -p $LOG_DIR

echo $CURDIR/bin/$BinaryName -conf-dir "$CONF_DIR" -log-dir $LOG_DIR run
$CURDIR/bin/$BinaryName -conf-dir "$CONF_FILE" -log-dir $LOG_DIR run 
