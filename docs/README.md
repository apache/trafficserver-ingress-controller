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
![Test](https://github.com/apache/trafficserver-ingress-controller/workflows/Test/badge.svg)
![Build and Integrate](https://github.com/apache/trafficserver-ingress-controller/workflows/Build%20and%20Integrate/badge.svg)

## Contents
- [Introduction](#Introduction)
- [Versions of Software Used](#versions-of-software-used)
- [How to use](#how-to-use)
  - [Requirements](#requirements)
  - [Download project](#download-project)
  - [Example Walkthrough](#example-walkthrough)
    - [Proxy](#proxy)
    - [ConfigMap](#configmap)
    - [Snippet](#snippet)
    - [Ingress Class](#ingressclass)
  - [Logging and Monitoring](#logging-and-monitoring)
    - [Fluentd](#fluend)
    - [Prometheus and Grafana](#prometheus-and-grafana)
- [Development](#development)
  - [Develop with Go-Lang in Linux](#develop-with-go-lang-in-linux)
  - [Compilation](#compilation)
  - [Unit Tests](#unit-tests)
  - [Text-Editor](#text-editor)
- [Documentation](#documentation)

## Introduction 
[Apache Traffic Server (ATS)](https://trafficserver.apache.org/) is a high performance, open-source, caching proxy server that is scalable and configurable. This project uses ATS as a [Kubernetes(K8s)](https://kubernetes.io/) [ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/)

![Abstract](docs/images/abstract.png)

From high-level, the ingress controller talks to K8s' API and sets up `watchers` on specific resources that are interesting to ATS. Then, the controller _controls_ ATS by either(1) relay the information from K8s API to ATS, or (2) configure ATS directly.

![How](docs/images/how-it-works.png)

## Versions of Software Used
- Alpine 3.12.1
- Apache Traffic Server 8.1.0
- LuaJIT 2.0.4
- Lua 5.1.4
- Go 1.12.8
- Other Packages
  - luasocket 3.0rc1
  - redis-lua 2.0.4

## How to use

### Requirements
- Docker
- Kubernetes 1.18.10 (Minikube 1.14.2)

To install Docker, visit its [official page](https://docs.docker.com/) and install the correct version for your system.

The walkthrough uses Minikube to guide you through the setup process. Visit the [official Minikube page](https://kubernetes.io/docs/tasks/tools/install-minikube/) to install Minikube. 

### Download project 
If you are cloning this project for development, visit [Setting up Go-Lang](#setting-up-go-lang) for detailed guide on how to develop projects in Go. 

For other purposes, you can use `git clone` or directly download repository to your computer.

### Example Walkthrough
Once you have cloned the project repo and started Docker and Minikube, in the terminal:
1. `$ eval $(minikube docker-env)`
      - To understand why we do this, please read [Use local images by re-using the docker daemon](https://kubernetes.io/docs/setup/learning-environment/minikube/#use-local-images-by-re-using-the-docker-daemon)
2. `$ cd trafficserver-ingress-controller`
3. `$ git submodule update --init`
4. `$ docker build -t ats_alpine .` 
5. `$ docker build -t tsexporter k8s/backend/trafficserver_exporter/` 
6. `$ docker build -t node-app-1 k8s/backend/node-app-1/`    
7. `$ docker build -t node-app-2 k8s/backend/node-app-2/`
8. `$ docker pull fluent/fluentd:v1.6-debian-1`

- At this point, we have created necessary images for our example. Let's talk about what each step does:
  - Step 4 builds an image to create a Docker container that will contain the Apache Traffic Server (ATS) itself, the kubernetes ingress controller, along with other software required for the controller to do its job.
  - Step 5 builds an image for the trafficserver exporter. This exports the ATS statistics over HTTP for Prometheus to read. 
  - Steps 6 and 7 build 2 images that will serve as backends to [kubernetes services](https://kubernetes.io/docs/concepts/services-networking/service/) which we will shortly create

9. `$ kubectl create namespace trafficserver-test`
    - Create a namespace for ATS pod
10. `$ openssl req -x509 -sha256 -nodes -days 365 -newkey rsa:2048 -keyout tls.key -out tls.crt -subj "/CN=atssvc/O=atssvc"`
    - Create a self-signed certificate
11. `$ kubectl create secret tls tls-secret --key tls.key --cert tls.crt -n trafficserver-test --dry-run=client -o yaml | kubectl apply -f -`
    - Create a secret in the namespace just created
12. `$ kubectl apply -f k8s/configmaps/fluentd-configmap.yaml`
    - Create config map for fluentd
13. `$ kubectl apply -f k8s/traffic-server/`
    -  will define a new [kubernetes namespace](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/) named `trafficserver-test` and deploy a single ATS pod to said namespace. The ATS pod is also where the ingress controller lives. 

#### Proxy

The following steps can be executed in any order, thus list numbers are not used.

- `$ kubectl apply -f k8s/apps/`
  - creates namespaces `trafficserver-test-2` and `trafficserver-test-3` if not already exist
  - creates kubernetes services and [deployments](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/) for `appsvc1` and `appsvc2`
  - deploy 2 of each `appsvc1`, and `appsvc2` pods in `trafficserver-test-2`, totally 4 pods in said namespace.
  - similarly, deploy 2 of each `appsvc1`, and `appsvc2` pods in `trafficserver-test-3`, totally 4 pods in this namespace. We now have 8 pods in total for the 2 services we have created and deployed in the 2 namespaces.

- `$ kubectl apply -f k8s/ingresses/`
  - creates namespaces `trafficserver-test-2` and `trafficserver-test-3` if not already exist
  - defines an ingress resource in both `trafficserver-test-2` and `trafficserver-test-3`
  - the ingress resource in `trafficserver-test-2` defines domain name `test.media.com` with `/app1` and `/app2` as its paths
  - both ingress resources define domain name `test.edge.com`; however, `test.edge.com/app1` is only defined in `trafficserver-test-2` and `test.edge.com/app2` is only defined in `trafficserver-test-3`
  - Addtionally, an ingress resources defines HTTPS access for `test.edge.com/app2` in namespace `trafficserver-test-3`

When both steps _above_ have executed at least once, ATS proxying will have started to work. To see proxy in action, we can use [curl](https://linux.die.net/man/1/curl):

1. `$ curl -vH "HOST:test.media.com" "$(minikube ip):30000/app1"`
2. `$ curl -vH "HOST:test.media.com" "$(minikube ip):30000/app2"`
3. `$ curl -vH "HOST:test.edge.com" "$(minikube ip):30000/app1"`
4. `$ curl -vH "HOST:test.edge.com" "$(minikube ip):30000/app2"`
5. `$ curl -vH "HOST:test.edge.com" -k "https://$(minikube ip):30043/app2"`

#### ConfigMap

Below is an example of configuring Apache Traffic Server [_reloadable_ configurations](https://docs.trafficserver.apache.org/en/8.0.x/admin-guide/files/records.config.en.html#reloadable) using [kubernetes configmap](https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/) resource:

- `$ kubectl apply -f k8s/configmaps/ats-configmap.yaml`
  - create a ConfigMap resource in `trafficserver-test` with the annotation `"ats-configmap":"true"` if not already exist
  - configure 3 _reloadable_ ATS configurations:
    1. `proxy.config.output.logfile.rolling_enabled: "1"`
    2. `proxy.config.output.logfile.rolling_interval_sec: "3000"`
    3. `proxy.config.restart.active_client_threshold: "0"`

#### Snippet

You can attach [ATS lua script](https://docs.trafficserver.apache.org/en/8.0.x/admin-guide/plugins/lua.en.html) to an ingress object and ATS will execute it for requests matching the routing rules defined in the ingress object. See an example in annotation section of yaml file [here](k8s/ingresses/ats-ingress-2.yaml) 

#### Ingress Class

You can provide an environment variable called `INGRESS_CLASS` in the deployment to specify the ingress class. Only ingress object with annotation `kubernetes.io/ingress.class` with value equal to the environment variable value will be used by ATS for routing

### Logging and Monitoring

#### Fluentd

This project ships with [Fluentd](https://docs.fluentd.org/) already integrated with the Apache Traffic Server. The configuration file used for the same can be found [here](k8s/configmaps/fluentd-configmap.yaml)

As can be seen from the default configuration file, Fluentd reads the Apache Traffic Server access logs located at `/usr/local/var/log/trafficserver/squid.log` and outputs them to `stdout`. The ouput plugin for Fluentd can be changed to send the logs to any desired location supported by Fluentd including Elasticsearch, Kafka, MongoDB etc. You can read more about output plugins [here](https://docs.fluentd.org/output). 

#### Prometheus and Grafana

Use the following steps to install [Prometheus](https://prometheus.io/docs/prometheus/latest/getting_started/) and [Grafana](https://grafana.com/docs/grafana/latest/) and use them to monitor the Apache Traffic Server statistics.

1. `$ kubectl apply -f k8s/prometheus/ats-stats.yaml`
  - Creates a new service which connects to the ATS pod on port 9122. This service will be used by Prometheus to read the Apache Traffic Server stats.  
2. `$ kubectl apply -f k8s/configmaps/prometheus-configmap.yaml`
  - Creates a new configmap which holds the configuration file for Prometheus. You can modify this configuration file to suit your needs. More about that can be read [here](https://prometheus.io/docs/prometheus/latest/configuration/configuration/)
3. `$ kubectl apply -f k8s/prometheus/prometheus-deployment.yaml`
  - Creates a new deployment consisting of Prometheus and Grafana. Also creates two new services to access prometheus and grafana. 
4. Open `x.x.x.x:30090` in your web browser to access Prometheus where `x.x.x.x` is the IP returned by the command: `$ minikube ip` 
5. Open `x.x.x.x:30030` in your web browser to access the Grafana dashboard where `x.x.x.x` is the IP returned by the command: `$ minikube ip`.
6. The default credentials for logging into Grafana are `admin:admin`
7. Click on `Add your first data source' and select Prometheus under the 'Time series databases category'
8. Set an appropriate name for the data source and enter `localhost:9090` as the URL
  ![New Datasource](docs/images/new-datasource.png)
9. Click on 'Save & Test'. If everything has been installed correctly you should get a notification saying 'Data source is working'
  ![Datasource add success](docs/images/datasource-success.png) 
10. Click on the '+' icon in the left handside column and select 'Dashboard'
11. Click on '+ Add new panel'
12. Enter a PromQL query. For example if you want to add a graph showing the total number of responses over time enter `trafficserver_responses_total` and press Shift + Enter.
  ![New Graph](docs/images/new-graph.png)
13. Click on Apply to add the graph to your dashboard. You can similarly make add more graphs to your dashboard to suit your needs. To learn more about Grafana click [here](https://grafana.com/docs/grafana/latest/)

## Development

### Develop with Go-Lang in Linux
1. Get Go-lang 1.12 from [official site](https://golang.org/dl/)
2. Add `go` command to your PATH: `export PATH=$PATH:/usr/local/go/bin`
3. Define GOPATH: `export GOPATH=$(go env GOPATH)`
4. Add Go workspace to your PATH: `export PATH=$PATH:$(go env GOPATH)/bin`
5. Define Go import Paths
   - Go's import path is different from other languages in that all import paths are _absolute paths_. Due to this reason, it is important to set up your project paths correctly
   - define the base path: `mkdir -p $GOPATH/src/github.com/`
6. Clone the project:
   - `cd $GOPATH/src/github.com/`
   - `git clone <project>`
7. As of Go 1.12 in order to have `go.mod` within Go paths, you must export: `export GO111MODULE=on` to be able to compile locally. 

### Compilation
To compile, type: `go build -o ingress-ats main/main.go`

### Unit Tests
The project includes unit tests for the controller written in Golang and the plugin written in Lua.

To run the Golang unit tests: `go test ./watcher/ && go test ./redis/`

The Lua unit tests use `busted` for testing. `busted` can be installed using `luarocks`:`luarocks install busted`. More information on how to install busted is available [here](https://olivinelabs.com/busted/). 
> :warning: **Note that the project uses Lua 5.1 version**

To run the Lua unit tests: 
- `cd pluginats`
- `busted connect_redis_test.lua` 

### Text-Editor
The repository comes with basic support for both [vscode](https://code.visualstudio.com/) and `vim`. 

If you're using `vscode`:
- `.vscode/settings.json` contains some basic settings for whitespaces and tabs
- `.vscode/extensions.json` contains a few recommended extensions for this project. It is highly recommended to install the [Go extension](https://github.com/Microsoft/vscode-go) since it contains the code lint this project used during development.

If you're using `vim`, a `vimrc` file with basic whitespace and tab configurations is also provided

