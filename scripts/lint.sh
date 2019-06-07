#!/usr/bin/env bash
golangci-lint run --exclude "cyclomatic complexity" --exclude "weak cryptographic primitive" --exclude "should not be capitalized" -v
