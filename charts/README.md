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
Helm support for ATS Ingress Controller is still under development and can only be used locally after building the following docker images:
- ats_alpine
- tsexporter

After building the above images, do the following to install ATS Ingress using Helm:
1. `$ kubectl create namespace ats-ingress`
2. Prepare a self-signed certificate:
`$ openssl req -x509 -sha256 -nodes -days 365 -newkey rsa:2048 -keyout tls.key -out tls.crt -subj "/CN=atssvc/O=atssvc"`
3. Create a file named `override.yaml` which contains the following two values:
```yaml
tls:
    crt: <TLS certificate>
    key: <TLS key>
```
4. `$ helm install -f override.yaml charts/ats-ingress --generate-name -n ats-ingress`

## TODO for enabling Helm
- [ ] Upload ats_alpine docker image to a public repository and make corresponding changes to `image.repository` value in values.yaml
- [ ] Upload trafficserver-exporter docker image to a public repository and make corresponding changes to `ats.exporter.image.repository` value in values.yaml 
- [ ] Hosting the helm chart on github page. Follow the [chart repository guide](https://helm.sh/docs/topics/chart_repository/). 
