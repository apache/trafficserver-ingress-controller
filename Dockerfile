# 
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

FROM alpine:3.12.7 as builder 

RUN apk add --no-cache --virtual .tools \
  bzip2 curl git automake libtool autoconf make sed file perl openrc openssl

# ATS dependencies
RUN apk add --no-cache --virtual .ats-build-deps \
  build-base openssl-dev tcl-dev pcre-dev zlib-dev \
  libexecinfo-dev linux-headers libunwind-dev \
  brotli-dev jansson-dev luajit-dev readline-dev geoip-dev 

RUN apk add --no-cache --virtual .ats-extra-build-deps --repository https://dl-cdn.alpinelinux.org/alpine/edge/community hwloc-dev

RUN addgroup -Sg 1000 ats

RUN adduser -S -D -H -u 1000 -h /tmp -s /sbin/nologin -G ats -g ats ats

# download and build ATS
RUN curl -L https://downloads.apache.org/trafficserver/trafficserver-9.0.0.tar.bz2 | bzip2 -dc | tar xf - \
  && cd trafficserver-9.0.0/ \
  && autoreconf -if \
  && ./configure --enable-debug=yes --prefix=/opt/ats --with-user=ats \
  && make \
  && make install

COPY ["./config/plugin.config", "/opt/ats/etc/trafficserver/plugin.config"]
COPY ["./config/healthchecks.config", "/opt/ats/etc/trafficserver/healthchecks.config"]
COPY ["./config/records.config", "/opt/ats/etc/trafficserver/records.config"]
COPY ["./config/logging.yaml", "/opt/ats/etc/trafficserver/logging.yaml"]

# enable traffic.out for alpine/gentoo
RUN sed -i "s/TM_DAEMON_ARGS=\"\"/TM_DAEMON_ARGS=\" --bind_stdout \/opt\/ats\/var\/log\/trafficserver\/traffic.out --bind_stderr \/opt\/ats\/var\/log\/trafficserver\/traffic.out \"/" /opt/ats/bin/trafficserver
RUN sed -i "s/TS_DAEMON_ARGS=\"\"/TS_DAEMON_ARGS=\" --bind_stdout \/opt\/ats\/var\/log\/trafficserver\/traffic.out --bind_stderr \/opt\/ats\/var\/log\/trafficserver\/traffic.out \"/" /opt/ats/bin/trafficserver

# Installing lua 5.1.4 and provide header files to compile luasocket 
RUN curl -R -O http://www.lua.org/ftp/lua-5.1.4.tar.gz \
    && tar zxf lua-5.1.4.tar.gz \
    && cd lua-5.1.4 \
    && make linux test \
    && make linux install

# luasocket
RUN wget https://github.com/diegonehab/luasocket/archive/v3.0-rc1.tar.gz \
  && tar zxf v3.0-rc1.tar.gz \
  && cd luasocket-3.0-rc1 \
  && sed -i "s/LDFLAGS_linux=-O -shared -fpic -o/LDFLAGS_linux=-O -shared -fpic -L\/usr\/lib -lluajit-5.1 -o/" src/makefile \
  && ln -sf /usr/lib/libluajit-5.1.so.2.1.0 /usr/lib/libluajit-5.1.so \
  && make \
  && make install-unix prefix=/opt/ats

# redis.lua
RUN wget https://github.com/nrk/redis-lua/archive/v2.0.4.tar.gz \
  && tar zxf v2.0.4.tar.gz \
  && cp redis-lua-2.0.4/src/redis.lua /opt/ats/share/lua/5.1/redis.lua

# ingress-ats
RUN apk add --no-cache --virtual .ingress-build-deps \
  bash gcc musl-dev openssl go

# Installing Golang https://github.com/CentOS/CentOS-Dockerfiles/blob/master/golang/centos7/Dockerfile
RUN wget https://dl.google.com/go/go1.15.11.src.tar.gz \
    && tar -C /opt/ats -xzf go1.15.11.src.tar.gz && cd /opt/ats/go/src/ && ./make.bash
ENV PATH=${PATH}:/opt/ats/go/bin
ENV GOPATH="/opt/ats/go/bin"

# ----------------------- Copy over Project Code to Go path ------------------------
RUN mkdir -p /opt/ats/go/bin/src/ingress-ats 

COPY ["./main/", "$GOPATH/src/ingress-ats/main"]
COPY ["./proxy/", "$GOPATH/src/ingress-ats/proxy"]
COPY ["./namespace/", "$GOPATH/src/ingress-ats/namespace"]
COPY ["./endpoint/", "$GOPATH/src/ingress-ats/endpoint"]
COPY ["./util/", "$GOPATH/src/ingress-ats/util"]
COPY ["./watcher/", "$GOPATH/src/ingress-ats/watcher"]
COPY ["./pluginats/", "$GOPATH/src/ingress-ats/pluginats"]
COPY ["./redis/", "$GOPATH/src/ingress-ats/redis"]
COPY ["./go.mod", "$GOPATH/src/ingress-ats/go.mod"]

