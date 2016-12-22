#!/bin/sh

set -e

CONFIG_FILE=$GOPATH/etc/tgres.conf

envsubst <${CONFIG_FILE}.template >${CONFIG_FILE}

$GOPATH/bin/tgres -c $CONFIG_FILE
