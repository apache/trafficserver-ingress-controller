#!/bin/bash -e

# build
REPONAME="ats-ingress"
TAG="latest"
#TARGET="quay.io"/${QUAY_USERNAME}/${REPONAME}:${TAG}
TARGET="ats-ingress:latest"
#echo ${TARGET}

#echo "${QUAY_PASSWORD}" | docker login -u "${QUAY_USERNAME}" quay.io --password-stdin
docker build -t ${TARGET} .
#docker push ${TARGET}