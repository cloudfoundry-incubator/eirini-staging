#!/usr/bin/env bash
ginkgo -p --nodes=4 -mod=vendor -race -randomizeAllSpecs -randomizeSuites -r "$@" integration
