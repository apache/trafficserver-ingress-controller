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

## Tutorial
- [Usage](#usage)
  - [ConfigMap](#configmap)
  - [Namespaces for Ingresses](#namespaces-for-ingresses)
  - [Snippet](#snippet)
  - [Ingress Class](#ingress-class)
  - [Customizing Logging and TLS](#customizing-logging-and-tls)
  - [Customizing plugins](#customizing-plugins)
  - [Enabling Controller Debug Log](#enabling-controller-debug-log)
  - [Resync Period of Controller](#resync-period-of-controller)
- [Integrating with Fluentd and Prometheus](#integrating-with-fluentd-and-prometheus)
- [Helm Chart](#helm-chart)

### Usage

Check out the project's github action "Build and Integrate". It has an example of building the docker image for the ingress controller. It also demonstrate the usage of the ingress controller throught integration tests. A Kubernetes cluster is setup with applications and ingress controller for them. 

#### ConfigMap

The above also shows example of configuring Apache Traffic Server [_reloadable_ configurations](https://docs.trafficserver.apache.org/en/9.2.x/admin-guide/files/records.config.en.html#reloadable) using [kubernetes configmap](https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/) resource:

#### Namespaces for Ingresses

You can specifiy the list of namespaces to look for ingress object by providing an environment variable called `INGRESS_NS`. The default is `all`, which tells the controller to look for ingress objects in all namespaces. Alternatively you can provide a comma-separated list of namespaces for the controller to look for ingresses. Similarly you can specifiy a comma-separated list of namespaces to ignore while the controller is looking for ingresses by providing `INGRESS_IGNORE_NS`.

#### Snippet

You can attach [ATS lua script](https://docs.trafficserver.apache.org/en/9.2.x/admin-guide/plugins/lua.en.html) to an ingress object and ATS will execute it for requests matching the routing rules defined in the ingress object. 

#### Ingress Class

You can provide an environment variable called `INGRESS_CLASS` in the deployment to specify the ingress class. The above contains an example commented out in the deployment yaml file. Only ingress object with parameter `ingressClassName` in `spec` section with value equal to the environment variable value will be used by ATS for routing

#### Customizing Logging and TLS

You can specify a different
[logging.yaml](https://docs.trafficserver.apache.org/en/9.2.x/admin-guide/files/logging.yaml.en.html) and [sni.yaml](https://docs.trafficserver.apache.org/en/9.2.x/admin-guide/files/sni.yaml.en.html) by providing environment variable `LOG_CONFIG_FNAME` and `SSL_SERVERNAME_FNAME` respsectively. The new contents of them can be provided through a ConfigMap and loaded to a volume mounted for the ATS container (Example [here](https://kubernetes.io/docs/concepts/storage/volumes/#configmap) ). Similarly certificates needed for the connection between ATS and origin can be provided through a Secret that loaded to a volume mounted for the ATS container as well (Example [here](https://kubernetes.io/docs/concepts/configuration/secret/#using-secrets-as-files-from-a-pod) ). To refresh these certificates we may need to override the entrypoint with our own command and add extra script to watch for changes in those secret in order to reload ATS (Example [here](../bin/tls-reload.sh) ).

#### Customizing Plugins

You can specify extra plugins for [plugin.config](https://docs.trafficserver.apache.org/en/9.2.x/admin-guide/files/plugin.config.en.html) by providing environment variable `EXTRA_PLUGIN_FNAME`. Its contents can be provided through a ConfigMap and loaded to a volume mounted for the ATS container (Example [here](https://kubernetes.io/docs/concepts/storage/volumes/#configmap) ).

#### Enabling Controller Debug Log

You can enable debug for the controller by providing environment variable `INGRESS_DEBUG`.

#### Resync Period of Controller

You can adjust the resync period for the controller by providing environment variable `RESYNC_PRIOD`.

### Integrating with Fluentd and Prometheus

[Fluentd](https://docs.fluentd.org/) can be used to capture the traffic server access logs. [Prometheus](https://prometheus.io/) can be used to capture metrics. Please checkout the below projects for examples.

* https://github.com/gdvalle/trafficserver_exporter
* https://github.com/buraksarp/apache-traffic-server-exporter

### Helm Chart

Helm Chart is provided [here](../charts/ats-ingress/README.md).
