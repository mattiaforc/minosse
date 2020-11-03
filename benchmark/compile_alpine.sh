#!/bin/sh

# mattiaforc 2020

CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o minosse .