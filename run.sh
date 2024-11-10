#!/usr/bin/bash

# Kill any process running on port 9080
fuser -k 9080/tcp

go generate
go run ./*.go
