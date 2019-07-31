#!/usr/bin/env bash
golangci-lint run --deadline 2m --exclude "cyclomatic complexity" --exclude "weak cryptographic primitive" --exclude "should not be capitalized" -v
