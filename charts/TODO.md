# TODO for Helm support
Helm support for ATS Ingress Controller is still under development and can only be used locally after building the following docker images:
- ats_alpine
- tsexporter

After building the above images, do the following to install ATS Ingress using Helm:
1. `$ kubectl create namespace ats-ingress`
    - Create the namespace where the ingress controller will be installed
2. `$ helm install charts/ats-ingress --generate-name -n ats-ingress`
    - Use helm install to install the chart specifying the namespace created in step one

To install the ingress controller with TLS support, create a file named `override.yaml` which contains the following two values:
```yaml
tls:
    crt: <TLS certificate>
    key: <TLS key>
```
and then:
`$ helm install -f override.yaml charts/ats-ingress --generate-name -n ats-ingress`

## TODO for enabling Helm
- [ ] Upload ats_alpine docker image to a public repository and replace `image.repository` value in values.yaml
- [ ] Upload trafficserver-exporter docker image to a public repository and make the corresponding changes in `.Values.ats.exporter.image.repository`
- [ ] Hosting the helm chart on a public domain.

### Hosting the helm chart

From the [chart repository guide](https://helm.sh/docs/topics/chart_repository/):
> A chart repository is an HTTP server that houses an `index.yaml` file and optionally some packaged charts. 

This can be done in two ways, we can either use our own web server or a cloud storage option like Google Cloud Storage bucket, Amazon S3 bucket, or Github Pages. The chart repository would consist of our packaged chart, a provenance file and a special file called index.yaml which contains an index of all of the charts in the repository. Read the [chart repository guide](https://helm.sh/docs/topics/chart_repository/) for an in-depth explanation of what these are. 

The chart is located at `charts/ats-ingress` and the `index.yaml` file can be generated using the command `$ helm repo index`. The [chart repository guide](https://helm.sh/docs/topics/chart_repository/) details everything that needs to be done.

Helm has created a tool that can easily help us turn out Github repo self-host our own charts using Github Pages. Read more about the tool here. [Chart Releaser](https://github.com/helm/chart-releaser). Chart releaser can be also be used to generate the necessary files and host the same files on any another cloud storage solution if Github Pages is not an option.