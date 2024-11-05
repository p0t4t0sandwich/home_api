#!/bin/bash

mkdir -p ./build

# Build
go generate
CGO_ENABLED=0 GOOS=linux go build -o ./build/home-api
# CGO_ENABLED=0 GOOS=windows go build -o ./build/home-api.exe

cd ./build

# Zip
zip -r ./home-api.zip ./*
