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

# Helm support
This is the ats-ingress chart repository for Helm V3. 
It contains chart for ats-ingress, which contains pods for 
- Apache Traffic Server + Ingress Controller
- fluentd v1.6 
- trafficserver_exporter v0.3.3

## To build new version of the helm chart
1. Update version in ats-ingress/Chart.yaml
2. `$ helm package ats-ingress`
3. `$ helm repo index . --url https://apache.github.com/trafficserver-ingress-controller`
4. Commit and push the changes

## To install from git source
1. `$ kubectl create namespace ats-helm`
2. `$ kubectl create secret tls tls-secret --key tls.key --cert tls.crt -n ats-helm --dry-run=client -o yaml | kubectl apply -f -`
3. `$ helm install charts/ats-ingress --generate-name -n ats-helm`

## To install from repo
1. `$ kubectl create namespace ats-helm`
2. `$ kubectl create secret tls tls-secret --key tls.key --cert tls.crt -n ats-helm --dry-run=client -o yaml | kubectl apply -f -`
3. `$ helm repo add ats-ingress https://apache.github.io/trafficserver-ingress-controller`
4. `$ helm repo update`
5. `$ helm install ats-ingress/ats-ingress --generate-name -n ats-helm` 

