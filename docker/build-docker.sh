#!/bin/sh

set -e -x

cd $(dirname "$0")

go build ..

docker build -t tgres .
