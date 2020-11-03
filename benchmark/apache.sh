#!/bin/sh

# mattiaforc 2020

docker run -dit --name apache-bench -p 8080:80 -v "$PWD":/usr/local/apache2/htdocs/ httpd:2.4-alpine