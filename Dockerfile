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

FROM alpine:3.12 as builder 

RUN apk add --no-cache --virtual .tools \
  bzip2 curl git automake libtool autoconf make sed file perl openrc openssl

# ATS
RUN apk add --no-cache --virtual .ats-build-deps \
  build-base openssl-dev tcl-dev pcre-dev zlib-dev \
  libexecinfo-dev linux-headers libunwind-dev \
  brotli-dev jansson-dev luajit-dev readline-dev geoip-dev 

RUN curl -L https://www-us.apache.org/dist/trafficserver/trafficserver-8.0.8.tar.bz2 | bzip2 -dc | tar xf - \
  && cd trafficserver-8.0.8/ \
  && autoreconf -if \
  && ./configure --enable-debug=yes \
  && make \
  && make install

COPY ["./plugin.config", "/usr/local/etc/trafficserver/plugin.config"]
COPY ["./records.config", "/usr/local/etc/trafficserver/records.config"]
COPY ["./logging.yaml", "/usr/local/etc/trafficserver/logging.yaml"]

# enable traffic.out for alpine/gentoo
RUN sed -i "s/TM_DAEMON_ARGS=\"\"/TM_DAEMON_ARGS=\" --bind_stdout \/usr\/local\/var\/log\/trafficserver\/traffic.out --bind_stderr \/usr\/local\/var\/log\/trafficserver\/traffic.out \"/" /usr/local/bin/trafficserver
RUN sed -i "s/TS_DAEMON_ARGS=\"\"/TS_DAEMON_ARGS=\" --bind_stdout \/usr\/local\/var\/log\/trafficserver\/traffic.out --bind_stderr \/usr\/local\/var\/log\/trafficserver\/traffic.out \"/" /usr/local/bin/trafficserver

# Installing lua 5.1.4 
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
  && make install-unix

# redis.lua
RUN wget https://github.com/nrk/redis-lua/archive/v2.0.4.tar.gz \
  && tar zxf v2.0.4.tar.gz \
  && cp redis-lua-2.0.4/src/redis.lua /usr/local/share/lua/5.1/redis.lua

# ingress-ats
RUN apk add --no-cache --virtual .ingress-build-deps \
  bash gcc musl-dev openssl go

# Installing Golang https://github.com/CentOS/CentOS-Dockerfiles/blob/master/golang/centos7/Dockerfile
RUN wget https://dl.google.com/go/go1.12.8.src.tar.gz \
    && tar -C /usr/local -xzf go1.12.8.src.tar.gz && cd /usr/local/go/src/ && ./make.bash
ENV PATH=${PATH}:/usr/local/go/bin
ENV GOPATH="/usr/local/go/bin"

# ----------------------- Copy over Project Code to Go path ------------------------
RUN mkdir -p /usr/local/go/bin/src/ingress-ats 

COPY ["./main/", "$GOPATH/src/ingress-ats/main"]
COPY ["./proxy/", "$GOPATH/src/ingress-ats/proxy"]
COPY ["./namespace/", "$GOPATH/src/ingress-ats/namespace"]
COPY ["./endpoint/", "$GOPATH/src/ingress-ats/endpoint"]
COPY ["./types/", "$GOPATH/src/ingress-ats/types"]
COPY ["./util/", "$GOPATH/src/ingress-ats/util"]
COPY ["./watcher/", "$GOPATH/src/ingress-ats/watcher"]
COPY ["./pluginats/", "$GOPATH/src/ingress-ats/pluginats"]
COPY ["./redis/", "$GOPATH/src/ingress-ats/redis"]
COPY ["./go.mod", "$GOPATH/src/ingress-ats/go.mod"]

# Building Project Main
WORKDIR /usr/local/go/bin/src/ingress-ats
ENV GO111MODULE=on
RUN go build -o ingress_ats main/main.go 

# redis conf 
COPY ["./redis.conf", "/usr/local/etc/redis.conf"]

# entry.sh + other scripts
COPY ["./tls-config.sh", "/usr/local/bin/tls-config.sh"]
COPY ["./tls-reload.sh", "/usr/local/bin/tls-reload.sh"]
COPY ["./entry.sh", "/usr/local/bin/entry.sh"]
WORKDIR /usr/local/bin/
RUN chmod 755 tls-config.sh
RUN chmod 755 tls-reload.sh
RUN chmod 755 entry.sh

FROM alpine:3.12

COPY --from=builder /usr/local /usr/local

# essential library  
RUN apk add -U \
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
    logrotate

# redis
RUN mkdir -p /var/run/redis/ \
  && touch /var/run/redis/redis.sock \
  && mkdir -p /var/log/redis

# symlink for luajit
RUN ln -sf /usr/lib/libluajit-5.1.so.2.1.0 /usr/lib/libluajit-5.1.so

# set up ingress log location
RUN mkdir -p /usr/local/var/log/ingress/
COPY ["./logrotate.ingress", "/etc/logrotate.d/ingress"]

ENTRYPOINT ["/usr/local/bin/entry.sh"]
