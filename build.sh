#!/bin/bash

mkdir -p ./build

# Build
go generate
CGO_ENABLED=0 GOOS=linux go build -o ./build/home-api
# CGO_ENABLED=0 GOOS=windows go build -o ./build/home-api.exe

cd ./build

# Zip
zip -r ./home-api.zip ./*

# Notes
# -- sudo apt install libexiv2-dev

# OR
# Download https://github.com/Exiv2/exiv2/releases/tag/v0.27.2
# Run:
# mkdir build && cd build
# cmake .. -DCMAKE_BUILD_TYPE=Release
# cmake --build .
# sudo make install
# sudo ldconfig
# export PKG_CONFIG_PATH=/usr/local/lib64/pkgconfig
