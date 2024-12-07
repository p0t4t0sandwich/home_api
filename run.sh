#!/usr/bin/bash

./build.sh

cd data
docker compose down

docker compose up