# Building Project Main
WORKDIR /opt/ats/go/bin/src/ingress-ats
ENV GO111MODULE=on
RUN go build -o ingress_ats main/main.go 

# redis conf 
COPY ["./config/redis.conf", "/opt/ats/etc/redis.conf"]

# entry.sh + other scripts
COPY ["./bin/tls-config.sh", "/opt/ats/bin/tls-config.sh"]
COPY ["./bin/tls-reload.sh", "/opt/ats/bin/tls-reload.sh"]
COPY ["./bin/records-config.sh", "/opt/ats/bin/records-config.sh"]
COPY ["./bin/entry.sh", "/opt/ats/bin/entry.sh"]
WORKDIR /opt/ats/bin/
RUN chmod 755 tls-config.sh
RUN chmod 755 tls-reload.sh
RUN chmod 755 records-config.sh
RUN chmod 755 entry.sh

# redis
RUN mkdir -p /opt/ats/var/run/redis/ \
  && touch /opt/ats/var/run/redis/redis.sock \
  && mkdir -p /opt/ats/var/log/redis

# set up ingress log location
RUN mkdir -p /opt/ats/var/log/ingress/
RUN mkdir -p /usr/local/etc/

FROM alpine:3.12.7

# essential library  
RUN apk add --no-cache -U \
    bash \
    build-base \
    curl ca-certificates \
    pcre \
    zlib \
    openssl \
    brotli \
    jansson \
    luajit \
    libunwind \ 
    readline \
    geoip \
    libexecinfo \
    redis \
    tcl \
    openrc \
    inotify-tools \
    cpulimit \
    protobuf-dev \
    cmake \
    git openssl-dev curl curl-dev libcurl libressl-dev 

RUN apk add --no-cache -U --repository https://dl-cdn.alpinelinux.org/alpine/edge/community hwloc

# symlink for luajit
RUN ln -sf /usr/lib/libluajit-5.1.so.2.1.0 /usr/lib/libluajit-5.1.so

# setup_thrift.sh
RUN mkdir -p /tmp
COPY ["./opentelemetry-tools/setup_thrift.sh", "/tmp/setup_thrift.sh"]
RUN chmod 775 /tmp/setup_thrift.sh && ./tmp/setup_thrift.sh && rm -rf /tmp/setup_thrift.sh

# nlohmann-json: JSON for modern C++
COPY ["./opentelemetry-tools/json-3.9.1.tar.gz", "/tmp/json-3.9.1.tar.gz"]
RUN cd /tmp && tar zxf json-3.9.1.tar.gz \
 && cd json-3.9.1 \
 && mkdir build && cd build\
 && cmake .. \
 && make \
 && make install

# jinja2cpp: Jinja2С++
RUN cd /tmp && git clone https://github.com/flexferrum/Jinja2Cpp.git \
 && cd Jinja2Cpp \
 && mkdir build \
 && cd build \
 && cmake .. -DCMAKE_INSTALL_PREFIX=../install \
 && cmake --build . --target all \
 && cmake --build . --target install

# opentelementry-cpp
# https://github.com/open-telemetry/opentelemetry-cpp/blob/main/INSTALL.md
# -- Change some setting in opentelemetry-cpp-1.0.0-rc2 (Jaeger)
COPY ["./opentelemetry-tools/opentelemetry-cpp-1.0.0-rc3.tar.gz", "/opentelemetry-cpp-1.0.0-rc3.tar.gz"]
RUN tar zxf opentelemetry-cpp-1.0.0-rc3.tar.gz && cd opentelemetry-cpp-1.0.0-rc3 \
 && mkdir build \
 && cd build \
 && cmake .. -DBUILD_TESTING=OFF -DWITH_JAEGER=ON -DWITH_OTLP=OFF \
 && cmake --build . --target all \
 && cmake --install . --config Debug --prefix /usr/local/ \
 && chmod 777 -R /opentelemetry-cpp-1.0.0-rc3 \
 && rm -rf opentelemetry-cpp-1.0.0-rc3.tar.gz

# remove pkg to save memorty
RUN apk del git boost boost-dev libevent libevent-dev ninja openssl-dev curl curl-dev libcurl libressl-dev 

# create ats user/group
RUN addgroup -Sg 1000 ats

RUN adduser -S -D -H -u 1000 -h /tmp -s /sbin/nologin -G ats -g ats ats

COPY --from=builder --chown=ats:ats /opt/ats /opt/ats

USER ats

ENTRYPOINT ["/opt/ats/bin/entry.sh"]

