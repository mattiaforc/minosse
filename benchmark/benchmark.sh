#!/bin/sh

# mattiaforc 2020

# linux -> --add-host=host.docker.internal:host-gateway
docker run -i loadimpact/k6 run - <benchmark.js