#!/usr/bin/env bash
ginkgo -mod=vendor -race -randomizeAllSpecs -randomizeSuites -r "$@" integration
