<!--
    Licensed to the Apache Software Foundation (ASF) under one
    or more contributor license agreements.  See the NOTICE file
    distributed with this work for additional information
    regarding copyright ownership.  The ASF licenses this file
    to you under the Apache License, Version 2.0 (the
    "License"); you may not use this file except in compliance
    with the License.  You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

    Unless required by applicable law or agreed to in writing,
    software distributed under the License is distributed on an
    "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
    KIND, either express or implied.  See the License for the
    specific language governing permissions and limitations
    under the License.
-->

ATS Kubernetes Ingress Controller
=================================
![Test](https://github.com/apache/trafficserver-ingress-controller/actions/workflows/test.yml/badge.svg)
![Build and Integrate](https://github.com/apache/trafficserver-ingress-controller/actions/workflows/build.yml/badge.svg)
[![Go Report
Card](https://goreportcard.com/badge/github.com/apache/trafficserver-ingress-controller)](https://goreportcard.com/report/github.com/apache/trafficserver-ingress-controller)

## Introduction
[Apache Traffic Server (ATS)](https://trafficserver.apache.org/) is a high performance, open-source, caching proxy server that is scalable and configurable. This project uses ATS as a [Kubernetes(K8s)](https://kubernetes.io/) [ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/)

- [Architecture](https://github.com/apache/trafficserver-ingress-controller/blob/master/docs/ARCHITECTURE.md)
- [Tutorial](https://github.com/apache/trafficserver-ingress-controller/blob/master/docs/TUTORIAL.md)
- [Development](https://github.com/apache/trafficserver-ingress-controller/blob/master/docs/DEVELOPMENT.md)

## Dependencies
- Alpine Linux 3.16.9
- Apache Traffic Server 9.2.4
- LuaJIT 2.0.4
- Go (Version can be found in `GO_VERSION` file found at the base of this repository)
- Other Packages
  - luasocket 3.0.0
  - redis-lua 2.0.4
- Tested on Minikube 1.30.1 / Kubernetes 1.27.11 

