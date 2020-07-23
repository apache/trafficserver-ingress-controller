#!/bin/bash -e

# build
REPONAME="ats-ingress"
TAG="latest"
TARGET=atsingress/${REPONAME}:${TAG}

echo "${DOCKERHUB_TOKEN}" | docker login -u "${DOCKERHUB_USER}" --password-stdin
docker build -t ${TARGET} .
docker push ${TARGET}