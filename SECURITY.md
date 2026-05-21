# Security Policy

This is a project of the [Apache Software Foundation](https://apache.org/) and follows the ASF [vulnerability handling process](https://apache.org/security/#vulnerability-handling).

We strongly encourage folks to report such problems to our private security mailing list first, before disclosing them publicly.

# Reporting a Vulnerability

To report a new vulnerability you have discovered please follow the ASF [vulnerability reporting process](https://apache.org/security/#reporting-a-vulnerability).

# Security Model

The Apache Traffic Server PMC's umbrella security model lives at [apache/trafficserver/SECURITY.md](https://github.com/apache/trafficserver/blob/master/SECURITY.md) and is the canonical reference for the trust boundaries that apply to the C++ core (administrative users trusted, binaries / scripts / configuration trusted, plugins in scope including `plugins/experimental/`).

This repository (`apache/trafficserver-ingress-controller`) adds the Kubernetes ingress integration on top of that core, with two additional trust surfaces that the umbrella model does not cover directly:

- **Kubernetes admission webhook.** The controller exposes a validating admission webhook to the API server. Webhook requests originate from the Kubernetes control plane and are authenticated by the API server's TLS bearer-token bootstrap; webhook bodies (the proposed `Ingress`, `Service`, `Endpoints`, and `ConfigMap` payloads) are treated as adversary-controlled input on the wire — admission decisions and parsing of those payloads must not crash, leak, or allow injection.
- **Ingress routing.** Resolved `Ingress` resources are translated into Traffic Server `remap.config` / `plugin.config` directives that route HTTP traffic to upstream `Service`s. Cluster users with `Ingress` create/update privileges can influence the routing table and therefore the proxy's outbound destinations; the trust boundary is the RBAC scope on those resource kinds in the namespace.

Reports that fall under either of those surfaces — admission-webhook handling, or ingress-to-`remap.config` translation — should be reported via the same ASF reporting process above. The C++ core surface (proxy data plane, plugins, configuration loading) remains governed by the umbrella model linked above.
