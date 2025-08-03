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

FROM alpine:3.20.7 as builder

RUN apk add --no-cache --virtual .tools \
  bzip2 curl nghttp2-libs git automake libtool autoconf make sed file perl openrc openssl

# ATS dependencies
RUN apk add --no-cache --virtual .ats-build-deps \
  build-base openssl-dev tcl-dev pcre-dev zlib-dev \
  linux-headers libunwind-dev curl-dev \
  brotli-dev jansson-dev readline-dev geoip-dev libxml2-dev

RUN apk add --no-cache --virtual .ats-extra-build-deps --repository https://dl-cdn.alpinelinux.org/alpine/edge/community hwloc-dev

RUN apk add --no-cache --repository=https://dl-cdn.alpinelinux.org/alpine/v3.16/main libexecinfo-dev

RUN apk add --no-cache --repository=https://dl-cdn.alpinelinux.org/alpine/v3.19/main luajit-dev=2.1_p20230410-r3

RUN addgroup -Sg 1000 ats

RUN adduser -S -D -H -u 1000 -h /tmp -s /sbin/nologin -G ats -g ats ats

# download and build ATS
# patch 2 files due to pthread in musl vs glibc - see https://github.com/apache/trafficserver/pull/7611/files
RUN curl -L https://downloads.apache.org/trafficserver/trafficserver-9.2.11.tar.bz2 | bzip2 -dc | tar xf - \
  && cd trafficserver-9.2.11/ \
  && sed -i "s/PTHREAD_RWLOCK_WRITER_NONRECURSIVE_INITIALIZER_NP/PTHREAD_RWLOCK_INITIALIZER/" include/tscore/ink_rwlock.h \
  && sed -i "s/PTHREAD_RWLOCK_WRITER_NONRECURSIVE_INITIALIZER_NP/PTHREAD_RWLOCK_INITIALIZER/" include/tscpp/util/TsSharedMutex.h \
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

# luasocket
RUN wget https://github.com/lunarmodules/luasocket/archive/refs/tags/v3.0.0.tar.gz \
  && tar zxf v3.0.0.tar.gz \
  && cd luasocket-3.0.0 \
  && sed -i "s/LDFLAGS_linux=-O -shared -fpic -o/LDFLAGS_linux=-O -shared -fpic -L\/usr\/lib -lluajit-5.1 -o/" src/makefile \
  && ln -sf /usr/lib/libluajit-5.1.so.2.1.0 /usr/lib/libluajit-5.1.so \
  && mkdir -p /usr/include/lua \
  && ln -sf /usr/include/luajit-2.1 /usr/include/lua/5.1 \
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
COPY GO_VERSION /
RUN go_version=$(cat /GO_VERSION) \
    && wget https://dl.google.com/go/go${go_version}.linux-amd64.tar.gz \
    && rm -rf /opt/ats/go && tar -C /opt/ats -xzf go${go_version}.linux-amd64.tar.gz
ENV PATH=${PATH}:/opt/ats/go/bin
ENV GOPATH="/opt/ats/go/bin"

# ----------------------- Copy over Project Code to Go path ------------------------
RUN mkdir -p /opt/ats/go/bin/src/github.com/apache/trafficserver-ingress-controller 

COPY ["./main/", "$GOPATH/src/github.com/apache/trafficserver-ingress-controller/main"]
COPY ["./proxy/", "$GOPATH/src/github.com/apache/trafficserver-ingress-controller/proxy"]
COPY ["./namespace/", "$GOPATH/src/github.com/apache/trafficserver-ingress-controller/namespace"]
COPY ["./endpoint/", "$GOPATH/src/github.com/apache/trafficserver-ingress-controller/endpoint"]
COPY ["./util/", "$GOPATH/src/github.com/apache/trafficserver-ingress-controller/util"]
COPY ["./watcher/", "$GOPATH/src/github.com/apache/trafficserver-ingress-controller/watcher"]
COPY ["./pluginats/", "/opt/ats/var/pluginats"]
COPY ["./redis/", "$GOPATH/src/github.com/apache/trafficserver-ingress-controller/redis"]
COPY ["./go.mod", "$GOPATH/src/github.com/apache/trafficserver-ingress-controller/go.mod"]
COPY ["./go.sum", "$GOPATH/src/github.com/apache/trafficserver-ingress-controller/go.sum"]

# Building Project Main
WORKDIR /opt/ats/go/bin/src/github.com/apache/trafficserver-ingress-controller
ENV GO111MODULE=on
RUN /opt/ats/go/bin/go build -o /opt/ats/bin/ingress_ats main/main.go 
RUN /opt/ats/go/bin/go clean -modcache
RUN rm -rf /opt/ats/go

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

FROM alpine:3.20.7

# essential library  
RUN apk add --no-cache -U \
    bash \
    build-base \
    curl \
    nghttp2-libs \
    ca-certificates \
    pcre \
    zlib \
    openssl \
    brotli \
    jansson \
    libunwind \ 
    readline \
    geoip \
    redis \
    tcl \
    openrc \
    inotify-tools \
    cpulimit \
    libxml2

RUN apk add --no-cache -U --repository https://dl-cdn.alpinelinux.org/alpine/edge/community hwloc

RUN apk add --no-cache --repository=https://dl-cdn.alpinelinux.org/alpine/v3.16/main libexecinfo

RUN apk add --no-cache --repository=https://dl-cdn.alpinelinux.org/alpine/v3.19/main luajit=2.1_p20230410-r3

# symlink for luajit
RUN ln -sf /usr/lib/libluajit-5.1.so.2.1.0 /usr/lib/libluajit-5.1.so

# create ats user/group
RUN addgroup -Sg 1000 ats

RUN adduser -S -D -H -u 1000 -h /tmp -s /sbin/nologin -G ats -g ats ats

COPY --from=builder --chown=ats:ats /opt/ats /opt/ats

USER ats

ENTRYPOINT ["/opt/ats/bin/entry.sh"]
