#!/bin/bash

set -e
export DEBIAN_FRONTEND=noninteractive
export THRIFT_VERSION=0.14.1

if ! type cmake > /dev/null; then
    #cmake not installed, exiting
    exit 1
fi
export BUILD_DIR=/tmp/
export INSTALL_DIR=/usr/local/

apk update && apk add boost boost-dev libevent libevent-dev ninja

pushd $BUILD_DIR
wget https://github.com/apache/thrift/archive/refs/tags/v${THRIFT_VERSION}.tar.gz
tar -zxvf v${THRIFT_VERSION}.tar.gz
cd thrift-${THRIFT_VERSION}
mkdir -p out
pushd out
cmake -G Ninja .. \
    -DBUILD_COMPILER=OFF \
    -DBUILD_CPP=ON \
    -DBUILD_LIBRARIES=ON \
    -DBUILD_NODEJS=OFF \
    -DBUILD_PYTHON=OFF \
    -DBUILD_JAVASCRIPT=OFF \
    -DBUILD_C_GLIB=OFF \
    -DBUILD_JAVA=OFF \
    -DBUILD_TESTING=OFF \
    -DBUILD_TUTORIALS=OFF \
    ..

ninja -j $(nproc)
ninja install
apk del boost boost-dev libevent libevent-dev ninja

popd
popd
