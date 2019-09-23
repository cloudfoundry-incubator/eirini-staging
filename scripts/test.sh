#!/usr/bin/env bash
export GO111MODULE=on
ginkgo -mod=vendor -race -p -randomizeAllSpecs -randomizeSuites -r -skipPackage integration "$@"
