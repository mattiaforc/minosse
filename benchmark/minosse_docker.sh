#!/bin/sh

# mattiaforc 2020

docker run -d -i -t --name minosse-bench -p 8080:8080 -v "${PWD%/*}"/:/web minosse