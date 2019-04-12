#! /bin/bash

set -exuo pipefail

WORKSPACE=$(pwd)

docker build -t eirinistaging/integration-tests:latest -f ./image/integration-tests/Dockerfile .
docker run --rm \
  -v ${WORKSPACE}:/eirinistaging \
  eirinistaging/integration-tests ginkgo -r
