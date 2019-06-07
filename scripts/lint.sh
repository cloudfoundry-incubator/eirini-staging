#!/usr/bin/env bash
golangci-lint run --exclude "weak cryptographic primitive" --exclude "should not be capitalized" -v
