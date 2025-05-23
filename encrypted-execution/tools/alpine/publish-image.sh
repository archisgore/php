#!/bin/sh
# Copyright (c) 2020 Polyverse Corporation

image="ghcr.io/encrypted-execution/ee-${PHP_VERSION}-${BUILD_TYPE}-alpine-fpm"

echo "$(date) Obtaining current git sha for tagging the docker image"
headsha=$(git rev-parse --verify HEAD)


docker build -t $image:alpine-$headsha .
docker push $image:alpine-$headsha

if [[ "$1" == "-p" ]]; then
    echo "Pushing as latest tag..."
    docker tag $image:alpine-$headsha $image:latest
    docker push $image:latest
fi
