#!/bin/bash

set -e
if ! type cmake > /dev/null; then
    #cmake not installed, exiting
    exit 1
fi
export BUILD_DIR=/tmp/
export INSTALL_DIR=/usr/local/

pushd $BUILD_DIR
wget https://github.com/nlohmann/json/archive/refs/tags/v3.9.1.tar.gz
tar zxf v3.9.1.tar.gz
cd json-3.9.1
mkdir build
cd build
cmake ..
make
make install
cd ..

#CMD ["./currency_converter", "0.0.0.0", "8080", "."]

popd