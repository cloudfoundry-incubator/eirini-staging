#!/bin/bash

set -exuo pipefail

WORKSPACE=$(pwd)

docker run -it --rm  \
  -v "${WORKSPACE}:/eirinistaging" \
  eirini/staging-integration /eirinistaging/scripts/integration-test.sh
