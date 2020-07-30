# TODO for Helm support
Helm support for ATS Ingress Controller is still under development and can only be used locally after building the following docker images:
- ats_alpine
- tsexporter

After building the above images, do the following to install ATS Ingress using Helm:
1. `$ kubectl create namespace trafficserver-test`
    - Create the namespace where the ingress controller will be installed
2. `$ helm install charts/ats-ingress --generate-name -n trafficserver-test`
    - Use helm install to install the chart specifying the namespace created in step one

To install the ingress controller with TLS support, create a file named `override.yaml` which contains the following two values:
```yaml
tls:
    crt: <TLS certificate>
    key: <TLS key>
```
and then:
`$ helm install -f override.yaml charts/ats-ingress --generate-name -n trafficserver-test`

## TODO for enabling Helm
- [ ] Upload ats_alpine docker image to a public repository and replace `image.repository` value in values.yaml
- [ ] Upload trafficserver-exporter docker image to a public repository and make the corresponding changes in `.Values.ats.exporter.name`
- [ ] Host the helm chart on either your own server or a public server so that users don't need to clone the repo to use helm for installation