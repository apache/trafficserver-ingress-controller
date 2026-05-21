# AGENTS.md

This file provides guidance to coding agents working with this repository.

## Project Overview

Apache Traffic Server Ingress Controller is a Kubernetes ingress controller for Apache Traffic Server (ATS). It watches `Ingress` / `Service` / `Endpoints` resources and translates them into ATS `remap.config` / `plugin.config` configuration; an admission webhook validates Ingress resources before the API server admits them.

## Security model

For security scans, vulnerability triage, security reviews, and any change touching the admission webhook, ingress-to-`remap.config` translation, RBAC, or how Kubernetes API objects cross the trust boundary, read [`SECURITY.md`](./SECURITY.md) first. The repo-specific surfaces (admission webhook, ingress routing) are documented there in addition to the umbrella model at [apache/trafficserver/SECURITY.md](https://github.com/apache/trafficserver/blob/master/SECURITY.md).
