#!/bin/bash

docker build -t zhoushoujian/go-net-monitoring:latest ./ && \
docker-compose --profile monitoring down && \
docker-compose --profile monitoring up