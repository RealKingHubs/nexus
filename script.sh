#!/bin/bash

# Clean project module requirements caches
go mod tidy

# Execute a native test validation build profile string
go build -o bin/nexus main.go

./bin/nexus --help
