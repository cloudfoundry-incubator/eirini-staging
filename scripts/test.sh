#!/usr/bin/env bash
ginkgo -mod=vendor -race -p -randomizeAllSpecs -randomizeSuites -r -skipPackage integration .
